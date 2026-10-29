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
	"github.com/yourusername/kapufresh-erp/internal/routes"
	"github.com/yourusername/kapufresh-erp/internal/services"
)

func TestCatalogRoutesUseValidationAndRBAC(t *testing.T) {
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
	if err := database.SeedAdmin(db, "admin@example.com", "catalog-handler-password", "Test Admin"); err != nil {
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
	router := routes.New(auth, catalog)

	unauthorized := httptest.NewRecorder()
	router.ServeHTTP(unauthorized, httptest.NewRequest(http.MethodGet, "/api/v1/customers", nil))
	if unauthorized.Code != http.StatusUnauthorized {
		t.Fatalf("unauthorized catalog status = %d, want %d", unauthorized.Code, http.StatusUnauthorized)
	}

	_, tokens, err := auth.Login("admin@example.com", "catalog-handler-password")
	if err != nil {
		t.Fatal(err)
	}

	customerBody := []byte(`{"name":"Sample Shop","email":"shop@example.com"}`)
	createCustomer := httptest.NewRecorder()
	createCustomerRequest := httptest.NewRequest(http.MethodPost, "/api/v1/customers", bytes.NewReader(customerBody))
	createCustomerRequest.Header.Set("Content-Type", "application/json")
	createCustomerRequest.Header.Set("Authorization", "Bearer "+tokens.AccessToken)
	router.ServeHTTP(createCustomer, createCustomerRequest)
	if createCustomer.Code != http.StatusCreated {
		t.Fatalf("create customer status = %d, want %d; body: %s", createCustomer.Code, http.StatusCreated, createCustomer.Body.String())
	}
	var customerResponse struct {
		Data struct {
			ID uint `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(createCustomer.Body.Bytes(), &customerResponse); err != nil {
		t.Fatal(err)
	}
	listCustomers := httptest.NewRecorder()
	listRequest := httptest.NewRequest(http.MethodGet, "/api/v1/customers?search=sample&page=1&per_page=5&sort=name&order=asc", nil)
	listRequest.Header.Set("Authorization", "Bearer "+tokens.AccessToken)
	router.ServeHTTP(listCustomers, listRequest)
	if listCustomers.Code != http.StatusOK {
		t.Fatalf("list customers status = %d, want %d; body: %s", listCustomers.Code, http.StatusOK, listCustomers.Body.String())
	}
	customerPath := "/api/v1/customers/" + strconv.FormatUint(uint64(customerResponse.Data.ID), 10)
	getCustomer := httptest.NewRecorder()
	getRequest := httptest.NewRequest(http.MethodGet, customerPath, nil)
	getRequest.Header.Set("Authorization", "Bearer "+tokens.AccessToken)
	router.ServeHTTP(getCustomer, getRequest)
	if getCustomer.Code != http.StatusOK {
		t.Fatalf("get customer status = %d, want %d; body: %s", getCustomer.Code, http.StatusOK, getCustomer.Body.String())
	}
	patchCustomer := httptest.NewRecorder()
	patchRequest := httptest.NewRequest(http.MethodPatch, customerPath, bytes.NewReader([]byte(`{"phone":"555-1234"}`)))
	patchRequest.Header.Set("Content-Type", "application/json")
	patchRequest.Header.Set("Authorization", "Bearer "+tokens.AccessToken)
	router.ServeHTTP(patchCustomer, patchRequest)
	if patchCustomer.Code != http.StatusOK {
		t.Fatalf("patch customer status = %d, want %d; body: %s", patchCustomer.Code, http.StatusOK, patchCustomer.Body.String())
	}

	productBody := []byte(`{"sku":"GRO-001","name":"Maize Flour","category":"Grocery","unit":"bag","unit_price_minor":1000,"cost_price_minor":750,"reorder_level":10}`)
	createProduct := httptest.NewRecorder()
	createProductRequest := httptest.NewRequest(http.MethodPost, "/api/v1/products", bytes.NewReader(productBody))
	createProductRequest.Header.Set("Content-Type", "application/json")
	createProductRequest.Header.Set("Authorization", "Bearer "+tokens.AccessToken)
	router.ServeHTTP(createProduct, createProductRequest)
	if createProduct.Code != http.StatusCreated {
		t.Fatalf("create product status = %d, want %d; body: %s", createProduct.Code, http.StatusCreated, createProduct.Body.String())
	}
	var productResponse struct {
		Data struct {
			ID uint `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(createProduct.Body.Bytes(), &productResponse); err != nil {
		t.Fatal(err)
	}
	productPath := "/api/v1/products/" + strconv.FormatUint(uint64(productResponse.Data.ID), 10)
	listProducts := httptest.NewRecorder()
	listProductsRequest := httptest.NewRequest(http.MethodGet, "/api/v1/products?category=Grocery&sort=name&order=desc", nil)
	listProductsRequest.Header.Set("Authorization", "Bearer "+tokens.AccessToken)
	router.ServeHTTP(listProducts, listProductsRequest)
	if listProducts.Code != http.StatusOK {
		t.Fatalf("list products status = %d, want %d; body: %s", listProducts.Code, http.StatusOK, listProducts.Body.String())
	}
	getProduct := httptest.NewRecorder()
	getProductRequest := httptest.NewRequest(http.MethodGet, productPath, nil)
	getProductRequest.Header.Set("Authorization", "Bearer "+tokens.AccessToken)
	router.ServeHTTP(getProduct, getProductRequest)
	if getProduct.Code != http.StatusOK {
		t.Fatalf("get product status = %d, want %d; body: %s", getProduct.Code, http.StatusOK, getProduct.Body.String())
	}
	patchProduct := httptest.NewRecorder()
	patchProductRequest := httptest.NewRequest(http.MethodPatch, productPath, bytes.NewReader([]byte(`{"unit_price_minor":1050}`)))
	patchProductRequest.Header.Set("Content-Type", "application/json")
	patchProductRequest.Header.Set("Authorization", "Bearer "+tokens.AccessToken)
	router.ServeHTTP(patchProduct, patchProductRequest)
	if patchProduct.Code != http.StatusOK {
		t.Fatalf("patch product status = %d, want %d; body: %s", patchProduct.Code, http.StatusOK, patchProduct.Body.String())
	}

	invalidProduct := httptest.NewRecorder()
	invalidProductRequest := httptest.NewRequest(http.MethodPost, "/api/v1/products", bytes.NewReader(
		[]byte(`{"sku":"GRO-002","name":"Invalid","unit":"bag","unit_price_minor":-1,"cost_price_minor":2,"reorder_level":0}`),
	))
	invalidProductRequest.Header.Set("Content-Type", "application/json")
	invalidProductRequest.Header.Set("Authorization", "Bearer "+tokens.AccessToken)
	router.ServeHTTP(invalidProduct, invalidProductRequest)
	if invalidProduct.Code != http.StatusBadRequest {
		t.Fatalf("invalid product status = %d, want %d; body: %s", invalidProduct.Code, http.StatusBadRequest, invalidProduct.Body.String())
	}

	supplierBody := []byte(`{"name":"Local Supplier","email":"sales@example.com"}`)
	createSupplier := httptest.NewRecorder()
	createSupplierRequest := httptest.NewRequest(http.MethodPost, "/api/v1/suppliers", bytes.NewReader(supplierBody))
	createSupplierRequest.Header.Set("Content-Type", "application/json")
	createSupplierRequest.Header.Set("Authorization", "Bearer "+tokens.AccessToken)
	router.ServeHTTP(createSupplier, createSupplierRequest)
	if createSupplier.Code != http.StatusCreated {
		t.Fatalf("create supplier status = %d, want %d; body: %s", createSupplier.Code, http.StatusCreated, createSupplier.Body.String())
	}
	var supplierResponse struct {
		Data struct {
			ID uint `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(createSupplier.Body.Bytes(), &supplierResponse); err != nil {
		t.Fatal(err)
	}
	supplierPath := "/api/v1/suppliers/" + strconv.FormatUint(uint64(supplierResponse.Data.ID), 10)
	listSuppliers := httptest.NewRecorder()
	listSuppliersRequest := httptest.NewRequest(http.MethodGet, "/api/v1/suppliers?search=local&sort=name", nil)
	listSuppliersRequest.Header.Set("Authorization", "Bearer "+tokens.AccessToken)
	router.ServeHTTP(listSuppliers, listSuppliersRequest)
	if listSuppliers.Code != http.StatusOK {
		t.Fatalf("list suppliers status = %d, want %d; body: %s", listSuppliers.Code, http.StatusOK, listSuppliers.Body.String())
	}
	getSupplier := httptest.NewRecorder()
	getSupplierRequest := httptest.NewRequest(http.MethodGet, supplierPath, nil)
	getSupplierRequest.Header.Set("Authorization", "Bearer "+tokens.AccessToken)
	router.ServeHTTP(getSupplier, getSupplierRequest)
	if getSupplier.Code != http.StatusOK {
		t.Fatalf("get supplier status = %d, want %d; body: %s", getSupplier.Code, http.StatusOK, getSupplier.Body.String())
	}
	patchSupplier := httptest.NewRecorder()
	patchSupplierRequest := httptest.NewRequest(http.MethodPatch, supplierPath, bytes.NewReader([]byte(`{"phone":"555-8888"}`)))
	patchSupplierRequest.Header.Set("Content-Type", "application/json")
	patchSupplierRequest.Header.Set("Authorization", "Bearer "+tokens.AccessToken)
	router.ServeHTTP(patchSupplier, patchSupplierRequest)
	if patchSupplier.Code != http.StatusOK {
		t.Fatalf("patch supplier status = %d, want %d; body: %s", patchSupplier.Code, http.StatusOK, patchSupplier.Body.String())
	}
	deleteSupplier := httptest.NewRecorder()
	deleteSupplierRequest := httptest.NewRequest(http.MethodDelete, supplierPath, nil)
	deleteSupplierRequest.Header.Set("Authorization", "Bearer "+tokens.AccessToken)
	router.ServeHTTP(deleteSupplier, deleteSupplierRequest)
	if deleteSupplier.Code != http.StatusOK {
		t.Fatalf("delete supplier status = %d, want %d; body: %s", deleteSupplier.Code, http.StatusOK, deleteSupplier.Body.String())
	}

	deleteProduct := httptest.NewRecorder()
	deleteProductRequest := httptest.NewRequest(http.MethodDelete, productPath, nil)
	deleteProductRequest.Header.Set("Authorization", "Bearer "+tokens.AccessToken)
	router.ServeHTTP(deleteProduct, deleteProductRequest)
	if deleteProduct.Code != http.StatusOK {
		t.Fatalf("delete product status = %d, want %d; body: %s", deleteProduct.Code, http.StatusOK, deleteProduct.Body.String())
	}

	deleteCustomer := httptest.NewRecorder()
	deleteCustomerRequest := httptest.NewRequest(http.MethodDelete, customerPath, nil)
	deleteCustomerRequest.Header.Set("Authorization", "Bearer "+tokens.AccessToken)
	router.ServeHTTP(deleteCustomer, deleteCustomerRequest)
	if deleteCustomer.Code != http.StatusOK {
		t.Fatalf("delete customer status = %d, want %d; body: %s", deleteCustomer.Code, http.StatusOK, deleteCustomer.Body.String())
	}

	if _, err := auth.Register("Staff User", "staff@example.com", "catalog-staff-password", "staff"); err != nil {
		t.Fatal(err)
	}
	_, staffTokens, err := auth.Login("staff@example.com", "catalog-staff-password")
	if err != nil {
		t.Fatal(err)
	}
	staffCreate := httptest.NewRecorder()
	staffCreateRequest := httptest.NewRequest(http.MethodPost, "/api/v1/customers", bytes.NewReader(customerBody))
	staffCreateRequest.Header.Set("Content-Type", "application/json")
	staffCreateRequest.Header.Set("Authorization", "Bearer "+staffTokens.AccessToken)
	router.ServeHTTP(staffCreate, staffCreateRequest)
	if staffCreate.Code != http.StatusForbidden {
		t.Fatalf("staff create customer status = %d, want %d", staffCreate.Code, http.StatusForbidden)
	}
}
