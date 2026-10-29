package services_test

import (
	"errors"
	"testing"
	"time"

	"github.com/yourusername/kapufresh-erp/internal/database"
	"github.com/yourusername/kapufresh-erp/internal/models"
	"github.com/yourusername/kapufresh-erp/internal/services"
)

func newTestAuthService(t *testing.T) (*services.AuthService, *models.User) {
	t.Helper()
	db, err := database.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	if err := database.Migrate(db); err != nil {
		t.Fatal(err)
	}
	if err := database.SeedAdmin(db, "admin@example.com", "valid-test-password", "Test Admin"); err != nil {
		t.Fatal(err)
	}
	auth, err := services.NewAuthService(db, "0123456789abcdef0123456789abcdef", 15*time.Minute, 24*time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	var admin models.User
	if err := db.Where("email = ?", "admin@example.com").First(&admin).Error; err != nil {
		t.Fatal(err)
	}
	return auth, &admin
}

func TestRefreshRotatesAndLogoutRevokesToken(t *testing.T) {
	auth, _ := newTestAuthService(t)
	user, original, err := auth.Login("ADMIN@example.com", "valid-test-password")
	if err != nil {
		t.Fatal(err)
	}
	if user.Role != models.RoleAdmin {
		t.Fatalf("role = %q, want admin", user.Role)
	}

	_, rotated, err := auth.Refresh(original.RefreshToken)
	if err != nil {
		t.Fatal(err)
	}
	if rotated.RefreshToken == original.RefreshToken {
		t.Fatal("refresh token was not rotated")
	}
	if _, _, err := auth.Refresh(original.RefreshToken); !errors.Is(err, services.ErrInvalidToken) {
		t.Fatalf("reuse of previous token error = %v, want invalid token", err)
	}
	if err := auth.Logout(user.ID, rotated.RefreshToken); err != nil {
		t.Fatal(err)
	}
	if _, _, err := auth.Refresh(rotated.RefreshToken); !errors.Is(err, services.ErrInvalidToken) {
		t.Fatalf("refresh after logout error = %v, want invalid token", err)
	}
}

func TestRegisterAndProtectFinalAdmin(t *testing.T) {
	auth, admin := newTestAuthService(t)
	user, err := auth.Register("Manager", "manager@example.com", "manager-password-123", models.RoleManager)
	if err != nil {
		t.Fatal(err)
	}
	if user.PasswordHash == "manager-password-123" {
		t.Fatal("password was stored without hashing")
	}
	if _, err := auth.Register("Duplicate", "manager@example.com", "another-password-123", models.RoleStaff); !errors.Is(err, services.ErrConflict) {
		t.Fatalf("duplicate registration error = %v, want conflict", err)
	}
	if err := auth.DeleteUser(admin.ID, admin.ID); !errors.Is(err, services.ErrForbidden) {
		t.Fatalf("self-delete error = %v, want forbidden", err)
	}
	if err := auth.DeleteUser(admin.ID, user.ID); err != nil {
		t.Fatal(err)
	}
	if err := auth.DeleteUser(admin.ID+1, admin.ID); !errors.Is(err, services.ErrForbidden) {
		t.Fatalf("final admin deletion error = %v, want forbidden", err)
	}
}
