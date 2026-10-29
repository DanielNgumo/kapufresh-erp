package services

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/yourusername/kapufresh-erp/internal/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var (
	ErrInvalidState      = errors.New("operation is invalid for this record state")
	ErrPaymentExceedsDue = errors.New("payment exceeds the outstanding balance")
)

type PaymentProvider interface {
	Charge(ctx context.Context, request ProviderPaymentRequest) (ProviderPaymentResult, error)
}

type ProviderPaymentRequest struct {
	OrderNumber  string
	AmountMinor  int64
	CurrencyCode string
	Reference    string
}

type ProviderPaymentResult struct {
	ProviderReference string
}

type CommerceService struct {
	db           *gorm.DB
	currencyCode string
}

type SalesOrderPage struct {
	Items      []models.SalesOrder
	Page       int
	PerPage    int
	Total      int64
	TotalPages int
}

type PurchaseOrderPage struct {
	Items      []models.PurchaseOrder
	Page       int
	PerPage    int
	Total      int64
	TotalPages int
}

type SalesItemInput struct {
	ProductID uint
	Quantity  int64
}

type PurchaseItemInput struct {
	ProductID     uint
	Quantity      int64
	UnitCostMinor int64
}

type PurchaseReceiptInput struct {
	ItemID   uint
	Quantity int64
}

func NewCommerceService(db *gorm.DB, currencyCode string) (*CommerceService, error) {
	currencyCode = strings.ToUpper(strings.TrimSpace(currencyCode))
	if db == nil {
		return nil, fmt.Errorf("database is required")
	}
	if len(currencyCode) != 3 {
		return nil, fmt.Errorf("currency code must be a three-letter code")
	}
	return &CommerceService{db: db, currencyCode: currencyCode}, nil
}

func (s *CommerceService) CreateSalesOrder(customerID, actorID uint, items []SalesItemInput) (models.SalesOrder, error) {
	if customerID == 0 || actorID == 0 || len(items) == 0 {
		return models.SalesOrder{}, ErrInvalidInput
	}
	quantities, err := aggregateQuantities(items)
	if err != nil {
		return models.SalesOrder{}, err
	}
	orderNumber, err := newOrderNumber("SO")
	if err != nil {
		return models.SalesOrder{}, fmt.Errorf("generate sales order number: %w", err)
	}

	var order models.SalesOrder
	err = s.db.Transaction(func(tx *gorm.DB) error {
		var customer models.Customer
		if err := tx.First(&customer, customerID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrNotFound
			}
			return fmt.Errorf("find customer: %w", err)
		}
		order = models.SalesOrder{
			OrderNumber: orderNumber, CustomerID: customerID, Status: models.SalesOrderDraft,
			CurrencyCode: s.currencyCode, CreatedByID: &actorID,
		}
		var lines []models.SalesOrderItem
		productIDs := sortedProductIDs(quantities)
		for _, productID := range productIDs {
			var product models.Product
			if err := tx.First(&product, productID).Error; err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return ErrNotFound
				}
				return fmt.Errorf("find product: %w", err)
			}
			quantity := quantities[productID]
			if product.UnitPriceMinor > 0 && quantity > math.MaxInt64/product.UnitPriceMinor {
				return ErrInvalidInput
			}
			lineTotal := product.UnitPriceMinor * quantity
			if order.TotalMinor > math.MaxInt64-lineTotal {
				return ErrInvalidInput
			}
			order.TotalMinor += lineTotal
			lines = append(lines, models.SalesOrderItem{
				ProductID: product.ID, ProductSKU: product.SKU, ProductName: product.Name,
				Quantity: quantity, UnitPriceMinor: product.UnitPriceMinor, LineTotalMinor: lineTotal,
			})
		}
		order.SubtotalMinor = order.TotalMinor
		if err := tx.Create(&order).Error; err != nil {
			return fmt.Errorf("create sales order: %w", err)
		}
		for i := range lines {
			lines[i].SalesOrderID = order.ID
		}
		if err := tx.Create(&lines).Error; err != nil {
			return fmt.Errorf("create sales order items: %w", err)
		}
		return loadSalesOrder(tx, order.ID, &order)
	})
	if err != nil {
		return models.SalesOrder{}, err
	}
	return order, nil
}

func (s *CommerceService) ConfirmSalesOrder(orderID, actorID uint) (models.SalesOrder, error) {
	if orderID == 0 || actorID == 0 {
		return models.SalesOrder{}, ErrInvalidInput
	}
	var order models.SalesOrder
	err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&order, orderID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrNotFound
			}
			return fmt.Errorf("find sales order: %w", err)
		}
		if order.Status != models.SalesOrderDraft {
			return ErrInvalidState
		}
		var lines []models.SalesOrderItem
		if err := tx.Where("sales_order_id = ?", order.ID).Order("product_id ASC").Find(&lines).Error; err != nil {
			return fmt.Errorf("find sales order items: %w", err)
		}
		if len(lines) == 0 {
			return ErrInvalidState
		}
		for _, line := range lines {
			result := tx.Model(&models.Product{}).
				Where("id = ? AND deleted_at IS NULL AND stock_on_hand >= ?", line.ProductID, line.Quantity).
				Update("stock_on_hand", gorm.Expr("stock_on_hand - ?", line.Quantity))
			if result.Error != nil {
				return fmt.Errorf("deduct product stock: %w", result.Error)
			}
			if result.RowsAffected == 0 {
				var product models.Product
				if err := tx.First(&product, line.ProductID).Error; err != nil {
					if errors.Is(err, gorm.ErrRecordNotFound) {
						return ErrNotFound
					}
					return fmt.Errorf("find product during confirmation: %w", err)
				}
				return ErrInsufficientStock
			}
			orderID := order.ID
			movement := models.StockMovement{
				ProductID: line.ProductID, Type: models.StockMovementOut, Quantity: -line.Quantity,
				Reason: "Sales order " + order.OrderNumber, SourceType: "sales_order",
				SourceID: &orderID, CreatedByID: &actorID,
			}
			if err := tx.Create(&movement).Error; err != nil {
				return fmt.Errorf("create sales stock movement: %w", err)
			}
		}
		if err := tx.Model(&order).Updates(map[string]any{
			"status": models.SalesOrderConfirmed, "updated_at": time.Now(),
		}).Error; err != nil {
			return fmt.Errorf("confirm sales order: %w", err)
		}
		return loadSalesOrder(tx, order.ID, &order)
	})
	if err != nil {
		return models.SalesOrder{}, err
	}
	return order, nil
}

func (s *CommerceService) CancelSalesOrder(orderID, actorID uint) (models.SalesOrder, error) {
	if orderID == 0 || actorID == 0 {
		return models.SalesOrder{}, ErrInvalidInput
	}
	var order models.SalesOrder
	err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&order, orderID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrNotFound
			}
			return fmt.Errorf("find sales order: %w", err)
		}
		if order.Status != models.SalesOrderDraft && order.Status != models.SalesOrderConfirmed {
			return ErrInvalidState
		}
		if order.Status == models.SalesOrderConfirmed {
			var lines []models.SalesOrderItem
			if err := tx.Where("sales_order_id = ?", order.ID).Order("product_id ASC").Find(&lines).Error; err != nil {
				return fmt.Errorf("find sales order items: %w", err)
			}
			for _, line := range lines {
				result := tx.Unscoped().Model(&models.Product{}).
					Where("id = ? AND stock_on_hand <= ?", line.ProductID, int64(math.MaxInt64)-line.Quantity).
					Update("stock_on_hand", gorm.Expr("stock_on_hand + ?", line.Quantity))
				if result.Error != nil {
					return fmt.Errorf("restore product stock: %w", result.Error)
				}
				if result.RowsAffected == 0 {
					return ErrNotFound
				}
				orderID := order.ID
				movement := models.StockMovement{
					ProductID: line.ProductID, Type: models.StockMovementIn, Quantity: line.Quantity,
					Reason: "Cancellation of sales order " + order.OrderNumber, SourceType: "sales_order_cancel",
					SourceID: &orderID, CreatedByID: &actorID,
				}
				if err := tx.Create(&movement).Error; err != nil {
					return fmt.Errorf("create cancellation stock movement: %w", err)
				}
			}
		}
		if err := tx.Model(&order).Update("status", models.SalesOrderCancelled).Error; err != nil {
			return fmt.Errorf("cancel sales order: %w", err)
		}
		return loadSalesOrder(tx, order.ID, &order)
	})
	if err != nil {
		return models.SalesOrder{}, err
	}
	return order, nil
}

func (s *CommerceService) RecordManualPayment(orderID, actorID uint, amount int64, method models.PaymentMethod, reference string) (models.SalesOrder, error) {
	if orderID == 0 || actorID == 0 || amount <= 0 || len(strings.TrimSpace(reference)) > 120 ||
		(method != models.PaymentMethodCash && method != models.PaymentMethodBank && method != models.PaymentMethodOther) {
		return models.SalesOrder{}, ErrInvalidInput
	}
	reference = strings.TrimSpace(reference)
	var order models.SalesOrder
	err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&order, orderID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrNotFound
			}
			return fmt.Errorf("find sales order: %w", err)
		}
		if order.Status != models.SalesOrderConfirmed && order.Status != models.SalesOrderPaid {
			return ErrInvalidState
		}
		var paid int64
		if err := tx.Model(&models.Payment{}).Where("sales_order_id = ?", order.ID).Select("COALESCE(SUM(amount_minor), 0)").Scan(&paid).Error; err != nil {
			return fmt.Errorf("sum payments: %w", err)
		}
		if paid < 0 || paid > order.TotalMinor || amount > order.TotalMinor-paid {
			return ErrPaymentExceedsDue
		}
		payment := models.Payment{
			SalesOrderID: order.ID, AmountMinor: amount, Method: method, Reference: reference,
			PaidAt: time.Now(), CreatedByID: &actorID,
		}
		if err := tx.Create(&payment).Error; err != nil {
			return fmt.Errorf("record payment: %w", err)
		}
		if paid+amount == order.TotalMinor {
			if err := tx.Model(&order).Update("status", models.SalesOrderPaid).Error; err != nil {
				return fmt.Errorf("mark sales order paid: %w", err)
			}
		}
		return loadSalesOrder(tx, order.ID, &order)
	})
	if err != nil {
		return models.SalesOrder{}, err
	}
	return order, nil
}

func (s *CommerceService) GetSalesOrder(id uint) (models.SalesOrder, error) {
	var order models.SalesOrder
	if err := loadSalesOrder(s.db, id, &order); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return order, ErrNotFound
		}
		return order, fmt.Errorf("find sales order: %w", err)
	}
	return order, nil
}

func (s *CommerceService) ListSalesOrders(page, perPage int, status models.SalesOrderStatus) (SalesOrderPage, error) {
	if err := validateOrderPage(page, perPage); err != nil {
		return SalesOrderPage{}, err
	}
	if status != "" && !validSalesStatus(status) {
		return SalesOrderPage{}, ErrInvalidInput
	}
	query := s.db.Model(&models.SalesOrder{})
	if status != "" {
		query = query.Where("status = ?", status)
	}
	var result SalesOrderPage
	result.Page, result.PerPage = page, perPage
	if err := query.Count(&result.Total).Error; err != nil {
		return SalesOrderPage{}, fmt.Errorf("count sales orders: %w", err)
	}
	var orders []models.SalesOrder
	if err := query.Order("created_at DESC, id DESC").Offset((page - 1) * perPage).Limit(perPage).Find(&orders).Error; err != nil {
		return SalesOrderPage{}, fmt.Errorf("list sales orders: %w", err)
	}
	result.Items = make([]models.SalesOrder, 0, len(orders))
	for _, order := range orders {
		if err := loadSalesOrder(s.db, order.ID, &order); err != nil {
			return SalesOrderPage{}, fmt.Errorf("load sales order details: %w", err)
		}
		result.Items = append(result.Items, order)
	}
	result.TotalPages = pageCount(result.Total, perPage)
	return result, nil
}

func (s *CommerceService) CreatePurchaseOrder(supplierID, actorID uint, items []PurchaseItemInput) (models.PurchaseOrder, error) {
	if supplierID == 0 || actorID == 0 || len(items) == 0 {
		return models.PurchaseOrder{}, ErrInvalidInput
	}
	aggregated, err := aggregatePurchaseItems(items)
	if err != nil {
		return models.PurchaseOrder{}, err
	}
	orderNumber, err := newOrderNumber("PO")
	if err != nil {
		return models.PurchaseOrder{}, fmt.Errorf("generate purchase order number: %w", err)
	}
	var order models.PurchaseOrder
	err = s.db.Transaction(func(tx *gorm.DB) error {
		var supplier models.Supplier
		if err := tx.First(&supplier, supplierID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrNotFound
			}
			return fmt.Errorf("find supplier: %w", err)
		}
		order = models.PurchaseOrder{
			OrderNumber: orderNumber, SupplierID: supplierID, Status: models.PurchaseOrderDraft,
			CurrencyCode: s.currencyCode, CreatedByID: &actorID,
		}
		var lines []models.PurchaseOrderItem
		productIDs := sortedProductIDs(aggregated)
		for _, productID := range productIDs {
			var product models.Product
			if err := tx.First(&product, productID).Error; err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return ErrNotFound
				}
				return fmt.Errorf("find purchase product: %w", err)
			}
			input := aggregated[productID]
			if input.UnitCostMinor > 0 && input.Quantity > math.MaxInt64/input.UnitCostMinor {
				return ErrInvalidInput
			}
			lineTotal := input.UnitCostMinor * input.Quantity
			if order.TotalMinor > math.MaxInt64-lineTotal {
				return ErrInvalidInput
			}
			order.TotalMinor += lineTotal
			lines = append(lines, models.PurchaseOrderItem{
				ProductID: product.ID, Quantity: input.Quantity, UnitCostMinor: input.UnitCostMinor,
			})
		}
		if err := tx.Create(&order).Error; err != nil {
			return fmt.Errorf("create purchase order: %w", err)
		}
		for i := range lines {
			lines[i].PurchaseOrderID = order.ID
		}
		if err := tx.Create(&lines).Error; err != nil {
			return fmt.Errorf("create purchase order items: %w", err)
		}
		return loadPurchaseOrder(tx, order.ID, &order)
	})
	if err != nil {
		return models.PurchaseOrder{}, err
	}
	return order, nil
}

func (s *CommerceService) ReceivePurchaseOrder(orderID, actorID uint, receipts []PurchaseReceiptInput) (models.PurchaseOrder, error) {
	if orderID == 0 || actorID == 0 || len(receipts) == 0 {
		return models.PurchaseOrder{}, ErrInvalidInput
	}
	receiptMap := make(map[uint]int64, len(receipts))
	for _, receipt := range receipts {
		if receipt.ItemID == 0 || receipt.Quantity <= 0 || receiptMap[receipt.ItemID] > math.MaxInt64-receipt.Quantity {
			return models.PurchaseOrder{}, ErrInvalidInput
		}
		receiptMap[receipt.ItemID] += receipt.Quantity
	}
	var order models.PurchaseOrder
	err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&order, orderID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrNotFound
			}
			return fmt.Errorf("find purchase order: %w", err)
		}
		if order.Status != models.PurchaseOrderOrdered {
			return ErrInvalidState
		}
		var lines []models.PurchaseOrderItem
		if err := tx.Where("purchase_order_id = ?", order.ID).Order("product_id ASC").Find(&lines).Error; err != nil {
			return fmt.Errorf("find purchase order items: %w", err)
		}
		if len(lines) == 0 {
			return ErrInvalidState
		}
		seen := make(map[uint]bool, len(lines))
		orderID := order.ID
		for i := range lines {
			line := &lines[i]
			seen[line.ID] = true
			quantity := receiptMap[line.ID]
			if quantity == 0 {
				continue
			}
			if line.ReceivedQuantity > line.Quantity || quantity > line.Quantity-line.ReceivedQuantity {
				return ErrInvalidInput
			}
			result := tx.Model(&models.Product{}).
				Where("id = ? AND deleted_at IS NULL AND stock_on_hand <= ?", line.ProductID, int64(math.MaxInt64)-quantity).
				Update("stock_on_hand", gorm.Expr("stock_on_hand + ?", quantity))
			if result.Error != nil {
				return fmt.Errorf("increase received product stock: %w", result.Error)
			}
			if result.RowsAffected == 0 {
				return ErrNotFound
			}
			if err := tx.Model(line).Update("received_quantity", line.ReceivedQuantity+quantity).Error; err != nil {
				return fmt.Errorf("update received purchase quantity: %w", err)
			}
			movement := models.StockMovement{
				ProductID: line.ProductID, Type: models.StockMovementIn, Quantity: quantity,
				Reason: "Receipt for purchase order " + order.OrderNumber, SourceType: "purchase_order",
				SourceID: &orderID, CreatedByID: &actorID,
			}
			if err := tx.Create(&movement).Error; err != nil {
				return fmt.Errorf("create purchase receipt stock movement: %w", err)
			}
		}
		for itemID := range receiptMap {
			if !seen[itemID] {
				return ErrInvalidInput
			}
		}
		status := models.PurchaseOrderOrdered
		allReceived := true
		for _, line := range lines {
			received := line.ReceivedQuantity + receiptMap[line.ID]
			if received < line.Quantity {
				allReceived = false
				break
			}
		}
		if allReceived {
			status = models.PurchaseOrderReceived
		}
		if err := tx.Model(&order).Update("status", status).Error; err != nil {
			return fmt.Errorf("update purchase order status: %w", err)
		}
		return loadPurchaseOrder(tx, order.ID, &order)
	})
	if err != nil {
		return models.PurchaseOrder{}, err
	}
	return order, nil
}

func (s *CommerceService) OrderPurchaseOrder(orderID uint) (models.PurchaseOrder, error) {
	if orderID == 0 {
		return models.PurchaseOrder{}, ErrInvalidInput
	}
	var order models.PurchaseOrder
	err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&order, orderID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrNotFound
			}
			return fmt.Errorf("find purchase order: %w", err)
		}
		if order.Status != models.PurchaseOrderDraft {
			return ErrInvalidState
		}
		if err := tx.Model(&order).Update("status", models.PurchaseOrderOrdered).Error; err != nil {
			return fmt.Errorf("submit purchase order: %w", err)
		}
		return loadPurchaseOrder(tx, order.ID, &order)
	})
	if err != nil {
		return models.PurchaseOrder{}, err
	}
	return order, nil
}

func (s *CommerceService) ListPurchaseOrders(page, perPage int, status models.PurchaseOrderStatus) (PurchaseOrderPage, error) {
	if err := validateOrderPage(page, perPage); err != nil {
		return PurchaseOrderPage{}, err
	}
	if status != "" && !validPurchaseStatus(status) {
		return PurchaseOrderPage{}, ErrInvalidInput
	}
	query := s.db.Model(&models.PurchaseOrder{})
	if status != "" {
		query = query.Where("status = ?", status)
	}
	var result PurchaseOrderPage
	result.Page, result.PerPage = page, perPage
	if err := query.Count(&result.Total).Error; err != nil {
		return PurchaseOrderPage{}, fmt.Errorf("count purchase orders: %w", err)
	}
	var orders []models.PurchaseOrder
	if err := query.Order("created_at DESC, id DESC").Offset((page - 1) * perPage).Limit(perPage).Find(&orders).Error; err != nil {
		return PurchaseOrderPage{}, fmt.Errorf("list purchase orders: %w", err)
	}
	result.Items = make([]models.PurchaseOrder, 0, len(orders))
	for _, order := range orders {
		if err := loadPurchaseOrder(s.db, order.ID, &order); err != nil {
			return PurchaseOrderPage{}, fmt.Errorf("load purchase order details: %w", err)
		}
		result.Items = append(result.Items, order)
	}
	result.TotalPages = pageCount(result.Total, perPage)
	return result, nil
}

func (s *CommerceService) GetPurchaseOrder(id uint) (models.PurchaseOrder, error) {
	var order models.PurchaseOrder
	if err := loadPurchaseOrder(s.db, id, &order); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return order, ErrNotFound
		}
		return order, fmt.Errorf("find purchase order: %w", err)
	}
	return order, nil
}

func aggregateQuantities(items []SalesItemInput) (map[uint]int64, error) {
	quantities := make(map[uint]int64, len(items))
	for _, item := range items {
		if item.ProductID == 0 || item.Quantity <= 0 || quantities[item.ProductID] > math.MaxInt64-item.Quantity {
			return nil, ErrInvalidInput
		}
		quantities[item.ProductID] += item.Quantity
	}
	return quantities, nil
}

type purchaseAggregate struct {
	Quantity      int64
	UnitCostMinor int64
}

func aggregatePurchaseItems(items []PurchaseItemInput) (map[uint]purchaseAggregate, error) {
	aggregated := make(map[uint]purchaseAggregate, len(items))
	for _, item := range items {
		if item.ProductID == 0 || item.Quantity <= 0 || item.UnitCostMinor < 0 {
			return nil, ErrInvalidInput
		}
		current, exists := aggregated[item.ProductID]
		if exists && current.UnitCostMinor != item.UnitCostMinor {
			return nil, ErrInvalidInput
		}
		if current.Quantity > math.MaxInt64-item.Quantity {
			return nil, ErrInvalidInput
		}
		current.Quantity += item.Quantity
		current.UnitCostMinor = item.UnitCostMinor
		aggregated[item.ProductID] = current
	}
	return aggregated, nil
}

func sortedProductIDs[T any](items map[uint]T) []uint {
	ids := make([]uint, 0, len(items))
	for id := range items {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	return ids
}

func newOrderNumber(prefix string) (string, error) {
	random := make([]byte, 12)
	if _, err := rand.Read(random); err != nil {
		return "", err
	}
	return prefix + "-" + strings.ToUpper(hex.EncodeToString(random)), nil
}

func loadSalesOrder(db *gorm.DB, id uint, order *models.SalesOrder) error {
	return db.Preload("Customer").Preload("Items").Preload("Payments").
		First(order, id).Error
}

func loadPurchaseOrder(db *gorm.DB, id uint, order *models.PurchaseOrder) error {
	return db.Preload("Supplier").Preload("Items").Preload("Items.Product").
		First(order, id).Error
}

func validateOrderPage(page, perPage int) error {
	if page < 1 || perPage < 1 || perPage > 100 {
		return ErrInvalidInput
	}
	maxInt := int(^uint(0) >> 1)
	if page-1 > maxInt/perPage {
		return ErrInvalidInput
	}
	return nil
}

func validSalesStatus(status models.SalesOrderStatus) bool {
	return status == models.SalesOrderDraft || status == models.SalesOrderConfirmed ||
		status == models.SalesOrderPaid || status == models.SalesOrderCancelled
}

func validPurchaseStatus(status models.PurchaseOrderStatus) bool {
	return status == models.PurchaseOrderDraft || status == models.PurchaseOrderOrdered ||
		status == models.PurchaseOrderReceived || status == models.PurchaseOrderCancelled
}
