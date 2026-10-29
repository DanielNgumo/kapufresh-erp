package dto

import "github.com/yourusername/kapufresh-erp/internal/models"

type RegisterRequest struct {
	Name     string          `json:"name" binding:"required,min=1,max=120"`
	Email    string          `json:"email" binding:"required,email,max=254"`
	Password string          `json:"password" binding:"required,min=12,max=72"`
	Role     models.UserRole `json:"role" binding:"omitempty,oneof=admin manager staff"`
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email,max=254"`
	Password string `json:"password" binding:"required,min=1,max=72"`
}

type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

type UpdateUserRequest struct {
	Name     *string          `json:"name" binding:"omitempty,min=1,max=120"`
	Email    *string          `json:"email" binding:"omitempty,email,max=254"`
	Password *string          `json:"password" binding:"omitempty,min=12,max=72"`
	Role     *models.UserRole `json:"role" binding:"omitempty,oneof=admin manager staff"`
	IsActive *bool            `json:"is_active"`
}

type UserResponse struct {
	ID        uint            `json:"id"`
	Name      string          `json:"name"`
	Email     string          `json:"email"`
	Role      models.UserRole `json:"role"`
	IsActive  bool            `json:"is_active"`
	CreatedAt string          `json:"created_at"`
	UpdatedAt string          `json:"updated_at"`
}

type TokenResponse struct {
	AccessToken  string       `json:"access_token"`
	RefreshToken string       `json:"refresh_token"`
	TokenType    string       `json:"token_type"`
	ExpiresIn    int64        `json:"expires_in"`
	User         UserResponse `json:"user"`
}

type Pagination struct {
	Page       int   `json:"page"`
	PerPage    int   `json:"per_page"`
	Total      int64 `json:"total"`
	TotalPages int   `json:"total_pages"`
}

type UserListResponse struct {
	Users      []UserResponse `json:"users"`
	Pagination Pagination     `json:"pagination"`
}
