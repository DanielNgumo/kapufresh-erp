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
	"github.com/yourusername/kapufresh-erp/internal/models"
	"github.com/yourusername/kapufresh-erp/internal/services"
	"github.com/yourusername/kapufresh-erp/internal/utils"
)

type CatalogHandler struct {
	catalog *services.CatalogService
}

func NewCatalogHandler(catalog *services.CatalogService) *CatalogHandler {
	return &CatalogHandler{catalog: catalog}
}

func (h *CatalogHandler) ListCustomers(c *gin.Context) {
	filter, ok := catalogFilter(c)
	if !ok {
		return
	}
	result, err := h.catalog.ListCustomers(filter)
	if err != nil {
		h.respondError(c, err)
		return
	}
	items := make([]dto.CustomerResponse, 0, len(result.Items))
	for _, item := range result.Items {
		items = append(items, customerResponse(item))
	}
	utils.RespondData(c, http.StatusOK, dto.CustomerListResponse{Customers: items, Pagination: catalogPagination(result.Page, result.PerPage, result.Total, result.TotalPages)})
}

func (h *CatalogHandler) CreateCustomer(c *gin.Context) {
	var request dto.CustomerRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		invalidBody(c)
		return
	}
	item, err := h.catalog.CreateCustomer(request.Name, request.Email, request.Phone, request.Address, request.Notes)
	if err != nil {
		h.respondError(c, err)
		return
	}
	utils.RespondData(c, http.StatusCreated, customerResponse(item))
}

func (h *CatalogHandler) GetCustomer(c *gin.Context) {
	id, ok := catalogPathID(c)
	if !ok {
		return
	}
	item, err := h.catalog.GetCustomer(id)
	if err != nil {
		h.respondError(c, err)
		return
	}
	utils.RespondData(c, http.StatusOK, customerResponse(item))
}

func (h *CatalogHandler) UpdateCustomer(c *gin.Context) {
	id, ok := catalogPathID(c)
	if !ok {
		return
	}
	var request dto.UpdateCustomerRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		invalidBody(c)
		return
	}
	item, err := h.catalog.UpdateCustomer(id, services.CustomerUpdate{
		Name: request.Name, Email: request.Email, Phone: request.Phone,
		Address: request.Address, Notes: request.Notes,
	})
	if err != nil {
		h.respondError(c, err)
		return
	}
	utils.RespondData(c, http.StatusOK, customerResponse(item))
}

func (h *CatalogHandler) DeleteCustomer(c *gin.Context) {
	id, ok := catalogPathID(c)
	if !ok {
		return
	}
	if err := h.catalog.DeleteCustomer(id); err != nil {
		h.respondError(c, err)
		return
	}
	utils.RespondData(c, http.StatusOK, gin.H{"message": "Customer deleted"})
}

func (h *CatalogHandler) ListSuppliers(c *gin.Context) {
	filter, ok := catalogFilter(c)
	if !ok {
		return
	}
	result, err := h.catalog.ListSuppliers(filter)
	if err != nil {
		h.respondError(c, err)
		return
	}
	items := make([]dto.SupplierResponse, 0, len(result.Items))
	for _, item := range result.Items {
		items = append(items, supplierResponse(item))
	}
	utils.RespondData(c, http.StatusOK, dto.SupplierListResponse{Suppliers: items, Pagination: catalogPagination(result.Page, result.PerPage, result.Total, result.TotalPages)})
}

func (h *CatalogHandler) CreateSupplier(c *gin.Context) {
	var request dto.SupplierRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		invalidBody(c)
		return
	}
	item, err := h.catalog.CreateSupplier(request.Name, request.Email, request.Phone, request.Address, request.Notes)
	if err != nil {
		h.respondError(c, err)
		return
	}
	utils.RespondData(c, http.StatusCreated, supplierResponse(item))
}

func (h *CatalogHandler) GetSupplier(c *gin.Context) {
	id, ok := catalogPathID(c)
	if !ok {
		return
	}
	item, err := h.catalog.GetSupplier(id)
	if err != nil {
		h.respondError(c, err)
		return
	}
	utils.RespondData(c, http.StatusOK, supplierResponse(item))
}

func (h *CatalogHandler) UpdateSupplier(c *gin.Context) {
	id, ok := catalogPathID(c)
	if !ok {
		return
	}
	var request dto.UpdateSupplierRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		invalidBody(c)
		return
	}
	item, err := h.catalog.UpdateSupplier(id, services.SupplierUpdate{
		Name: request.Name, Email: request.Email, Phone: request.Phone,
		Address: request.Address, Notes: request.Notes,
	})
	if err != nil {
		h.respondError(c, err)
		return
	}
	utils.RespondData(c, http.StatusOK, supplierResponse(item))
}

func (h *CatalogHandler) DeleteSupplier(c *gin.Context) {
	id, ok := catalogPathID(c)
	if !ok {
		return
	}
	if err := h.catalog.DeleteSupplier(id); err != nil {
		h.respondError(c, err)
		return
	}
	utils.RespondData(c, http.StatusOK, gin.H{"message": "Supplier deleted"})
}

func (h *CatalogHandler) ListProducts(c *gin.Context) {
	filter, ok := catalogFilter(c)
	if !ok {
		return
	}
	filter.Category = strings.TrimSpace(c.Query("category"))
	result, err := h.catalog.ListProducts(filter)
	if err != nil {
		h.respondError(c, err)
		return
	}
	items := make([]dto.ProductResponse, 0, len(result.Items))
	for _, item := range result.Items {
		items = append(items, productResponse(item))
	}
	utils.RespondData(c, http.StatusOK, dto.ProductListResponse{Products: items, Pagination: catalogPagination(result.Page, result.PerPage, result.Total, result.TotalPages)})
}

func (h *CatalogHandler) CreateProduct(c *gin.Context) {
	var request dto.ProductRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		invalidBody(c)
		return
	}
	item, err := h.catalog.CreateProduct(request.SKU, request.Name, request.Category, request.Unit, request.UnitPriceMinor, request.CostPriceMinor, request.ReorderLevel)
	if err != nil {
		h.respondError(c, err)
		return
	}
	utils.RespondData(c, http.StatusCreated, productResponse(item))
}

func (h *CatalogHandler) GetProduct(c *gin.Context) {
	id, ok := catalogPathID(c)
	if !ok {
		return
	}
	item, err := h.catalog.GetProduct(id)
	if err != nil {
		h.respondError(c, err)
		return
	}
	utils.RespondData(c, http.StatusOK, productResponse(item))
}

func (h *CatalogHandler) UpdateProduct(c *gin.Context) {
	id, ok := catalogPathID(c)
	if !ok {
		return
	}
	var request dto.UpdateProductRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		invalidBody(c)
		return
	}
	item, err := h.catalog.UpdateProduct(id, services.ProductUpdate{
		SKU: request.SKU, Name: request.Name, Category: request.Category, Unit: request.Unit,
		UnitPriceMinor: request.UnitPriceMinor, CostPriceMinor: request.CostPriceMinor,
		ReorderLevel: request.ReorderLevel,
	})
	if err != nil {
		h.respondError(c, err)
		return
	}
	utils.RespondData(c, http.StatusOK, productResponse(item))
}

func (h *CatalogHandler) DeleteProduct(c *gin.Context) {
	id, ok := catalogPathID(c)
	if !ok {
		return
	}
	if err := h.catalog.DeleteProduct(id); err != nil {
		h.respondError(c, err)
		return
	}
	utils.RespondData(c, http.StatusOK, gin.H{"message": "Product deleted"})
}

func (h *CatalogHandler) respondError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, services.ErrNotFound):
		utils.RespondError(c, http.StatusNotFound, "not_found", "Record was not found")
	case errors.Is(err, services.ErrConflict):
		utils.RespondError(c, http.StatusConflict, "conflict", "A product with that SKU already exists")
	case errors.Is(err, services.ErrInvalidInput):
		utils.RespondError(c, http.StatusBadRequest, "invalid_request", "Request values are invalid")
	default:
		log.Printf("catalog handler error: %v", err)
		utils.RespondError(c, http.StatusInternalServerError, "internal_error", "An unexpected error occurred")
	}
}

func customerResponse(item models.Customer) dto.CustomerResponse {
	return dto.CustomerResponse{
		ID: item.ID, Name: item.Name, Email: item.Email, Phone: item.Phone, Address: item.Address, Notes: item.Notes,
		CreatedAt: item.CreatedAt.Format(time.RFC3339), UpdatedAt: item.UpdatedAt.Format(time.RFC3339),
	}
}

func supplierResponse(item models.Supplier) dto.SupplierResponse {
	return dto.SupplierResponse{
		ID: item.ID, Name: item.Name, Email: item.Email, Phone: item.Phone, Address: item.Address, Notes: item.Notes,
		CreatedAt: item.CreatedAt.Format(time.RFC3339), UpdatedAt: item.UpdatedAt.Format(time.RFC3339),
	}
}

func productResponse(item models.Product) dto.ProductResponse {
	return dto.ProductResponse{
		ID: item.ID, SKU: item.SKU, Name: item.Name, Category: item.Category, Unit: item.Unit,
		UnitPriceMinor: item.UnitPriceMinor, CostPriceMinor: item.CostPriceMinor,
		StockOnHand: item.StockOnHand, ReorderLevel: item.ReorderLevel,
		CreatedAt: item.CreatedAt.Format(time.RFC3339), UpdatedAt: item.UpdatedAt.Format(time.RFC3339),
	}
}

func catalogPagination(page, perPage int, total int64, totalPages int) dto.Pagination {
	return dto.Pagination{Page: page, PerPage: perPage, Total: total, TotalPages: totalPages}
}

func catalogFilter(c *gin.Context) (services.CatalogFilter, bool) {
	page, err := queryInt(c, "page", 1)
	if err != nil {
		utils.RespondError(c, http.StatusBadRequest, "invalid_query", "page must be a positive integer")
		return services.CatalogFilter{}, false
	}
	perPage, err := queryInt(c, "per_page", 20)
	if err != nil || perPage > 100 {
		utils.RespondError(c, http.StatusBadRequest, "invalid_query", "per_page must be between 1 and 100")
		return services.CatalogFilter{}, false
	}
	return services.CatalogFilter{
		Page: page, PerPage: perPage, Search: strings.TrimSpace(c.Query("search")),
		Sort: strings.TrimSpace(c.Query("sort")), Order: strings.ToLower(strings.TrimSpace(c.Query("order"))),
	}, true
}

func catalogPathID(c *gin.Context) (uint, bool) {
	id, err := strconv.ParseUint(c.Param("id"), 10, strconv.IntSize)
	if err != nil || id == 0 {
		utils.RespondError(c, http.StatusBadRequest, "invalid_id", "ID must be a positive integer")
		return 0, false
	}
	return uint(id), true
}

func invalidBody(c *gin.Context) {
	utils.RespondError(c, http.StatusBadRequest, "invalid_request", "Request body is invalid")
}
