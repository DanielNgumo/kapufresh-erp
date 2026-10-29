package middleware

import (
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/yourusername/kapufresh-erp/internal/models"
	"github.com/yourusername/kapufresh-erp/internal/services"
	"github.com/yourusername/kapufresh-erp/internal/utils"
	"gorm.io/gorm"
)

const (
	ContextUserID = "user_id"
	ContextRole   = "user_role"
)

func RequireAuth(auth *services.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := strings.TrimSpace(c.GetHeader("Authorization"))
		parts := strings.Fields(header)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			utils.RespondError(c, http.StatusUnauthorized, "unauthorized", "A valid bearer token is required")
			c.Abort()
			return
		}

		claims, err := auth.ParseAccessToken(parts[1])
		if err != nil {
			utils.RespondError(c, http.StatusUnauthorized, "unauthorized", "Access token is invalid or expired")
			c.Abort()
			return
		}
		userID64, err := strconv.ParseUint(claims.Subject, 10, strconv.IntSize)
		if err != nil || userID64 == 0 {
			utils.RespondError(c, http.StatusUnauthorized, "unauthorized", "Access token is invalid or expired")
			c.Abort()
			return
		}
		user, err := auth.GetUser(uint(userID64))
		if errors.Is(err, services.ErrNotFound) || (err == nil && !user.IsActive) {
			utils.RespondError(c, http.StatusUnauthorized, "unauthorized", "Access token is invalid or expired")
			c.Abort()
			return
		}
		if err != nil {
			if !errors.Is(err, gorm.ErrRecordNotFound) {
				log.Printf("authentication middleware error: %v", err)
			}
			utils.RespondError(c, http.StatusInternalServerError, "internal_error", "An unexpected error occurred")
			c.Abort()
			return
		}

		c.Set(ContextUserID, user.ID)
		c.Set(ContextRole, user.Role)
		c.Next()
	}
}

func RequireRoles(allowed ...models.UserRole) gin.HandlerFunc {
	roles := make(map[models.UserRole]struct{}, len(allowed))
	for _, role := range allowed {
		roles[role] = struct{}{}
	}
	return func(c *gin.Context) {
		role, exists := c.Get(ContextRole)
		userRole, ok := role.(models.UserRole)
		if !exists || !ok {
			utils.RespondError(c, http.StatusUnauthorized, "unauthorized", "Authentication is required")
			c.Abort()
			return
		}
		if _, ok := roles[userRole]; !ok {
			utils.RespondError(c, http.StatusForbidden, "forbidden", "You do not have permission to perform this action")
			c.Abort()
			return
		}
		c.Next()
	}
}

func CurrentUserID(c *gin.Context) (uint, bool) {
	value, exists := c.Get(ContextUserID)
	id, ok := value.(uint)
	return id, exists && ok && id > 0
}
