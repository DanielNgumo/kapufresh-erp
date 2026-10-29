package dto

type SalesOrderItemRequest struct {
	ProductID uint  `json:"product_id" binding:"required"`
	Quantity  int64 `json:"quantity" binding:"required,gt=0"`
}

type CreateSalesOrderRequest struct {
	CustomerID uint                    `json:"customer_id" binding:"required"`
	Items      []SalesOrderItemRequest `json:"items" binding:"required,min=1,dive"`
}

type RecordPaymentRequest struct {
	AmountMinor int64  `json:"amount_minor" binding:"required,gt=0"`
	Method      string `json:"method" binding:"required,oneof=cash bank other"`
	Reference   string `json:"reference" binding:"omitempty,max=120"`
}

type PurchaseOrderItemRequest struct {
	ProductID     uint  `json:"product_id" binding:"required"`
	Quantity      int64 `json:"quantity" binding:"required,gt=0"`
	UnitCostMinor int64 `json:"unit_cost_minor" binding:"gte=0"`
}

type CreatePurchaseOrderRequest struct {
	SupplierID uint                       `json:"supplier_id" binding:"required"`
	Items      []PurchaseOrderItemRequest `json:"items" binding:"required,min=1,dive"`
}

type ReceivePurchaseItemRequest struct {
	PurchaseOrderItemID uint  `json:"purchase_order_item_id" binding:"required"`
	Quantity            int64 `json:"quantity" binding:"required,gt=0"`
}

type ReceivePurchaseOrderRequest struct {
	Items []ReceivePurchaseItemRequest `json:"items" binding:"required,min=1,dive"`
}

type SalesOrderItemResponse struct {
	ID             uint   `json:"id"`
	ProductID      uint   `json:"product_id"`
	ProductSKU     string `json:"product_sku"`
	ProductName    string `json:"product_name"`
	Quantity       int64  `json:"quantity"`
	UnitPriceMinor int64  `json:"unit_price_minor"`
	LineTotalMinor int64  `json:"line_total_minor"`
}

type PaymentResponse struct {
	ID          uint   `json:"id"`
	AmountMinor int64  `json:"amount_minor"`
	Method      string `json:"method"`
	Reference   string `json:"reference"`
	PaidAt      string `json:"paid_at"`
	CreatedByID *uint  `json:"created_by_id,omitempty"`
}

type SalesOrderResponse struct {
	ID            uint                     `json:"id"`
	OrderNumber   string                   `json:"order_number"`
	CustomerID    uint                     `json:"customer_id"`
	CustomerName  string                   `json:"customer_name"`
	Status        string                   `json:"status"`
	CurrencyCode  string                   `json:"currency_code"`
	SubtotalMinor int64                    `json:"subtotal_minor"`
	TotalMinor    int64                    `json:"total_minor"`
	PaidMinor     int64                    `json:"paid_minor"`
	Items         []SalesOrderItemResponse `json:"items"`
	Payments      []PaymentResponse        `json:"payments"`
	CreatedAt     string                   `json:"created_at"`
	UpdatedAt     string                   `json:"updated_at"`
}

type SalesOrderListResponse struct {
	Orders     []SalesOrderResponse `json:"orders"`
	Pagination Pagination           `json:"pagination"`
}

type PurchaseOrderItemResponse struct {
	ID               uint   `json:"id"`
	ProductID        uint   `json:"product_id"`
	ProductSKU       string `json:"product_sku"`
	ProductName      string `json:"product_name"`
	Quantity         int64  `json:"quantity"`
	ReceivedQuantity int64  `json:"received_quantity"`
	UnitCostMinor    int64  `json:"unit_cost_minor"`
	LineTotalMinor   int64  `json:"line_total_minor"`
}

type PurchaseOrderResponse struct {
	ID           uint                        `json:"id"`
	OrderNumber  string                      `json:"order_number"`
	SupplierID   uint                        `json:"supplier_id"`
	SupplierName string                      `json:"supplier_name"`
	Status       string                      `json:"status"`
	CurrencyCode string                      `json:"currency_code"`
	TotalMinor   int64                       `json:"total_minor"`
	Items        []PurchaseOrderItemResponse `json:"items"`
	CreatedAt    string                      `json:"created_at"`
	UpdatedAt    string                      `json:"updated_at"`
}

type PurchaseOrderListResponse struct {
	Orders     []PurchaseOrderResponse `json:"orders"`
	Pagination Pagination              `json:"pagination"`
}
