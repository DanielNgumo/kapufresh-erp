package services

import (
	"fmt"
	"math"
	"sort"
	"time"

	"github.com/yourusername/kapufresh-erp/internal/models"
	"gorm.io/gorm"
)

type ReportService struct {
	db           *gorm.DB
	currencyCode string
}

type DailySalesSummary struct {
	Date               string
	OrderCount         int64
	SalesTotalMinor    int64
	PaymentsTotalMinor int64
}

type SalesSummary struct {
	StartDate          string
	EndDate            string
	CurrencyCode       string
	OrderCount         int64
	SalesTotalMinor    int64
	PaymentsTotalMinor int64
	Daily              []DailySalesSummary
}

type TopProduct struct {
	ProductID       uint
	ProductSKU      string
	ProductName     string
	QuantitySold    int64
	SalesTotalMinor int64
}

type TopProductsReport struct {
	StartDate    string
	EndDate      string
	CurrencyCode string
	Products     []TopProduct
}

type StockValuationItem struct {
	ProductID        uint
	SKU              string
	Name             string
	StockOnHand      int64
	CostPriceMinor   int64
	UnitPriceMinor   int64
	CostValueMinor   int64
	RetailValueMinor int64
}

type StockValuation struct {
	Products         []StockValuationItem
	Page             int
	PerPage          int
	TotalProducts    int64
	TotalPages       int
	TotalCostMinor   int64
	TotalRetailMinor int64
	CurrencyCode     string
}

type dailySalesRow struct {
	Date            string
	OrderCount      int64
	SalesTotalMinor int64
}

type dailyPaymentRow struct {
	Date               string
	PaymentsTotalMinor int64
}

type topProductRow struct {
	ProductID       uint
	ProductSKU      string
	ProductName     string
	QuantitySold    int64
	SalesTotalMinor int64
}

func NewReportService(db *gorm.DB, currencyCode string) (*ReportService, error) {
	if db == nil {
		return nil, fmt.Errorf("database is required")
	}
	if len(currencyCode) != 3 {
		return nil, fmt.Errorf("currency code must be a three-letter code")
	}
	return &ReportService{db: db, currencyCode: currencyCode}, nil
}

func (s *ReportService) SalesSummary(startDate, endDate string) (SalesSummary, error) {
	start, end, err := reportDateRange(startDate, endDate)
	if err != nil {
		return SalesSummary{}, err
	}
	var orderRows []dailySalesRow
	if err := s.db.Model(&models.SalesOrder{}).
		Select("date(created_at) AS date, COUNT(*) AS order_count, COALESCE(SUM(total_minor), 0) AS sales_total_minor").
		Where("created_at >= ? AND created_at < ? AND status IN ?", start, end, []models.SalesOrderStatus{
			models.SalesOrderConfirmed, models.SalesOrderPaid,
		}).
		Group("date(created_at)").Order("date(created_at) ASC").Scan(&orderRows).Error; err != nil {
		return SalesSummary{}, fmt.Errorf("query sales summary: %w", err)
	}
	var paymentRows []dailyPaymentRow
	if err := s.db.Model(&models.Payment{}).
		Select("date(paid_at) AS date, COALESCE(SUM(amount_minor), 0) AS payments_total_minor").
		Where("paid_at >= ? AND paid_at < ?", start, end).
		Group("date(paid_at)").Order("date(paid_at) ASC").Scan(&paymentRows).Error; err != nil {
		return SalesSummary{}, fmt.Errorf("query payment summary: %w", err)
	}

	dailyByDate := make(map[string]*DailySalesSummary, len(orderRows)+len(paymentRows))
	for _, row := range orderRows {
		dailyByDate[row.Date] = &DailySalesSummary{
			Date: row.Date, OrderCount: row.OrderCount, SalesTotalMinor: row.SalesTotalMinor,
		}
	}
	for _, row := range paymentRows {
		daily, exists := dailyByDate[row.Date]
		if !exists {
			daily = &DailySalesSummary{Date: row.Date}
			dailyByDate[row.Date] = daily
		}
		daily.PaymentsTotalMinor = row.PaymentsTotalMinor
	}
	daily := make([]DailySalesSummary, 0, len(dailyByDate))
	var result SalesSummary
	result.StartDate, result.EndDate, result.CurrencyCode = startDate, endDate, s.currencyCode
	for _, row := range dailyByDate {
		result.OrderCount += row.OrderCount
		result.SalesTotalMinor += row.SalesTotalMinor
		result.PaymentsTotalMinor += row.PaymentsTotalMinor
		daily = append(daily, *row)
	}
	sortDailySales(daily)
	result.Daily = daily
	return result, nil
}

func (s *ReportService) TopProducts(startDate, endDate string, limit int) (TopProductsReport, error) {
	start, end, err := reportDateRange(startDate, endDate)
	if err != nil {
		return TopProductsReport{}, err
	}
	if limit < 1 || limit > 100 {
		return TopProductsReport{}, ErrInvalidInput
	}
	var rows []topProductRow
	err = s.db.Table("sales_order_items").
		Select("sales_order_items.product_id AS product_id, sales_order_items.product_sku AS product_sku, sales_order_items.product_name AS product_name, SUM(sales_order_items.quantity) AS quantity_sold, SUM(sales_order_items.line_total_minor) AS sales_total_minor").
		Joins("JOIN sales_orders ON sales_orders.id = sales_order_items.sales_order_id").
		Where("sales_orders.created_at >= ? AND sales_orders.created_at < ? AND sales_orders.status IN ?", start, end, []models.SalesOrderStatus{
			models.SalesOrderConfirmed, models.SalesOrderPaid,
		}).
		Group("sales_order_items.product_id, sales_order_items.product_sku, sales_order_items.product_name").
		Order("quantity_sold DESC, sales_total_minor DESC, sales_order_items.product_name ASC").
		Limit(limit).Scan(&rows).Error
	if err != nil {
		return TopProductsReport{}, fmt.Errorf("query top products: %w", err)
	}
	products := make([]TopProduct, 0, len(rows))
	for _, row := range rows {
		products = append(products, TopProduct{
			ProductID: row.ProductID, ProductSKU: row.ProductSKU, ProductName: row.ProductName,
			QuantitySold: row.QuantitySold, SalesTotalMinor: row.SalesTotalMinor,
		})
	}
	return TopProductsReport{
		StartDate: startDate, EndDate: endDate, CurrencyCode: s.currencyCode, Products: products,
	}, nil
}

func (s *ReportService) StockValuation(page, perPage int) (StockValuation, error) {
	if err := validateOrderPage(page, perPage); err != nil {
		return StockValuation{}, err
	}
	var aggregates struct {
		ProductCount int64
		CostTotal    int64
		RetailTotal  int64
	}
	if err := s.db.Model(&models.Product{}).
		Select("COUNT(*) AS product_count, COALESCE(SUM(stock_on_hand * cost_price_minor), 0) AS cost_total, COALESCE(SUM(stock_on_hand * unit_price_minor), 0) AS retail_total").
		Scan(&aggregates).Error; err != nil {
		return StockValuation{}, fmt.Errorf("calculate stock valuation totals: %w", err)
	}
	var products []models.Product
	if err := s.db.Order("name ASC, id ASC").Offset((page - 1) * perPage).Limit(perPage).Find(&products).Error; err != nil {
		return StockValuation{}, fmt.Errorf("list stock valuation products: %w", err)
	}
	items := make([]StockValuationItem, 0, len(products))
	for _, product := range products {
		costValue, ok := multiplyNonNegative(product.StockOnHand, product.CostPriceMinor)
		if !ok {
			return StockValuation{}, ErrInvalidInput
		}
		retailValue, ok := multiplyNonNegative(product.StockOnHand, product.UnitPriceMinor)
		if !ok {
			return StockValuation{}, ErrInvalidInput
		}
		items = append(items, StockValuationItem{
			ProductID: product.ID, SKU: product.SKU, Name: product.Name,
			StockOnHand: product.StockOnHand, CostPriceMinor: product.CostPriceMinor,
			UnitPriceMinor: product.UnitPriceMinor, CostValueMinor: costValue,
			RetailValueMinor: retailValue,
		})
	}
	return StockValuation{
		Products: items, Page: page, PerPage: perPage, TotalProducts: aggregates.ProductCount,
		TotalPages:     pageCount(aggregates.ProductCount, perPage),
		TotalCostMinor: aggregates.CostTotal, TotalRetailMinor: aggregates.RetailTotal,
		CurrencyCode: s.currencyCode,
	}, nil
}

func reportDateRange(startDate, endDate string) (time.Time, time.Time, error) {
	start, err := time.ParseInLocation("2006-01-02", startDate, time.Local)
	if err != nil || start.Format("2006-01-02") != startDate {
		return time.Time{}, time.Time{}, ErrInvalidInput
	}
	endInclusive, err := time.ParseInLocation("2006-01-02", endDate, time.Local)
	if err != nil || endInclusive.Format("2006-01-02") != endDate || endInclusive.Before(start) {
		return time.Time{}, time.Time{}, ErrInvalidInput
	}
	return start, endInclusive.AddDate(0, 0, 1), nil
}

func sortDailySales(daily []DailySalesSummary) {
	sort.Slice(daily, func(i, j int) bool { return daily[i].Date < daily[j].Date })
}

func multiplyNonNegative(left, right int64) (int64, bool) {
	if left < 0 || right < 0 || (right > 0 && left > math.MaxInt64/right) {
		return 0, false
	}
	return left * right, true
}
