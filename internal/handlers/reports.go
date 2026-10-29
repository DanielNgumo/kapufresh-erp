package handlers

import (
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/yourusername/kapufresh-erp/internal/dto"
	"github.com/yourusername/kapufresh-erp/internal/services"
	"github.com/yourusername/kapufresh-erp/internal/utils"
)

type ReportHandler struct {
	reports *services.ReportService
}

func NewReportHandler(reports *services.ReportService) *ReportHandler {
	return &ReportHandler{reports: reports}
}

func (h *ReportHandler) SalesSummary(c *gin.Context) {
	startDate := strings.TrimSpace(c.Query("start_date"))
	endDate := strings.TrimSpace(c.Query("end_date"))
	result, err := h.reports.SalesSummary(startDate, endDate)
	if err != nil {
		h.respondError(c, err)
		return
	}
	daily := make([]dto.DailySalesSummary, 0, len(result.Daily))
	for _, row := range result.Daily {
		daily = append(daily, dto.DailySalesSummary{
			Date: row.Date, OrderCount: row.OrderCount, SalesTotalMinor: row.SalesTotalMinor,
			PaymentsTotalMinor: row.PaymentsTotalMinor,
		})
	}
	utils.RespondData(c, http.StatusOK, dto.SalesSummaryResponse{
		StartDate: result.StartDate, EndDate: result.EndDate, CurrencyCode: result.CurrencyCode,
		OrderCount:      result.OrderCount,
		SalesTotalMinor: result.SalesTotalMinor, PaymentsTotalMinor: result.PaymentsTotalMinor,
		Daily: daily,
	})
}

func (h *ReportHandler) TopProducts(c *gin.Context) {
	limit := 10
	if raw := strings.TrimSpace(c.Query("limit")); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil {
			utils.RespondError(c, http.StatusBadRequest, "invalid_query", "limit must be between 1 and 100")
			return
		}
		limit = parsed
	}
	result, err := h.reports.TopProducts(c.Query("start_date"), c.Query("end_date"), limit)
	if err != nil {
		h.respondError(c, err)
		return
	}
	products := make([]dto.TopProductResponse, 0, len(result.Products))
	for _, product := range result.Products {
		products = append(products, dto.TopProductResponse{
			ProductID: product.ProductID, ProductSKU: product.ProductSKU, ProductName: product.ProductName,
			QuantitySold: product.QuantitySold, SalesTotalMinor: product.SalesTotalMinor,
		})
	}
	utils.RespondData(c, http.StatusOK, dto.TopProductsReportResponse{
		StartDate: result.StartDate, EndDate: result.EndDate,
		CurrencyCode: result.CurrencyCode, Products: products,
	})
}

func (h *ReportHandler) StockValuation(c *gin.Context) {
	page, perPage, ok := catalogPageValues(c)
	if !ok {
		return
	}
	result, err := h.reports.StockValuation(page, perPage)
	if err != nil {
		h.respondError(c, err)
		return
	}
	products := make([]dto.StockValuationProduct, 0, len(result.Products))
	for _, product := range result.Products {
		products = append(products, dto.StockValuationProduct{
			ProductID: product.ProductID, SKU: product.SKU, Name: product.Name,
			StockOnHand: product.StockOnHand, CostPriceMinor: product.CostPriceMinor,
			UnitPriceMinor: product.UnitPriceMinor, CostValueMinor: product.CostValueMinor,
			RetailValueMinor: product.RetailValueMinor,
		})
	}
	utils.RespondData(c, http.StatusOK, dto.StockValuationResponse{
		Products: products,
		Pagination: dto.Pagination{
			Page: result.Page, PerPage: result.PerPage, Total: result.TotalProducts, TotalPages: result.TotalPages,
		},
		CurrencyCode:   result.CurrencyCode,
		TotalCostMinor: result.TotalCostMinor, TotalRetailMinor: result.TotalRetailMinor,
	})
}

func (h *ReportHandler) respondError(c *gin.Context, err error) {
	if errors.Is(err, services.ErrInvalidInput) {
		utils.RespondError(c, http.StatusBadRequest, "invalid_request", "Date range or report parameters are invalid")
		return
	}
	log.Printf("report handler error: %v", err)
	utils.RespondError(c, http.StatusInternalServerError, "internal_error", "An unexpected error occurred")
}
