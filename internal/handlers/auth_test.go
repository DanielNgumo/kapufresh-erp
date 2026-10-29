package handlers_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/yourusername/kapufresh-erp/internal/database"
	"github.com/yourusername/kapufresh-erp/internal/routes"
	"github.com/yourusername/kapufresh-erp/internal/services"
)

func TestLoginAndMeHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)
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
	if err := database.SeedAdmin(db, "admin@example.com", "handler-test-password", "Test Admin"); err != nil {
		t.Fatal(err)
	}
	auth, err := services.NewAuthService(db, "0123456789abcdef0123456789abcdef", 15*time.Minute, 24*time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	catalog, err := services.NewCatalogService(db)
	if err != nil {
		t.Fatal(err)
	}
	inventory, err := services.NewInventoryService(db)
	if err != nil {
		t.Fatal(err)
	}
	commerce, err := services.NewCommerceService(db, "KES")
	if err != nil {
		t.Fatal(err)
	}
	employees, err := services.NewEmployeeService(db)
	if err != nil {
		t.Fatal(err)
	}
	reports, err := services.NewReportService(db, "KES")
	if err != nil {
		t.Fatal(err)
	}
	router := routes.New(auth, catalog, inventory, commerce, employees, reports, nil)

	loginBody := []byte(`{"email":"admin@example.com","password":"handler-test-password"}`)
	login := httptest.NewRecorder()
	loginRequest := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(loginBody))
	loginRequest.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(login, loginRequest)
	if login.Code != http.StatusOK {
		t.Fatalf("login status = %d, want %d; body: %s", login.Code, http.StatusOK, login.Body.String())
	}

	var response struct {
		Data struct {
			AccessToken string `json:"access_token"`
			User        struct {
				Email string `json:"email"`
			} `json:"user"`
		} `json:"data"`
	}
	if err := json.Unmarshal(login.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response.Data.AccessToken == "" || response.Data.User.Email != "admin@example.com" {
		t.Fatalf("unexpected login response: %+v", response)
	}

	me := httptest.NewRecorder()
	meRequest := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	meRequest.Header.Set("Authorization", "Bearer "+response.Data.AccessToken)
	router.ServeHTTP(me, meRequest)
	if me.Code != http.StatusOK {
		t.Fatalf("me status = %d, want %d; body: %s", me.Code, http.StatusOK, me.Body.String())
	}

	adminUsers := httptest.NewRecorder()
	adminUsersRequest := httptest.NewRequest(http.MethodGet, "/api/v1/users?page=1&per_page=10", nil)
	adminUsersRequest.Header.Set("Authorization", "Bearer "+response.Data.AccessToken)
	router.ServeHTTP(adminUsers, adminUsersRequest)
	if adminUsers.Code != http.StatusOK {
		t.Fatalf("admin users status = %d, want %d; body: %s", adminUsers.Code, http.StatusOK, adminUsers.Body.String())
	}

	register := httptest.NewRecorder()
	registerRequest := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewReader(
		[]byte(`{"name":"Staff User","email":"staff@example.com","password":"handler-staff-password","role":"staff"}`),
	))
	registerRequest.Header.Set("Content-Type", "application/json")
	registerRequest.Header.Set("Authorization", "Bearer "+response.Data.AccessToken)
	router.ServeHTTP(register, registerRequest)
	if register.Code != http.StatusCreated {
		t.Fatalf("register status = %d, want %d; body: %s", register.Code, http.StatusCreated, register.Body.String())
	}

	staffLoginBody := []byte(`{"email":"staff@example.com","password":"handler-staff-password"}`)
	staffLogin := httptest.NewRecorder()
	staffLoginRequest := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(staffLoginBody))
	staffLoginRequest.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(staffLogin, staffLoginRequest)
	if staffLogin.Code != http.StatusOK {
		t.Fatalf("staff login status = %d, want %d; body: %s", staffLogin.Code, http.StatusOK, staffLogin.Body.String())
	}
	var staffResponse struct {
		Data struct {
			AccessToken string `json:"access_token"`
		} `json:"data"`
	}
	if err := json.Unmarshal(staffLogin.Body.Bytes(), &staffResponse); err != nil {
		t.Fatal(err)
	}
	restrictedUsers := httptest.NewRecorder()
	restrictedUsersRequest := httptest.NewRequest(http.MethodGet, "/api/v1/users", nil)
	restrictedUsersRequest.Header.Set("Authorization", "Bearer "+staffResponse.Data.AccessToken)
	router.ServeHTTP(restrictedUsers, restrictedUsersRequest)
	if restrictedUsers.Code != http.StatusForbidden {
		t.Fatalf("staff users status = %d, want %d; body: %s", restrictedUsers.Code, http.StatusForbidden, restrictedUsers.Body.String())
	}

	refreshBody := []byte(`{"refresh_token":"` + extractRefreshToken(t, login.Body.Bytes()) + `"}`)
	refresh := httptest.NewRecorder()
	refreshRequest := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", bytes.NewReader(refreshBody))
	refreshRequest.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(refresh, refreshRequest)
	if refresh.Code != http.StatusOK {
		t.Fatalf("refresh status = %d, want %d; body: %s", refresh.Code, http.StatusOK, refresh.Body.String())
	}
}

func extractRefreshToken(t *testing.T, body []byte) string {
	t.Helper()
	var response struct {
		Data struct {
			RefreshToken string `json:"refresh_token"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &response); err != nil {
		t.Fatal(err)
	}
	return response.Data.RefreshToken
}
