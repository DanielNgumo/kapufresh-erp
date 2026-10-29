package database

import (
	"fmt"
	"strings"

	"github.com/yourusername/kapufresh-erp/internal/models"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

func SeedAdmin(db *gorm.DB, email, password, name string) error {
	if db == nil {
		return fmt.Errorf("database is required")
	}
	if strings.TrimSpace(email) == "" || password == "" || strings.TrimSpace(name) == "" {
		return fmt.Errorf("admin email, password, and name are required")
	}

	return db.Transaction(func(tx *gorm.DB) error {
		var count int64
		if err := tx.Unscoped().Model(&models.User{}).Count(&count).Error; err != nil {
			return fmt.Errorf("count users: %w", err)
		}
		if count > 0 {
			return nil
		}

		passwordHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			return fmt.Errorf("hash admin password: %w", err)
		}
		user := models.User{
			Name:         strings.TrimSpace(name),
			Email:        strings.ToLower(strings.TrimSpace(email)),
			PasswordHash: string(passwordHash),
			Role:         models.RoleAdmin,
			IsActive:     true,
		}
		if err := tx.Create(&user).Error; err != nil {
			return fmt.Errorf("create admin user: %w", err)
		}
		return nil
	})
}
