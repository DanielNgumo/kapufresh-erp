package handlers

import (
	"errors"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/yourusername/kapufresh-erp/internal/dto"
	"github.com/yourusername/kapufresh-erp/internal/middleware"
	"github.com/yourusername/kapufresh-erp/internal/models"
	"github.com/yourusername/kapufresh-erp/internal/services"
	"github.com/yourusername/kapufresh-erp/internal/utils"
)

type CommerceHandler struct {
	commerce *services.CommerceService
}

func NewCommerceHandler(commerce *services.CommerceService) *CommerceHandler {
	return &CommerceHandler{commerce: commerce}
}

func (h *CommerceHandler) CreateSalesOrder(c *gin.Context) {
	actorID, ok := middleware.CurrentUserID(c)
	if !ok {
		unauthorized(c)
		return
	}
	var request dto.CreateSalesOrderRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		invalidBody(c)
		return
	}
	items := make([]services.SalesItemInput, 0, len(request.Items))
	for _, item := range request.Items {
		items = append(items, services.SalesItemInput{ProductID: item.ProductID, Quantity: item.Quantity})
	}
	order, err := h.commerce.CreateSalesOrder(request.CustomerID, actorID, items)
	if err != nil {
		h.respondError(c, err)
		return
	}
	utils.RespondData(c, http.StatusCreated, salesOrderResponse(order))
}

func (h *CommerceHandler) ListSalesOrders(c *gin.Context) {
	page, perPage, ok := catalogPageValues(c)
	if !ok {
		return
	}
	status := models.SalesOrderStatus(strings.TrimSpace(c.Query("status")))
	result, err := h.commerce.ListSalesOrders(page, perPage, status)
	if err != nil {
		h.respondError(c, err)
		return
	}
	orders := make([]dto.SalesOrderResponse, 0, len(result.Items))
	for _, order := range result.Items {
		orders = append(orders, salesOrderResponse(order))
	}
	utils.RespondData(c, http.StatusOK, dto.SalesOrderListResponse{
		Orders: orders, Pagination: dto.Pagination{
			Page: result.Page, PerPage: result.PerPage, Total: result.Total, TotalPages: result.TotalPages,
		},
	})
}

func (h *CommerceHandler) GetSalesOrder(c *gin.Context) {
	id, ok := catalogPathID(c)
	if !ok {
		return
	}
	order, err := h.commerce.GetSalesOrder(id)
	if err != nil {
		h.respondError(c, err)
		return
	}
	utils.RespondData(c, http.StatusOK, salesOrderResponse(order))
}

func (h *CommerceHandler) ConfirmSalesOrder(c *gin.Context) {
	id, ok := catalogPathID(c)
	if !ok {
		return
	}
	actorID, ok := middleware.CurrentUserID(c)
	if !ok {
		unauthorized(c)
		return
	}
	order, err := h.commerce.ConfirmSalesOrder(id, actorID)
	if err != nil {
		h.respondError(c, err)
		return
	}
	utils.RespondData(c, http.StatusOK, salesOrderResponse(order))
}

func (h *CommerceHandler) CancelSalesOrder(c *gin.Context) {
	id, ok := catalogPathID(c)
	if !ok {
		return
	}
	actorID, ok := middleware.CurrentUserID(c)
	if !ok {
		unauthorized(c)
		return
	}
	order, err := h.commerce.CancelSalesOrder(id, actorID)
	if err != nil {
		h.respondError(c, err)
		return
	}
	utils.RespondData(c, http.StatusOK, salesOrderResponse(order))
}

func (h *CommerceHandler) RecordPayment(c *gin.Context) {
	id, ok := catalogPathID(c)
	if !ok {
		return
	}
	actorID, ok := middleware.CurrentUserID(c)
	if !ok {
		unauthorized(c)
		return
	}
	var request dto.RecordPaymentRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		invalidBody(c)
		return
	}
	order, err := h.commerce.RecordManualPayment(id, actorID, request.AmountMinor, models.PaymentMethod(request.Method), request.Reference)
	if err != nil {
		h.respondError(c, err)
		return
	}
	utils.RespondData(c, http.StatusCreated, salesOrderResponse(order))
}

func (h *CommerceHandler) CreatePurchaseOrder(c *gin.Context) {
	actorID, ok := middleware.CurrentUserID(c)
	if !ok {
		unauthorized(c)
		return
	}
	var request dto.CreatePurchaseOrderRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		invalidBody(c)
		return
	}
	items := make([]services.PurchaseItemInput, 0, len(request.Items))
	for _, item := range request.Items {
		items = append(items, services.PurchaseItemInput{
			ProductID: item.ProductID, Quantity: item.Quantity, UnitCostMinor: item.UnitCostMinor,
		})
	}
	order, err := h.commerce.CreatePurchaseOrder(request.SupplierID, actorID, items)
	if err != nil {
		h.respondError(c, err)
		return
	}
	utils.RespondData(c, http.StatusCreated, purchaseOrderResponse(order))
}

func (h *CommerceHandler) ListPurchaseOrders(c *gin.Context) {
	page, perPage, ok := catalogPageValues(c)
	if !ok {
		return
	}
	status := models.PurchaseOrderStatus(strings.TrimSpace(c.Query("status")))
	result, err := h.commerce.ListPurchaseOrders(page, perPage, status)
	if err != nil {
		h.respondError(c, err)
		return
	}
	orders := make([]dto.PurchaseOrderResponse, 0, len(result.Items))
	for _, order := range result.Items {
		orders = append(orders, purchaseOrderResponse(order))
	}
	utils.RespondData(c, http.StatusOK, dto.PurchaseOrderListResponse{
		Orders: orders, Pagination: dto.Pagination{
			Page: result.Page, PerPage: result.PerPage, Total: result.Total, TotalPages: result.TotalPages,
		},
	})
}

func (h *CommerceHandler) GetPurchaseOrder(c *gin.Context) {
	id, ok := catalogPathID(c)
	if !ok {
		return
	}
	order, err := h.commerce.GetPurchaseOrder(id)
	if err != nil {
		h.respondError(c, err)
		return
	}
	utils.RespondData(c, http.StatusOK, purchaseOrderResponse(order))
}

func (h *CommerceHandler) OrderPurchaseOrder(c *gin.Context) {
	id, ok := catalogPathID(c)
	if !ok {
		return
	}
	order, err := h.commerce.OrderPurchaseOrder(id)
	if err != nil {
		h.respondError(c, err)
		return
	}
	utils.RespondData(c, http.StatusOK, purchaseOrderResponse(order))
}

func (h *CommerceHandler) ReceivePurchaseOrder(c *gin.Context) {
	id, ok := catalogPathID(c)
	if !ok {
		return
	}
	actorID, ok := middleware.CurrentUserID(c)
	if !ok {
		unauthorized(c)
		return
	}
	var request dto.ReceivePurchaseOrderRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		invalidBody(c)
		return
	}
	receipts := make([]services.PurchaseReceiptInput, 0, len(request.Items))
	for _, item := range request.Items {
		receipts = append(receipts, services.PurchaseReceiptInput{
			ItemID: item.PurchaseOrderItemID, Quantity: item.Quantity,
		})
	}
	order, err := h.commerce.ReceivePurchaseOrder(id, actorID, receipts)
	if err != nil {
		h.respondError(c, err)
		return
	}
	utils.RespondData(c, http.StatusOK, purchaseOrderResponse(order))
}

func (h *CommerceHandler) respondError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, services.ErrNotFound):
		utils.RespondError(c, http.StatusNotFound, "not_found", "Record was not found")
	case errors.Is(err, services.ErrInsufficientStock):
		utils.RespondError(c, http.StatusConflict, "insufficient_stock", "There is not enough stock to confirm this sales order")
	case errors.Is(err, services.ErrPaymentExceedsDue):
		utils.RespondError(c, http.StatusConflict, "payment_exceeds_due", "Payment exceeds the outstanding order balance")
	case errors.Is(err, services.ErrInvalidState):
		utils.RespondError(c, http.StatusConflict, "invalid_state", "This operation is not allowed for the current order status")
	case errors.Is(err, services.ErrInvalidInput):
		utils.RespondError(c, http.StatusBadRequest, "invalid_request", "Request values are invalid")
	default:
		log.Printf("commerce handler error: %v", err)
		utils.RespondError(c, http.StatusInternalServerError, "internal_error", "An unexpected error occurred")
	}
}

func salesOrderResponse(order models.SalesOrder) dto.SalesOrderResponse {
	items := make([]dto.SalesOrderItemResponse, 0, len(order.Items))
	for _, item := range order.Items {
		items = append(items, dto.SalesOrderItemResponse{
			ID: item.ID, ProductID: item.ProductID, ProductSKU: item.ProductSKU,
			ProductName: item.ProductName, Quantity: item.Quantity,
			UnitPriceMinor: item.UnitPriceMinor, LineTotalMinor: item.LineTotalMinor,
		})
	}
	payments := make([]dto.PaymentResponse, 0, len(order.Payments))
	var paid int64
	for _, payment := range order.Payments {
		paid += payment.AmountMinor
		payments = append(payments, dto.PaymentResponse{
			ID: payment.ID, AmountMinor: payment.AmountMinor, Method: string(payment.Method),
			Reference: payment.Reference, PaidAt: payment.PaidAt.Format(time.RFC3339),
			CreatedByID: payment.CreatedByID,
		})
	}
	customerName := order.Customer.Name
	return dto.SalesOrderResponse{
		ID: order.ID, OrderNumber: order.OrderNumber, CustomerID: order.CustomerID, CustomerName: customerName,
		Status: string(order.Status), CurrencyCode: order.CurrencyCode,
		SubtotalMinor: order.SubtotalMinor, TotalMinor: order.TotalMinor, PaidMinor: paid,
		Items: items, Payments: payments,
		CreatedAt: order.CreatedAt.Format(time.RFC3339), UpdatedAt: order.UpdatedAt.Format(time.RFC3339),
	}
}

func purchaseOrderResponse(order models.PurchaseOrder) dto.PurchaseOrderResponse {
	items := make([]dto.PurchaseOrderItemResponse, 0, len(order.Items))
	for _, item := range order.Items {
		items = append(items, dto.PurchaseOrderItemResponse{
			ID: item.ID, ProductID: item.ProductID, ProductSKU: item.Product.SKU,
			ProductName: item.Product.Name, Quantity: item.Quantity,
			ReceivedQuantity: item.ReceivedQuantity, UnitCostMinor: item.UnitCostMinor,
			LineTotalMinor: item.UnitCostMinor * item.Quantity,
		})
	}
	return dto.PurchaseOrderResponse{
		ID: order.ID, OrderNumber: order.OrderNumber, SupplierID: order.SupplierID,
		SupplierName: order.Supplier.Name, Status: string(order.Status), CurrencyCode: order.CurrencyCode,
		TotalMinor: order.TotalMinor, Items: items,
		CreatedAt: order.CreatedAt.Format(time.RFC3339), UpdatedAt: order.UpdatedAt.Format(time.RFC3339),
	}
}

func unauthorized(c *gin.Context) {
	utils.RespondError(c, http.StatusUnauthorized, "unauthorized", "Authentication is required")
}
