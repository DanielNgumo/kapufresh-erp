package handlers_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/yourusername/kapufresh-erp/internal/database"
	"github.com/yourusername/kapufresh-erp/internal/models"
	"github.com/yourusername/kapufresh-erp/internal/routes"
	"github.com/yourusername/kapufresh-erp/internal/services"
)

func TestSalesAndPurchaseEndpoints(t *testing.T) {
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
	if err := database.SeedAdmin(db, "commerce-handler@example.com", "commerce-handler-password", "Commerce Admin"); err != nil {
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
	_, tokens, err := auth.Login("commerce-handler@example.com", "commerce-handler-password")
	if err != nil {
		t.Fatal(err)
	}
	customer, err := catalog.CreateCustomer("Commerce Customer", "", "", "", "")
	if err != nil {
		t.Fatal(err)
	}
	supplier, err := catalog.CreateSupplier("Commerce Supplier", "", "", "", "")
	if err != nil {
		t.Fatal(err)
	}
	product, err := catalog.CreateProduct("COM-001", "Coffee", "Grocery", "bag", 1000, 600, 0)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := inventory.RecordMovement(product.ID, 1, models.StockMovementIn, 10, "Initial stock"); err != nil {
		t.Fatal(err)
	}
	authHeader := "Bearer " + tokens.AccessToken

	sale := doJSONRequest(t, router, http.MethodPost, "/api/v1/sales/orders",
		`{"customer_id":`+strconv.FormatUint(uint64(customer.ID), 10)+`,"items":[{"product_id":`+strconv.FormatUint(uint64(product.ID), 10)+`,"quantity":2}]}`, authHeader)
	if sale.Code != http.StatusCreated {
		t.Fatalf("create sale status = %d; body: %s", sale.Code, sale.Body.String())
	}
	var saleResponse struct {
		Data struct {
			ID     uint   `json:"id"`
			Status string `json:"status"`
		} `json:"data"`
	}
	decodeResponse(t, sale.Body.Bytes(), &saleResponse)
	if saleResponse.Data.Status != string(models.SalesOrderDraft) {
		t.Fatalf("new sale status = %q, want draft", saleResponse.Data.Status)
	}
	orderPath := "/api/v1/sales/orders/" + strconv.FormatUint(uint64(saleResponse.Data.ID), 10)
	confirmed := doJSONRequest(t, router, http.MethodPost, orderPath+"/confirm", `{}`, authHeader)
	if confirmed.Code != http.StatusOK {
		t.Fatalf("confirm sale status = %d; body: %s", confirmed.Code, confirmed.Body.String())
	}
	paid := doJSONRequest(t, router, http.MethodPost, orderPath+"/payments",
		`{"amount_minor":2000,"method":"cash","reference":"cash-sale-1"}`, authHeader)
	if paid.Code != http.StatusCreated {
		t.Fatalf("record payment status = %d; body: %s", paid.Code, paid.Body.String())
	}

	purchase := doJSONRequest(t, router, http.MethodPost, "/api/v1/purchases/orders",
		`{"supplier_id":`+strconv.FormatUint(uint64(supplier.ID), 10)+`,"items":[{"product_id":`+strconv.FormatUint(uint64(product.ID), 10)+`,"quantity":4,"unit_cost_minor":500}]}`, authHeader)
	if purchase.Code != http.StatusCreated {
		t.Fatalf("create purchase status = %d; body: %s", purchase.Code, purchase.Body.String())
	}
	var purchaseResponse struct {
		Data struct {
			ID    uint `json:"id"`
			Items []struct {
				ID uint `json:"id"`
			} `json:"items"`
		} `json:"data"`
	}
	decodeResponse(t, purchase.Body.Bytes(), &purchaseResponse)
	if len(purchaseResponse.Data.Items) != 1 {
		t.Fatalf("purchase response has %d items, want 1", len(purchaseResponse.Data.Items))
	}
	purchasePath := "/api/v1/purchases/orders/" + strconv.FormatUint(uint64(purchaseResponse.Data.ID), 10)
	ordered := doJSONRequest(t, router, http.MethodPost, purchasePath+"/order", `{}`, authHeader)
	if ordered.Code != http.StatusOK {
		t.Fatalf("submit purchase status = %d; body: %s", ordered.Code, ordered.Body.String())
	}
	receipt := doJSONRequest(t, router, http.MethodPost, purchasePath+"/receive",
		`{"items":[{"purchase_order_item_id":`+strconv.FormatUint(uint64(purchaseResponse.Data.Items[0].ID), 10)+`,"quantity":4}]}`, authHeader)
	if receipt.Code != http.StatusOK {
		t.Fatalf("receive purchase status = %d; body: %s", receipt.Code, receipt.Body.String())
	}

	listSales := httptest.NewRecorder()
	listSalesRequest := httptest.NewRequest(http.MethodGet, "/api/v1/sales/orders?status=paid&page=1&per_page=5", nil)
	listSalesRequest.Header.Set("Authorization", authHeader)
	router.ServeHTTP(listSales, listSalesRequest)
	if listSales.Code != http.StatusOK {
		t.Fatalf("list sales status = %d; body: %s", listSales.Code, listSales.Body.String())
	}

	listPurchases := httptest.NewRecorder()
	listPurchasesRequest := httptest.NewRequest(http.MethodGet, "/api/v1/purchases/orders?status=received", nil)
	listPurchasesRequest.Header.Set("Authorization", authHeader)
	router.ServeHTTP(listPurchases, listPurchasesRequest)
	if listPurchases.Code != http.StatusOK {
		t.Fatalf("list purchases status = %d; body: %s", listPurchases.Code, listPurchases.Body.String())
	}
}

func doJSONRequest(t *testing.T, router http.Handler, method, path, body, authHeader string) *httptest.ResponseRecorder {
	t.Helper()
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(method, path, bytes.NewBufferString(body))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", authHeader)
	router.ServeHTTP(recorder, request)
	return recorder
}

func decodeResponse(t *testing.T, body []byte, target any) {
	t.Helper()
	if err := json.Unmarshal(body, target); err != nil {
		t.Fatal(err)
	}
}
