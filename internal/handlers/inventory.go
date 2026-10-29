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

type InventoryHandler struct {
	inventory *services.InventoryService
}

func NewInventoryHandler(inventory *services.InventoryService) *InventoryHandler {
	return &InventoryHandler{inventory: inventory}
}

func (h *InventoryHandler) RecordMovement(c *gin.Context) {
	productID, ok := catalogPathID(c)
	if !ok {
		return
	}
	actorID, ok := middleware.CurrentUserID(c)
	if !ok {
		utils.RespondError(c, http.StatusUnauthorized, "unauthorized", "Authentication is required")
		return
	}
	var request dto.StockMovementRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		invalidBody(c)
		return
	}
	movement, err := h.inventory.RecordMovement(productID, actorID, models.StockMovementType(request.Type), request.Quantity, request.Reason)
	if err != nil {
		h.respondError(c, err)
		return
	}
	utils.RespondData(c, http.StatusCreated, stockMovementResponse(movement))
}

func (h *InventoryHandler) ListProductMovements(c *gin.Context) {
	productID, ok := catalogPathID(c)
	if !ok {
		return
	}
	if _, err := h.inventory.GetProduct(productID); err != nil {
		h.respondError(c, err)
		return
	}
	h.listMovements(c, productID)
}

func (h *InventoryHandler) ListMovements(c *gin.Context) {
	var productID uint
	if raw := strings.TrimSpace(c.Query("product_id")); raw != "" {
		parsed, err := strconv.ParseUint(raw, 10, strconv.IntSize)
		if err != nil || parsed == 0 {
			utils.RespondError(c, http.StatusBadRequest, "invalid_query", "product_id must be a positive integer")
			return
		}
		productID = uint(parsed)
	}
	h.listMovements(c, productID)
}

func (h *InventoryHandler) listMovements(c *gin.Context, productID uint) {
	page, perPage, ok := catalogPageValues(c)
	if !ok {
		return
	}
	movementType := models.StockMovementType(strings.TrimSpace(c.Query("type")))
	result, err := h.inventory.ListMovements(services.MovementFilter{
		ProductID: productID, Type: movementType, Page: page, PerPage: perPage,
	})
	if err != nil {
		h.respondError(c, err)
		return
	}
	items := make([]dto.StockMovementResponse, 0, len(result.Items))
	for _, item := range result.Items {
		items = append(items, stockMovementResponse(item))
	}
	utils.RespondData(c, http.StatusOK, dto.StockMovementListResponse{
		Movements: items,
		Pagination: dto.Pagination{
			Page: result.Page, PerPage: result.PerPage, Total: result.Total, TotalPages: result.TotalPages,
		},
	})
}

func (h *InventoryHandler) LowStock(c *gin.Context) {
	page, perPage, ok := catalogPageValues(c)
	if !ok {
		return
	}
	result, err := h.inventory.ListLowStock(page, perPage)
	if err != nil {
		h.respondError(c, err)
		return
	}
	items := make([]dto.LowStockProductResponse, 0, len(result.Items))
	for _, product := range result.Items {
		items = append(items, dto.LowStockProductResponse{
			ProductResponse: productResponse(product),
			StockShortfall:  product.ReorderLevel - product.StockOnHand,
		})
	}
	utils.RespondData(c, http.StatusOK, dto.LowStockListResponse{
		Products: items,
		Pagination: dto.Pagination{
			Page: result.Page, PerPage: result.PerPage, Total: result.Total, TotalPages: result.TotalPages,
		},
	})
}

func (h *InventoryHandler) respondError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, services.ErrNotFound):
		utils.RespondError(c, http.StatusNotFound, "not_found", "Product was not found")
	case errors.Is(err, services.ErrInsufficientStock):
		utils.RespondError(c, http.StatusConflict, "insufficient_stock", "The stock change would result in a negative or overflowing balance")
	case errors.Is(err, services.ErrInvalidInput):
		utils.RespondError(c, http.StatusBadRequest, "invalid_request", "Request values are invalid")
	default:
		log.Printf("inventory handler error: %v", err)
		utils.RespondError(c, http.StatusInternalServerError, "internal_error", "An unexpected error occurred")
	}
}

func stockMovementResponse(movement models.StockMovement) dto.StockMovementResponse {
	return dto.StockMovementResponse{
		ID: movement.ID, ProductID: movement.ProductID, ProductSKU: movement.Product.SKU,
		ProductName: movement.Product.Name, Type: string(movement.Type), Quantity: movement.Quantity,
		Reason: movement.Reason, SourceType: movement.SourceType, SourceID: movement.SourceID,
		CreatedByID: movement.CreatedByID, CreatedAt: movement.CreatedAt.Format(time.RFC3339),
	}
}

func catalogPageValues(c *gin.Context) (int, int, bool) {
	page, err := queryInt(c, "page", 1)
	if err != nil {
		utils.RespondError(c, http.StatusBadRequest, "invalid_query", "page must be a positive integer")
		return 0, 0, false
	}
	perPage, err := queryInt(c, "per_page", 20)
	if err != nil || perPage > 100 {
		utils.RespondError(c, http.StatusBadRequest, "invalid_query", "per_page must be between 1 and 100")
		return 0, 0, false
	}
	return page, perPage, true
}
