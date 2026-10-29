package services_test

import (
	"errors"
	"testing"

	"github.com/yourusername/kapufresh-erp/internal/database"
	"github.com/yourusername/kapufresh-erp/internal/models"
	"github.com/yourusername/kapufresh-erp/internal/services"
)

func newTestCommerceService(t *testing.T) (*services.CommerceService, *services.InventoryService, *services.CatalogService, uint) {
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
	if err := database.SeedAdmin(db, "commerce@example.com", "valid-commerce-password", "Commerce Admin"); err != nil {
		t.Fatal(err)
	}
	var admin models.User
	if err := db.Where("email = ?", "commerce@example.com").First(&admin).Error; err != nil {
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
	return commerce, inventory, catalog, admin.ID
}

func TestSalesOrderConfirmationPaymentAndCancellation(t *testing.T) {
	commerce, inventory, catalog, actorID := newTestCommerceService(t)
	customer, err := catalog.CreateCustomer("Retail Customer", "", "", "", "")
	if err != nil {
		t.Fatal(err)
	}
	first, err := catalog.CreateProduct("SALE-001", "Soap", "Household", "each", 250, 100, 0)
	if err != nil {
		t.Fatal(err)
	}
	second, err := catalog.CreateProduct("SALE-002", "Towel", "Household", "each", 500, 300, 0)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := inventory.RecordMovement(first.ID, actorID, models.StockMovementIn, 5, "Initial"); err != nil {
		t.Fatal(err)
	}

	order, err := commerce.CreateSalesOrder(customer.ID, actorID, []services.SalesItemInput{
		{ProductID: first.ID, Quantity: 2}, {ProductID: first.ID, Quantity: 1}, {ProductID: second.ID, Quantity: 1},
	})
	if err != nil {
		t.Fatal(err)
	}
	if order.Status != models.SalesOrderDraft || order.TotalMinor != 1250 || len(order.Items) != 2 {
		t.Fatalf("unexpected draft order: %+v", order)
	}
	firstAfterDraft, err := inventory.GetProduct(first.ID)
	if err != nil {
		t.Fatal(err)
	}
	if firstAfterDraft.StockOnHand != 5 {
		t.Fatalf("draft order changed stock to %d", firstAfterDraft.StockOnHand)
	}

	if _, err := commerce.ConfirmSalesOrder(order.ID, actorID); !errors.Is(err, services.ErrInsufficientStock) {
		t.Fatalf("confirm with insufficient second product error = %v, want insufficient stock", err)
	}
	firstAfterFailedConfirm, err := inventory.GetProduct(first.ID)
	if err != nil {
		t.Fatal(err)
	}
	if firstAfterFailedConfirm.StockOnHand != 5 {
		t.Fatalf("failed confirmation partially deducted stock: %d", firstAfterFailedConfirm.StockOnHand)
	}

	if _, err := inventory.RecordMovement(second.ID, actorID, models.StockMovementIn, 2, "Initial"); err != nil {
		t.Fatal(err)
	}
	order, err = commerce.ConfirmSalesOrder(order.ID, actorID)
	if err != nil {
		t.Fatal(err)
	}
	if order.Status != models.SalesOrderConfirmed {
		t.Fatalf("confirmed status = %q", order.Status)
	}
	firstAfterConfirm, _ := inventory.GetProduct(first.ID)
	secondAfterConfirm, _ := inventory.GetProduct(second.ID)
	if firstAfterConfirm.StockOnHand != 2 || secondAfterConfirm.StockOnHand != 1 {
		t.Fatalf("unexpected confirmed stock: first=%d second=%d", firstAfterConfirm.StockOnHand, secondAfterConfirm.StockOnHand)
	}

	order, err = commerce.RecordManualPayment(order.ID, actorID, 500, models.PaymentMethodCash, "cash-1")
	if err != nil {
		t.Fatal(err)
	}
	if order.Status != models.SalesOrderConfirmed || len(order.Payments) != 1 {
		t.Fatalf("partial payment changed order state unexpectedly: %+v", order)
	}
	if _, err := commerce.RecordManualPayment(order.ID, actorID, 751, models.PaymentMethodBank, "overpay"); !errors.Is(err, services.ErrPaymentExceedsDue) {
		t.Fatalf("overpayment error = %v, want payment exceeds due", err)
	}
	order, err = commerce.RecordManualPayment(order.ID, actorID, 750, models.PaymentMethodBank, "bank-1")
	if err != nil {
		t.Fatal(err)
	}
	if order.Status != models.SalesOrderPaid || len(order.Payments) != 2 {
		t.Fatalf("fully paid order state: %+v", order)
	}
	if _, err := commerce.CancelSalesOrder(order.ID, actorID); !errors.Is(err, services.ErrInvalidState) {
		t.Fatalf("cancel paid order error = %v, want invalid state", err)
	}

	cancelOrder, err := commerce.CreateSalesOrder(customer.ID, actorID, []services.SalesItemInput{{ProductID: first.ID, Quantity: 1}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := commerce.ConfirmSalesOrder(cancelOrder.ID, actorID); err != nil {
		t.Fatal(err)
	}
	cancelOrder, err = commerce.CancelSalesOrder(cancelOrder.ID, actorID)
	if err != nil {
		t.Fatal(err)
	}
	if cancelOrder.Status != models.SalesOrderCancelled {
		t.Fatalf("cancelled status = %q", cancelOrder.Status)
	}
	restored, err := inventory.GetProduct(first.ID)
	if err != nil {
		t.Fatal(err)
	}
	if restored.StockOnHand != 2 {
		t.Fatalf("cancel did not restore stock, remaining=%d", restored.StockOnHand)
	}
	if _, err := commerce.CancelSalesOrder(cancelOrder.ID, actorID); !errors.Is(err, services.ErrInvalidState) {
		t.Fatalf("repeat cancellation error = %v, want invalid state", err)
	}
}

func TestPurchaseOrderPartialReceiptsAreTransactional(t *testing.T) {
	commerce, inventory, catalog, actorID := newTestCommerceService(t)
	supplier, err := catalog.CreateSupplier("Produce Supplier", "", "", "", "")
	if err != nil {
		t.Fatal(err)
	}
	first, err := catalog.CreateProduct("PUR-001", "Flour", "Grocery", "bag", 300, 200, 0)
	if err != nil {
		t.Fatal(err)
	}
	second, err := catalog.CreateProduct("PUR-002", "Sugar", "Grocery", "bag", 400, 250, 0)
	if err != nil {
		t.Fatal(err)
	}
	order, err := commerce.CreatePurchaseOrder(supplier.ID, actorID, []services.PurchaseItemInput{
		{ProductID: first.ID, Quantity: 5, UnitCostMinor: 125},
		{ProductID: second.ID, Quantity: 3, UnitCostMinor: 250},
	})
	if err != nil {
		t.Fatal(err)
	}
	if order.Status != models.PurchaseOrderDraft || order.TotalMinor != 1375 || len(order.Items) != 2 {
		t.Fatalf("unexpected purchase order: %+v", order)
	}

	firstLineID, secondLineID := order.Items[0].ID, order.Items[1].ID
	order, err = commerce.OrderPurchaseOrder(order.ID)
	if err != nil {
		t.Fatal(err)
	}
	if order.Status != models.PurchaseOrderOrdered {
		t.Fatalf("submitted purchase order status = %q", order.Status)
	}
	order, err = commerce.ReceivePurchaseOrder(order.ID, actorID, []services.PurchaseReceiptInput{{ItemID: firstLineID, Quantity: 2}})
	if err != nil {
		t.Fatal(err)
	}
	if order.Status != models.PurchaseOrderOrdered || order.Items[0].ReceivedQuantity != 2 {
		t.Fatalf("unexpected partial receipt state: %+v", order)
	}
	firstStock, _ := inventory.GetProduct(first.ID)
	if firstStock.StockOnHand != 2 {
		t.Fatalf("partial receipt stock = %d, want 2", firstStock.StockOnHand)
	}

	if _, err := commerce.ReceivePurchaseOrder(order.ID, actorID, []services.PurchaseReceiptInput{{ItemID: firstLineID, Quantity: 4}}); !errors.Is(err, services.ErrInvalidInput) {
		t.Fatalf("over-receipt error = %v, want invalid input", err)
	}
	firstAfterRejectedReceipt, _ := inventory.GetProduct(first.ID)
	if firstAfterRejectedReceipt.StockOnHand != 2 {
		t.Fatalf("rejected receipt changed stock to %d", firstAfterRejectedReceipt.StockOnHand)
	}

	order, err = commerce.ReceivePurchaseOrder(order.ID, actorID, []services.PurchaseReceiptInput{
		{ItemID: firstLineID, Quantity: 3}, {ItemID: secondLineID, Quantity: 3},
	})
	if err != nil {
		t.Fatal(err)
	}
	if order.Status != models.PurchaseOrderReceived {
		t.Fatalf("fully received status = %q", order.Status)
	}
	firstStock, _ = inventory.GetProduct(first.ID)
	secondStock, _ := inventory.GetProduct(second.ID)
	if firstStock.StockOnHand != 5 || secondStock.StockOnHand != 3 {
		t.Fatalf("unexpected received stock: first=%d second=%d", firstStock.StockOnHand, secondStock.StockOnHand)
	}
	if _, err := commerce.ReceivePurchaseOrder(order.ID, actorID, []services.PurchaseReceiptInput{{ItemID: secondLineID, Quantity: 1}}); !errors.Is(err, services.ErrInvalidState) {
		t.Fatalf("receive completed order error = %v, want invalid state", err)
	}
	if _, err := commerce.OrderPurchaseOrder(order.ID); !errors.Is(err, services.ErrInvalidState) {
		t.Fatalf("reorder completed purchase error = %v, want invalid state", err)
	}
	history, err := inventory.ListMovements(services.MovementFilter{Page: 1, PerPage: 20})
	if err != nil {
		t.Fatal(err)
	}
	var purchaseMovements int
	for _, movement := range history.Items {
		if movement.SourceType == "purchase_order" && movement.SourceID != nil && *movement.SourceID == order.ID {
			purchaseMovements++
		}
	}
	if purchaseMovements != 3 {
		t.Fatalf("purchase stock movements = %d, want 3", purchaseMovements)
	}
}
