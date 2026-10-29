package services_test

import (
	"errors"
	"testing"

	"github.com/yourusername/kapufresh-erp/internal/database"
	"github.com/yourusername/kapufresh-erp/internal/services"
)

func newTestCatalogService(t *testing.T) *services.CatalogService {
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
	catalog, err := services.NewCatalogService(db)
	if err != nil {
		t.Fatal(err)
	}
	return catalog
}

func TestCustomerCRUDAndSearchPagination(t *testing.T) {
	catalog := newTestCatalogService(t)
	first, err := catalog.CreateCustomer("  Acme Market  ", "BUYER@EXAMPLE.COM", "555-0101", "Market Street", "")
	if err != nil {
		t.Fatal(err)
	}
	if first.Name != "Acme Market" || first.Email != "buyer@example.com" {
		t.Fatalf("customer not normalized: %+v", first)
	}
	if _, err := catalog.CreateCustomer("Another Market", "", "", "", ""); err != nil {
		t.Fatal(err)
	}

	page, err := catalog.ListCustomers(services.CatalogFilter{Page: 1, PerPage: 1, Search: "acme", Sort: "name", Order: "desc"})
	if err != nil {
		t.Fatal(err)
	}
	if page.Total != 1 || len(page.Items) != 1 || page.Items[0].ID != first.ID || page.TotalPages != 1 {
		t.Fatalf("unexpected customer search result: %+v", page)
	}

	name := "Acme Wholesale"
	updated, err := catalog.UpdateCustomer(first.ID, services.CustomerUpdate{Name: &name})
	if err != nil {
		t.Fatal(err)
	}
	if updated.Name != name {
		t.Fatalf("updated name = %q, want %q", updated.Name, name)
	}
	if err := catalog.DeleteCustomer(first.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := catalog.GetCustomer(first.ID); !errors.Is(err, services.ErrNotFound) {
		t.Fatalf("get deleted customer error = %v, want not found", err)
	}
	if _, err := catalog.ListCustomers(services.CatalogFilter{Page: 1, PerPage: 10, Sort: "name; DROP TABLE customers"}); !errors.Is(err, services.ErrInvalidInput) {
		t.Fatalf("invalid sort error = %v, want invalid input", err)
	}
}

func TestSupplierAndProductCRUD(t *testing.T) {
	catalog := newTestCatalogService(t)
	supplier, err := catalog.CreateSupplier("Fresh Farms", "sales@example.com", "", "Farm Road", "")
	if err != nil {
		t.Fatal(err)
	}
	supplierName := "Fresh Produce Farms"
	supplier, err = catalog.UpdateSupplier(supplier.ID, services.SupplierUpdate{Name: &supplierName})
	if err != nil {
		t.Fatal(err)
	}
	if supplier.Name != supplierName {
		t.Fatalf("supplier name = %q, want %q", supplier.Name, supplierName)
	}
	if err := catalog.DeleteSupplier(supplier.ID); err != nil {
		t.Fatal(err)
	}

	product, err := catalog.CreateProduct("VEG-001", "Tomatoes", "Produce", "kg", 250, 140, 5)
	if err != nil {
		t.Fatal(err)
	}
	if product.StockOnHand != 0 {
		t.Fatalf("new product stock = %d, want 0", product.StockOnHand)
	}
	if _, err := catalog.CreateProduct("VEG-001", "Duplicate SKU", "Produce", "kg", 100, 50, 0); !errors.Is(err, services.ErrConflict) {
		t.Fatalf("duplicate SKU error = %v, want conflict", err)
	}

	productPage, err := catalog.ListProducts(services.CatalogFilter{
		Page: 1, PerPage: 10, Search: "tomato", Category: "Produce", Sort: "unit_price_minor",
	})
	if err != nil {
		t.Fatal(err)
	}
	if productPage.Total != 1 || len(productPage.Items) != 1 || productPage.Items[0].ID != product.ID {
		t.Fatalf("unexpected product search result: %+v", productPage)
	}

	price := int64(275)
	product, err = catalog.UpdateProduct(product.ID, services.ProductUpdate{UnitPriceMinor: &price})
	if err != nil {
		t.Fatal(err)
	}
	if product.UnitPriceMinor != price {
		t.Fatalf("unit price = %d, want %d", product.UnitPriceMinor, price)
	}
	negative := int64(-1)
	if _, err := catalog.UpdateProduct(product.ID, services.ProductUpdate{CostPriceMinor: &negative}); !errors.Is(err, services.ErrInvalidInput) {
		t.Fatalf("negative cost error = %v, want invalid input", err)
	}
	if err := catalog.DeleteProduct(product.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := catalog.GetProduct(product.ID); !errors.Is(err, services.ErrNotFound) {
		t.Fatalf("get deleted product error = %v, want not found", err)
	}
}
