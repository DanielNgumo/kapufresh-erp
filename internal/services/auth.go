package services

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"net/mail"
	"strconv"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/yourusername/kapufresh-erp/internal/models"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

var (
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrInvalidToken       = errors.New("invalid or expired token")
	ErrForbidden          = errors.New("operation is not allowed")
	ErrNotFound           = errors.New("record not found")
	ErrConflict           = errors.New("record already exists")
	ErrInvalidInput       = errors.New("invalid input")
)

const accessTokenIssuer = "kapufresh-erp"

type AuthService struct {
	db              *gorm.DB
	jwtSecret       []byte
	accessTokenTTL  time.Duration
	refreshTokenTTL time.Duration
}

type AccessClaims struct {
	Role models.UserRole `json:"role"`
	jwt.RegisteredClaims
}

type TokenPair struct {
	AccessToken  string
	RefreshToken string
	ExpiresIn    int64
}

type UserPage struct {
	Users      []models.User
	Page       int
	PerPage    int
	Total      int64
	TotalPages int
}

type UserUpdate struct {
	Name     *string
	Email    *string
	Password *string
	Role     *models.UserRole
	IsActive *bool
}

func NewAuthService(db *gorm.DB, secret string, accessTokenTTL, refreshTokenTTL time.Duration) (*AuthService, error) {
	if db == nil {
		return nil, fmt.Errorf("database is required")
	}
	if len(secret) < 32 {
		return nil, fmt.Errorf("JWT secret must be at least 32 bytes")
	}
	if accessTokenTTL <= 0 || refreshTokenTTL <= 0 {
		return nil, fmt.Errorf("token lifetimes must be positive")
	}
	return &AuthService{
		db:              db,
		jwtSecret:       []byte(secret),
		accessTokenTTL:  accessTokenTTL,
		refreshTokenTTL: refreshTokenTTL,
	}, nil
}

func (s *AuthService) Register(name, email, password string, role models.UserRole) (models.User, error) {
	name = strings.TrimSpace(name)
	email, err := normalizeEmail(email)
	if err != nil {
		return models.User{}, err
	}
	if name == "" || len(name) > 120 || len(password) < 12 || len(password) > 72 {
		return models.User{}, ErrInvalidInput
	}
	if role == "" {
		role = models.RoleStaff
	}
	if !validRole(role) {
		return models.User{}, ErrInvalidInput
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return models.User{}, fmt.Errorf("hash password: %w", err)
	}
	user := models.User{
		Name:         name,
		Email:        email,
		PasswordHash: string(hash),
		Role:         role,
		IsActive:     true,
	}
	if err := s.db.Create(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return models.User{}, ErrConflict
		}
		return models.User{}, fmt.Errorf("create user: %w", err)
	}
	return user, nil
}

func (s *AuthService) Login(email, password string) (models.User, TokenPair, error) {
	email, err := normalizeEmail(email)
	if err != nil {
		return models.User{}, TokenPair{}, ErrInvalidCredentials
	}
	var user models.User
	if err := s.db.Where("email = ?", email).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return models.User{}, TokenPair{}, ErrInvalidCredentials
		}
		return models.User{}, TokenPair{}, fmt.Errorf("find user: %w", err)
	}
	if !user.IsActive || bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)) != nil {
		return models.User{}, TokenPair{}, ErrInvalidCredentials
	}
	tokens, err := s.createTokenPair(user)
	if err != nil {
		return models.User{}, TokenPair{}, err
	}
	if err := s.persistRefreshToken(user.ID, tokens.RefreshToken, time.Now().Add(s.refreshTokenTTL)); err != nil {
		return models.User{}, TokenPair{}, err
	}
	return user, tokens, nil
}

func (s *AuthService) Refresh(refreshToken string) (models.User, TokenPair, error) {
	if refreshToken == "" {
		return models.User{}, TokenPair{}, ErrInvalidToken
	}
	newRefreshToken, err := randomRefreshToken()
	if err != nil {
		return models.User{}, TokenPair{}, fmt.Errorf("generate refresh token: %w", err)
	}
	now := time.Now()
	var user models.User
	var pair TokenPair

	err = s.db.Transaction(func(tx *gorm.DB) error {
		var stored models.RefreshToken
		if err := tx.Where("token_hash = ?", hashToken(refreshToken)).First(&stored).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrInvalidToken
			}
			return fmt.Errorf("find refresh token: %w", err)
		}
		if stored.RevokedAt != nil || !stored.ExpiresAt.After(now) {
			return ErrInvalidToken
		}
		if err := tx.First(&user, stored.UserID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrInvalidToken
			}
			return fmt.Errorf("find refresh-token user: %w", err)
		}
		if !user.IsActive {
			return ErrInvalidToken
		}
		pair, err = s.createTokenPairWithRefresh(user, newRefreshToken, now)
		if err != nil {
			return err
		}
		result := tx.Model(&stored).Where("revoked_at IS NULL").Update("revoked_at", now)
		if result.Error != nil {
			return fmt.Errorf("revoke old refresh token: %w", result.Error)
		}
		if result.RowsAffected != 1 {
			return ErrInvalidToken
		}
		record := models.RefreshToken{
			UserID:    user.ID,
			TokenHash: hashToken(newRefreshToken),
			ExpiresAt: now.Add(s.refreshTokenTTL),
		}
		if err := tx.Create(&record).Error; err != nil {
			return fmt.Errorf("store replacement refresh token: %w", err)
		}
		return nil
	})
	if err != nil {
		return models.User{}, TokenPair{}, err
	}
	return user, pair, nil
}

func (s *AuthService) Logout(userID uint, refreshToken string) error {
	if refreshToken == "" {
		return ErrInvalidToken
	}
	var stored models.RefreshToken
	if err := s.db.Where("token_hash = ? AND user_id = ?", hashToken(refreshToken), userID).First(&stored).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrInvalidToken
		}
		return fmt.Errorf("find refresh token: %w", err)
	}
	if stored.RevokedAt == nil {
		now := time.Now()
		if err := s.db.Model(&stored).Update("revoked_at", now).Error; err != nil {
			return fmt.Errorf("revoke refresh token: %w", err)
		}
	}
	return nil
}

func (s *AuthService) ParseAccessToken(raw string) (*AccessClaims, error) {
	claims := &AccessClaims{}
	token, err := jwt.ParseWithClaims(raw, claims, func(token *jwt.Token) (any, error) {
		if token.Method != jwt.SigningMethodHS256 {
			return nil, ErrInvalidToken
		}
		return s.jwtSecret, nil
	}, jwt.WithIssuer(accessTokenIssuer), jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))
	if err != nil || token == nil || !token.Valid || claims.Subject == "" {
		return nil, ErrInvalidToken
	}
	return claims, nil
}

func (s *AuthService) GetUser(id uint) (models.User, error) {
	var user models.User
	if err := s.db.First(&user, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return models.User{}, ErrNotFound
		}
		return models.User{}, fmt.Errorf("find user: %w", err)
	}
	return user, nil
}

func (s *AuthService) ListUsers(page, perPage int, search string, role models.UserRole) (UserPage, error) {
	if page < 1 || perPage < 1 || perPage > 100 || (role != "" && !validRole(role)) {
		return UserPage{}, ErrInvalidInput
	}
	query := s.db.Model(&models.User{})
	if term := strings.TrimSpace(search); term != "" {
		pattern := "%" + term + "%"
		query = query.Where("name LIKE ? OR email LIKE ?", pattern, pattern)
	}
	if role != "" {
		query = query.Where("role = ?", role)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return UserPage{}, fmt.Errorf("count users: %w", err)
	}
	var users []models.User
	if err := query.Order("id ASC").Offset((page - 1) * perPage).Limit(perPage).Find(&users).Error; err != nil {
		return UserPage{}, fmt.Errorf("list users: %w", err)
	}
	pages := int((total + int64(perPage) - 1) / int64(perPage))
	return UserPage{Users: users, Page: page, PerPage: perPage, Total: total, TotalPages: pages}, nil
}

func (s *AuthService) UpdateUser(actorID, userID uint, update UserUpdate) (models.User, error) {
	var user models.User
	err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.First(&user, userID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrNotFound
			}
			return fmt.Errorf("find user: %w", err)
		}
		changes := make(map[string]any)
		if update.Name != nil {
			name := strings.TrimSpace(*update.Name)
			if name == "" || len(name) > 120 {
				return ErrInvalidInput
			}
			changes["name"] = name
		}
		if update.Email != nil {
			email, err := normalizeEmail(*update.Email)
			if err != nil {
				return err
			}
			changes["email"] = email
		}
		if update.Password != nil {
			if len(*update.Password) < 12 || len(*update.Password) > 72 {
				return ErrInvalidInput
			}
			hash, err := bcrypt.GenerateFromPassword([]byte(*update.Password), bcrypt.DefaultCost)
			if err != nil {
				return fmt.Errorf("hash password: %w", err)
			}
			changes["password_hash"] = string(hash)
		}
		if update.Role != nil {
			if !validRole(*update.Role) {
				return ErrInvalidInput
			}
			changes["role"] = *update.Role
		}
		if update.IsActive != nil {
			changes["is_active"] = *update.IsActive
		}
		if len(changes) == 0 {
			return ErrInvalidInput
		}
		if userID == actorID && ((update.Role != nil && *update.Role != models.RoleAdmin) || (update.IsActive != nil && !*update.IsActive)) {
			return ErrForbidden
		}
		wouldRemoveAdmin := user.Role == models.RoleAdmin &&
			((update.Role != nil && *update.Role != models.RoleAdmin) || (update.IsActive != nil && !*update.IsActive))
		if wouldRemoveAdmin {
			var activeAdmins int64
			if err := tx.Model(&models.User{}).Where("role = ? AND is_active = ?", models.RoleAdmin, true).Count(&activeAdmins).Error; err != nil {
				return fmt.Errorf("count active admins: %w", err)
			}
			if activeAdmins <= 1 {
				return ErrForbidden
			}
		}
		if err := tx.Model(&user).Updates(changes).Error; err != nil {
			if errors.Is(err, gorm.ErrDuplicatedKey) {
				return ErrConflict
			}
			return fmt.Errorf("update user: %w", err)
		}
		return tx.First(&user, userID).Error
	})
	return user, err
}

func (s *AuthService) DeleteUser(actorID, userID uint) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		if actorID == userID {
			return ErrForbidden
		}
		var user models.User
		if err := tx.First(&user, userID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrNotFound
			}
			return fmt.Errorf("find user: %w", err)
		}
		if user.Role == models.RoleAdmin && user.IsActive {
			var activeAdmins int64
			if err := tx.Model(&models.User{}).Where("role = ? AND is_active = ?", models.RoleAdmin, true).Count(&activeAdmins).Error; err != nil {
				return fmt.Errorf("count active admins: %w", err)
			}
			if activeAdmins <= 1 {
				return ErrForbidden
			}
		}
		if err := tx.Delete(&user).Error; err != nil {
			return fmt.Errorf("delete user: %w", err)
		}
		return nil
	})
}

func (s *AuthService) createTokenPair(user models.User) (TokenPair, error) {
	refreshToken, err := randomRefreshToken()
	if err != nil {
		return TokenPair{}, fmt.Errorf("generate refresh token: %w", err)
	}
	return s.createTokenPairWithRefresh(user, refreshToken, time.Now())
}

func (s *AuthService) createTokenPairWithRefresh(user models.User, refresh string, now time.Time) (TokenPair, error) {
	expiresAt := now.Add(s.accessTokenTTL)
	claims := AccessClaims{
		Role: user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    accessTokenIssuer,
			Subject:   strconv.FormatUint(uint64(user.ID), 10),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	access, err := token.SignedString(s.jwtSecret)
	if err != nil {
		return TokenPair{}, fmt.Errorf("sign access token: %w", err)
	}
	return TokenPair{AccessToken: access, RefreshToken: refresh, ExpiresIn: int64(s.accessTokenTTL.Seconds())}, nil
}

func (s *AuthService) persistRefreshToken(userID uint, refresh string, expiresAt time.Time) error {
	record := models.RefreshToken{UserID: userID, TokenHash: hashToken(refresh), ExpiresAt: expiresAt}
	if err := s.db.Create(&record).Error; err != nil {
		return fmt.Errorf("store refresh token: %w", err)
	}
	return nil
}

func randomRefreshToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(bytes), nil
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func normalizeEmail(email string) (string, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	address, err := mail.ParseAddress(email)
	if err != nil || address.Address != email {
		return "", ErrInvalidInput
	}
	return email, nil
}

func validRole(role models.UserRole) bool {
	return role == models.RoleAdmin || role == models.RoleManager || role == models.RoleStaff
}
