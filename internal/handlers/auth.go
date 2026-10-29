package handlers

import (
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/yourusername/kapufresh-erp/internal/dto"
	"github.com/yourusername/kapufresh-erp/internal/middleware"
	"github.com/yourusername/kapufresh-erp/internal/models"
	"github.com/yourusername/kapufresh-erp/internal/services"
	"github.com/yourusername/kapufresh-erp/internal/utils"
)

type AuthHandler struct {
	auth *services.AuthService
}

func NewAuthHandler(auth *services.AuthService) *AuthHandler {
	return &AuthHandler{auth: auth}
}

func (h *AuthHandler) Register(c *gin.Context) {
	var request dto.RegisterRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		utils.RespondError(c, http.StatusBadRequest, "invalid_request", "Request body is invalid")
		return
	}
	user, err := h.auth.Register(request.Name, request.Email, request.Password, request.Role)
	if err != nil {
		h.respondServiceError(c, err)
		return
	}
	utils.RespondData(c, http.StatusCreated, userResponse(user))
}

func (h *AuthHandler) Login(c *gin.Context) {
	var request dto.LoginRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		utils.RespondError(c, http.StatusBadRequest, "invalid_request", "Email and password are required")
		return
	}
	user, tokens, err := h.auth.Login(request.Email, request.Password)
	if err != nil {
		h.respondServiceError(c, err)
		return
	}
	utils.RespondData(c, http.StatusOK, tokenResponse(user, tokens))
}

func (h *AuthHandler) Refresh(c *gin.Context) {
	var request dto.RefreshRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		utils.RespondError(c, http.StatusBadRequest, "invalid_request", "Refresh token is required")
		return
	}
	user, tokens, err := h.auth.Refresh(request.RefreshToken)
	if err != nil {
		h.respondServiceError(c, err)
		return
	}
	utils.RespondData(c, http.StatusOK, tokenResponse(user, tokens))
}

func (h *AuthHandler) Logout(c *gin.Context) {
	userID, ok := middleware.CurrentUserID(c)
	if !ok {
		utils.RespondError(c, http.StatusUnauthorized, "unauthorized", "Authentication is required")
		return
	}
	var request dto.RefreshRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		utils.RespondError(c, http.StatusBadRequest, "invalid_request", "Refresh token is required")
		return
	}
	if err := h.auth.Logout(userID, request.RefreshToken); err != nil {
		h.respondServiceError(c, err)
		return
	}
	utils.RespondData(c, http.StatusOK, gin.H{"message": "Logged out"})
}

func (h *AuthHandler) Me(c *gin.Context) {
	userID, ok := middleware.CurrentUserID(c)
	if !ok {
		utils.RespondError(c, http.StatusUnauthorized, "unauthorized", "Authentication is required")
		return
	}
	user, err := h.auth.GetUser(userID)
	if err != nil {
		h.respondServiceError(c, err)
		return
	}
	utils.RespondData(c, http.StatusOK, userResponse(user))
}

func (h *AuthHandler) ListUsers(c *gin.Context) {
	page, err := queryInt(c, "page", 1)
	if err != nil {
		utils.RespondError(c, http.StatusBadRequest, "invalid_query", "page must be a positive integer")
		return
	}
	perPage, err := queryInt(c, "per_page", 20)
	if err != nil || perPage > 100 {
		utils.RespondError(c, http.StatusBadRequest, "invalid_query", "per_page must be between 1 and 100")
		return
	}
	role := models.UserRole(strings.TrimSpace(c.Query("role")))
	result, err := h.auth.ListUsers(page, perPage, c.Query("search"), role)
	if err != nil {
		h.respondServiceError(c, err)
		return
	}
	users := make([]dto.UserResponse, 0, len(result.Users))
	for _, user := range result.Users {
		users = append(users, userResponse(user))
	}
	utils.RespondData(c, http.StatusOK, dto.UserListResponse{
		Users: users,
		Pagination: dto.Pagination{
			Page: result.Page, PerPage: result.PerPage, Total: result.Total, TotalPages: result.TotalPages,
		},
	})
}

func (h *AuthHandler) GetUser(c *gin.Context) {
	id, ok := pathUserID(c)
	if !ok {
		return
	}
	user, err := h.auth.GetUser(id)
	if err != nil {
		h.respondServiceError(c, err)
		return
	}
	utils.RespondData(c, http.StatusOK, userResponse(user))
}

func (h *AuthHandler) UpdateUser(c *gin.Context) {
	actorID, ok := middleware.CurrentUserID(c)
	if !ok {
		utils.RespondError(c, http.StatusUnauthorized, "unauthorized", "Authentication is required")
		return
	}
	id, ok := pathUserID(c)
	if !ok {
		return
	}
	var request dto.UpdateUserRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		utils.RespondError(c, http.StatusBadRequest, "invalid_request", "Request body is invalid")
		return
	}
	user, err := h.auth.UpdateUser(actorID, id, services.UserUpdate{
		Name: request.Name, Email: request.Email, Password: request.Password,
		Role: request.Role, IsActive: request.IsActive,
	})
	if err != nil {
		h.respondServiceError(c, err)
		return
	}
	utils.RespondData(c, http.StatusOK, userResponse(user))
}

func (h *AuthHandler) DeleteUser(c *gin.Context) {
	actorID, ok := middleware.CurrentUserID(c)
	if !ok {
		utils.RespondError(c, http.StatusUnauthorized, "unauthorized", "Authentication is required")
		return
	}
	id, ok := pathUserID(c)
	if !ok {
		return
	}
	if err := h.auth.DeleteUser(actorID, id); err != nil {
		h.respondServiceError(c, err)
		return
	}
	utils.RespondData(c, http.StatusOK, gin.H{"message": "User deleted"})
}

func (h *AuthHandler) respondServiceError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, services.ErrInvalidCredentials):
		utils.RespondError(c, http.StatusUnauthorized, "invalid_credentials", "Email or password is incorrect")
	case errors.Is(err, services.ErrInvalidToken):
		utils.RespondError(c, http.StatusUnauthorized, "invalid_token", "Token is invalid, expired, or revoked")
	case errors.Is(err, services.ErrForbidden):
		utils.RespondError(c, http.StatusForbidden, "forbidden", "This operation is not allowed")
	case errors.Is(err, services.ErrNotFound):
		utils.RespondError(c, http.StatusNotFound, "not_found", "User was not found")
	case errors.Is(err, services.ErrConflict):
		utils.RespondError(c, http.StatusConflict, "conflict", "A user with that email already exists")
	case errors.Is(err, services.ErrInvalidInput):
		utils.RespondError(c, http.StatusBadRequest, "invalid_request", "Request values are invalid")
	default:
		log.Printf("auth handler error: %v", err)
		utils.RespondError(c, http.StatusInternalServerError, "internal_error", "An unexpected error occurred")
	}
}

func userResponse(user models.User) dto.UserResponse {
	return dto.UserResponse{
		ID: user.ID, Name: user.Name, Email: user.Email, Role: user.Role, IsActive: user.IsActive,
		CreatedAt: user.CreatedAt.Format(time.RFC3339), UpdatedAt: user.UpdatedAt.Format(time.RFC3339),
	}
}

func tokenResponse(user models.User, tokens services.TokenPair) dto.TokenResponse {
	return dto.TokenResponse{
		AccessToken: tokens.AccessToken, RefreshToken: tokens.RefreshToken, TokenType: "Bearer",
		ExpiresIn: tokens.ExpiresIn, User: userResponse(user),
	}
}

func queryInt(c *gin.Context, key string, fallback int) (int, error) {
	value := strings.TrimSpace(c.Query(key))
	if value == "" {
		return fallback, nil
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed < 1 {
		return 0, errInvalidQuery
	}
	return parsed, nil
}

func pathUserID(c *gin.Context) (uint, bool) {
	value, err := strconv.ParseUint(c.Param("id"), 10, strconv.IntSize)
	if err != nil || value == 0 {
		utils.RespondError(c, http.StatusBadRequest, "invalid_id", "User ID must be a positive integer")
		return 0, false
	}
	return uint(value), true
}

var errInvalidQuery = errors.New("invalid query value")
