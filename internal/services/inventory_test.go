package services_test

import (
	"errors"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/yourusername/kapufresh-erp/internal/database"
	"github.com/yourusername/kapufresh-erp/internal/models"
	"github.com/yourusername/kapufresh-erp/internal/services"
)

func newTestInventoryService(t *testing.T) (*services.InventoryService, *services.CatalogService, uint) {
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
	if err := database.SeedAdmin(db, "inventory@example.com", "valid-inventory-password", "Inventory Admin"); err != nil {
		t.Fatal(err)
	}
	var admin models.User
	if err := db.Where("email = ?", "inventory@example.com").First(&admin).Error; err != nil {
		t.Fatal(err)
	}
	inventory, err := services.NewInventoryService(db)
	if err != nil {
		t.Fatal(err)
	}
	catalog, err := services.NewCatalogService(db)
	if err != nil {
		t.Fatal(err)
	}
	return inventory, catalog, admin.ID
}

func TestStockMovementTransactionsAndLowStock(t *testing.T) {
	inventory, catalog, actorID := newTestInventoryService(t)
	product, err := catalog.CreateProduct("INV-001", "Rice", "Grocery", "kg", 200, 125, 10)
	if err != nil {
		t.Fatal(err)
	}

	in, err := inventory.RecordMovement(product.ID, actorID, models.StockMovementIn, 8, "Opening delivery")
	if err != nil {
		t.Fatal(err)
	}
	if in.Quantity != 8 || in.Product.StockOnHand != 8 || in.CreatedByID == nil || *in.CreatedByID != actorID {
		t.Fatalf("unexpected inbound movement: %+v", in)
	}
	out, err := inventory.RecordMovement(product.ID, actorID, models.StockMovementOut, 3, "Damaged goods")
	if err != nil {
		t.Fatal(err)
	}
	if out.Quantity != -3 || out.Product.StockOnHand != 5 {
		t.Fatalf("unexpected outbound movement: %+v", out)
	}
	adjustment, err := inventory.RecordMovement(product.ID, actorID, models.StockMovementAdjustment, -2, "Count variance")
	if err != nil {
		t.Fatal(err)
	}
	if adjustment.Quantity != -2 || adjustment.Product.StockOnHand != 3 {
		t.Fatalf("unexpected adjustment movement: %+v", adjustment)
	}
	if _, err := inventory.RecordMovement(product.ID, actorID, models.StockMovementOut, 4, "Too much"); !errors.Is(err, services.ErrInsufficientStock) {
		t.Fatalf("excess outbound movement error = %v, want insufficient stock", err)
	}
	if _, err := inventory.RecordMovement(product.ID, actorID, models.StockMovementAdjustment, 0, ""); !errors.Is(err, services.ErrInvalidInput) {
		t.Fatalf("zero adjustment error = %v, want invalid input", err)
	}

	history, err := inventory.ListMovements(services.MovementFilter{
		ProductID: product.ID, Type: models.StockMovementAdjustment, Page: 1, PerPage: 10,
	})
	if err != nil {
		t.Fatal(err)
	}
	if history.Total != 1 || len(history.Items) != 1 || history.Items[0].Quantity != -2 {
		t.Fatalf("unexpected adjustment history: %+v", history)
	}
	allHistory, err := inventory.ListMovements(services.MovementFilter{Page: 1, PerPage: 10})
	if err != nil {
		t.Fatal(err)
	}
	if allHistory.Total != 3 {
		t.Fatalf("movement count = %d, want 3; rejected movement must not be recorded", allHistory.Total)
	}

	lowStock, err := inventory.ListLowStock(1, 10)
	if err != nil {
		t.Fatal(err)
	}
	if lowStock.Total != 1 || len(lowStock.Items) != 1 || lowStock.Items[0].ID != product.ID {
		t.Fatalf("unexpected low-stock result: %+v", lowStock)
	}
	if err := catalog.DeleteProduct(product.ID); err != nil {
		t.Fatal(err)
	}
	historyAfterDelete, err := inventory.ListMovements(services.MovementFilter{ProductID: product.ID, Page: 1, PerPage: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(historyAfterDelete.Items) != 3 || historyAfterDelete.Items[0].Product.SKU != product.SKU {
		t.Fatalf("movement history lost deleted product details: %+v", historyAfterDelete.Items)
	}
}

func TestConcurrentStockOutDoesNotGoNegative(t *testing.T) {
	inventory, catalog, actorID := newTestInventoryService(t)
	product, err := catalog.CreateProduct("INV-002", "Beans", "Grocery", "kg", 300, 180, 0)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := inventory.RecordMovement(product.ID, actorID, models.StockMovementIn, 5, "Initial stock"); err != nil {
		t.Fatal(err)
	}

	var succeeded atomic.Int32
	var wait sync.WaitGroup
	for i := 0; i < 10; i++ {
		wait.Add(1)
		go func() {
			defer wait.Done()
			_, err := inventory.RecordMovement(product.ID, actorID, models.StockMovementOut, 1, "Concurrent sale")
			if err == nil {
				succeeded.Add(1)
			} else if !errors.Is(err, services.ErrInsufficientStock) {
				t.Errorf("unexpected concurrent movement error: %v", err)
			}
		}()
	}
	wait.Wait()
	if succeeded.Load() != 5 {
		t.Fatalf("successful stock-outs = %d, want 5", succeeded.Load())
	}
	remaining, err := inventory.GetProduct(product.ID)
	if err != nil {
		t.Fatal(err)
	}
	if remaining.StockOnHand != 0 {
		t.Fatalf("stock on hand = %d, want 0", remaining.StockOnHand)
	}
}
