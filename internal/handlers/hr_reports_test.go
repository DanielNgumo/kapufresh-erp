package handlers_test

import (
	"encoding/json"
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/yourusername/kapufresh-erp/internal/database"
	"github.com/yourusername/kapufresh-erp/internal/models"
	"github.com/yourusername/kapufresh-erp/internal/routes"
	"github.com/yourusername/kapufresh-erp/internal/services"
)

func TestEmployeeAndReportRoutes(t *testing.T) {
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
	if err := database.SeedAdmin(db, "hr-reports@example.com", "hr-reports-handler-password", "HR Admin"); err != nil {
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

	unauthorized := doJSONRequest(t, router, http.MethodGet, "/api/v1/employees", "", "")
	if unauthorized.Code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated employee list status = %d, want %d", unauthorized.Code, http.StatusUnauthorized)
	}
	_, tokens, err := auth.Login("hr-reports@example.com", "hr-reports-handler-password")
	if err != nil {
		t.Fatal(err)
	}
	authHeader := "Bearer " + tokens.AccessToken
	if _, err := auth.Register("HR Staff", "hr-staff@example.com", "hr-staff-handler-password", models.RoleStaff); err != nil {
		t.Fatal(err)
	}
	_, staffTokens, err := auth.Login("hr-staff@example.com", "hr-staff-handler-password")
	if err != nil {
		t.Fatal(err)
	}
	staffHeader := "Bearer " + staffTokens.AccessToken

	created := doJSONRequest(t, router, http.MethodPost, "/api/v1/employees",
		`{"first_name":"Nia","last_name":"Wanjiku","email":"nia@example.com","department":"Finance","position":"Analyst"}`, authHeader)
	if created.Code != http.StatusCreated {
		t.Fatalf("create employee status = %d; body: %s", created.Code, created.Body.String())
	}
	var employeeResponse struct {
		Data struct {
			ID     uint   `json:"id"`
			Status string `json:"status"`
		} `json:"data"`
	}
	decodeResponse(t, created.Body.Bytes(), &employeeResponse)
	if employeeResponse.Data.Status != "active" {
		t.Fatalf("new employee status = %q, want active", employeeResponse.Data.Status)
	}
	deniedWrite := doJSONRequest(t, router, http.MethodPost, "/api/v1/employees",
		`{"first_name":"Staff","last_name":"Forbidden"}`, staffHeader)
	if deniedWrite.Code != http.StatusForbidden {
		t.Fatalf("staff employee-create status = %d, want %d", deniedWrite.Code, http.StatusForbidden)
	}
	staffReport := doJSONRequest(t, router, http.MethodGet,
		"/api/v1/reports/stock-valuation", "", staffHeader)
	if staffReport.Code != http.StatusOK {
		t.Fatalf("staff report status = %d, want %d; body: %s", staffReport.Code, http.StatusOK, staffReport.Body.String())
	}

	list := doJSONRequest(t, router, http.MethodGet, "/api/v1/employees?search=nia&department=Finance&status=active", "", authHeader)
	if list.Code != http.StatusOK {
		t.Fatalf("list employees status = %d; body: %s", list.Code, list.Body.String())
	}
	var listResponse struct {
		Data struct {
			Employees []json.RawMessage `json:"employees"`
		} `json:"data"`
	}
	decodeResponse(t, list.Body.Bytes(), &listResponse)
	if len(listResponse.Data.Employees) != 1 {
		t.Fatalf("employee list returned %d results, want 1", len(listResponse.Data.Employees))
	}

	employeePath := "/api/v1/employees/" + strconv.FormatUint(uint64(employeeResponse.Data.ID), 10)
	fetched := doJSONRequest(t, router, http.MethodGet, employeePath, "", authHeader)
	if fetched.Code != http.StatusOK {
		t.Fatalf("get employee status = %d; body: %s", fetched.Code, fetched.Body.String())
	}
	updated := doJSONRequest(t, router, http.MethodPatch, employeePath, `{"position":"Senior Analyst"}`, authHeader)
	if updated.Code != http.StatusOK {
		t.Fatalf("update employee status = %d; body: %s", updated.Code, updated.Body.String())
	}
	deleted := doJSONRequest(t, router, http.MethodDelete, employeePath, "", authHeader)
	if deleted.Code != http.StatusOK {
		t.Fatalf("delete employee status = %d; body: %s", deleted.Code, deleted.Body.String())
	}

	date := time.Now().Format("2006-01-02")
	summary := doJSONRequest(t, router, http.MethodGet,
		"/api/v1/reports/sales-summary?start_date="+date+"&end_date="+date, "", authHeader)
	if summary.Code != http.StatusOK {
		t.Fatalf("sales summary status = %d; body: %s", summary.Code, summary.Body.String())
	}
	topProducts := doJSONRequest(t, router, http.MethodGet,
		"/api/v1/reports/top-products?start_date="+date+"&end_date="+date+"&limit=10", "", authHeader)
	if topProducts.Code != http.StatusOK {
		t.Fatalf("top products status = %d; body: %s", topProducts.Code, topProducts.Body.String())
	}
	invalidLimit := doJSONRequest(t, router, http.MethodGet,
		"/api/v1/reports/top-products?start_date="+date+"&end_date="+date+"&limit=101", "", authHeader)
	if invalidLimit.Code != http.StatusBadRequest {
		t.Fatalf("invalid top-products limit status = %d, want %d", invalidLimit.Code, http.StatusBadRequest)
	}
	valuation := doJSONRequest(t, router, http.MethodGet, "/api/v1/reports/stock-valuation?page=1&per_page=20", "", authHeader)
	if valuation.Code != http.StatusOK {
		t.Fatalf("stock valuation status = %d; body: %s", valuation.Code, valuation.Body.String())
	}
}
