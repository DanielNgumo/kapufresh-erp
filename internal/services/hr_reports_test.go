package services_test

import (
	"errors"
	"testing"
	"time"

	"github.com/yourusername/kapufresh-erp/internal/database"
	"github.com/yourusername/kapufresh-erp/internal/models"
	"github.com/yourusername/kapufresh-erp/internal/services"
	"gorm.io/gorm"
)

func TestEmployeeServiceCRUDAndFiltering(t *testing.T) {
	db := newHRReportTestDB(t)
	service, err := services.NewEmployeeService(db)
	if err != nil {
		t.Fatal(err)
	}

	first, err := service.Create(nil, "  Amina ", "Otieno", "AMINA@example.com", "", "Sales", "Associate", "")
	if err != nil {
		t.Fatal(err)
	}
	if first.FirstName != "Amina" || first.Email != "amina@example.com" || first.Status != models.EmployeeStatusActive {
		t.Fatalf("employee normalization/default status = %+v", first)
	}
	if _, err := service.Create(nil, "", "Name", "", "", "", "", models.EmployeeStatusActive); !errors.Is(err, services.ErrInvalidInput) {
		t.Fatalf("invalid employee error = %v, want invalid input", err)
	}

	second, err := service.Create(nil, "Brian", "Kamau", "", "", "Operations", "Clerk", models.EmployeeStatusInactive)
	if err != nil {
		t.Fatal(err)
	}
	page, err := service.List(services.EmployeeFilter{
		Page: 1, PerPage: 10, Search: "amina", Department: "Sales",
		Status: models.EmployeeStatusActive, Sort: "first_name", Order: "asc",
	})
	if err != nil {
		t.Fatal(err)
	}
	if page.Total != 1 || len(page.Items) != 1 || page.Items[0].ID != first.ID {
		t.Fatalf("employee filters returned %+v", page)
	}

	newPosition := "Senior Associate"
	first, err = service.Update(first.ID, services.EmployeeUpdate{Position: &newPosition})
	if err != nil || first.Position != newPosition {
		t.Fatalf("update employee = %+v, error %v", first, err)
	}
	if err := service.Delete(second.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := service.Get(second.ID); !errors.Is(err, services.ErrNotFound) {
		t.Fatalf("deleted employee error = %v, want not found", err)
	}
}

func TestReportServiceDateRangesTopProductsAndStockValuation(t *testing.T) {
	db := newHRReportTestDB(t)
	catalog, err := services.NewCatalogService(db)
	if err != nil {
		t.Fatal(err)
	}
	reports, err := services.NewReportService(db, "KES")
	if err != nil {
		t.Fatal(err)
	}
	customer, err := catalog.CreateCustomer("Report Customer", "", "", "", "")
	if err != nil {
		t.Fatal(err)
	}
	rice, err := catalog.CreateProduct("RPT-001", "Rice", "Grocery", "kg", 300, 200, 0)
	if err != nil {
		t.Fatal(err)
	}
	tea, err := catalog.CreateProduct("RPT-002", "Tea", "Grocery", "box", 500, 300, 0)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&models.Product{}).Where("id = ?", rice.ID).Update("stock_on_hand", 4).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&models.Product{}).Where("id = ?", tea.ID).Update("stock_on_hand", 2).Error; err != nil {
		t.Fatal(err)
	}

	now := time.Now().In(time.Local)
	year, month, day := now.Date()
	today := time.Date(year, month, day, 0, 0, 0, 0, time.Local)
	date := today.Format("2006-01-02")
	makeOrder := func(number string, status models.SalesOrderStatus, product models.Product, quantity, total int64, createdAt time.Time) uint {
		t.Helper()
		order := models.SalesOrder{
			OrderNumber: number, CustomerID: customer.ID, Status: status,
			CurrencyCode: "KES", SubtotalMinor: total, TotalMinor: total,
			Model: models.Model{CreatedAt: createdAt, UpdatedAt: createdAt},
		}
		if err := db.Create(&order).Error; err != nil {
			t.Fatal(err)
		}
		item := models.SalesOrderItem{
			SalesOrderID: order.ID, ProductID: product.ID, ProductSKU: product.SKU,
			ProductName: product.Name, Quantity: quantity, UnitPriceMinor: total / quantity,
			LineTotalMinor: total,
		}
		if err := db.Create(&item).Error; err != nil {
			t.Fatal(err)
		}
		return order.ID
	}

	makeOrder("RPT-SALE-1", models.SalesOrderConfirmed, rice, 3, 900, today.Add(10*time.Hour))
	paidID := makeOrder("RPT-SALE-2", models.SalesOrderPaid, tea, 1, 500, today.Add(11*time.Hour))
	makeOrder("RPT-SALE-DRAFT", models.SalesOrderDraft, rice, 20, 6000, today.Add(12*time.Hour))
	cancelledID := makeOrder("RPT-SALE-CANCELLED", models.SalesOrderCancelled, tea, 10, 5000, today.Add(13*time.Hour))
	payments := []models.Payment{{
		SalesOrderID: paidID, AmountMinor: 500, Method: models.PaymentMethodCash,
		PaidAt: today.Add(23*time.Hour + 30*time.Minute),
	}, {
		SalesOrderID: cancelledID, AmountMinor: 100, Method: models.PaymentMethodCash,
		PaidAt: today.Add(15 * time.Hour),
	}}
	if err := db.Create(&payments).Error; err != nil {
		t.Fatal(err)
	}
	summary, err := reports.SalesSummary(date, date)
	if err != nil {
		t.Fatal(err)
	}
	if summary.OrderCount != 2 || summary.SalesTotalMinor != 1400 || summary.PaymentsTotalMinor != 600 {
		t.Fatalf("unexpected date-range summary: %+v", summary)
	}
	if len(summary.Daily) != 1 || summary.Daily[0].Date != date || summary.Daily[0].OrderCount != 2 {
		t.Fatalf("unexpected daily summary: %+v", summary.Daily)
	}
	if _, err := reports.SalesSummary(date, "not-a-date"); !errors.Is(err, services.ErrInvalidInput) {
		t.Fatalf("invalid date range error = %v, want invalid input", err)
	}

	top, err := reports.TopProducts(date, date, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(top.Products) != 2 || top.Products[0].ProductID != rice.ID || top.Products[0].QuantitySold != 3 {
		t.Fatalf("unexpected top products: %+v", top.Products)
	}
	if _, err := reports.TopProducts(date, date, 101); !errors.Is(err, services.ErrInvalidInput) {
		t.Fatalf("invalid top-products limit error = %v, want invalid input", err)
	}

	valuation, err := reports.StockValuation(1, 10)
	if err != nil {
		t.Fatal(err)
	}
	if valuation.TotalProducts != 2 || valuation.TotalCostMinor != 1400 || valuation.TotalRetailMinor != 2200 ||
		valuation.CurrencyCode != "KES" || len(valuation.Products) != 2 {
		t.Fatalf("unexpected stock valuation: %+v", valuation)
	}
}

func newHRReportTestDB(t *testing.T) *gorm.DB {
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
	return db
}
