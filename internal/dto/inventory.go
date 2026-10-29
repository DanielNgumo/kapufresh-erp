package dto

type StockMovementRequest struct {
	Type     string `json:"type" binding:"required,oneof=in out adjustment"`
	Quantity int64  `json:"quantity" binding:"required"`
	Reason   string `json:"reason" binding:"omitempty,max=500"`
}

type StockMovementResponse struct {
	ID          uint   `json:"id"`
	ProductID   uint   `json:"product_id"`
	ProductSKU  string `json:"product_sku"`
	ProductName string `json:"product_name"`
	Type        string `json:"type"`
	Quantity    int64  `json:"quantity"`
	Reason      string `json:"reason"`
	SourceType  string `json:"source_type"`
	SourceID    *uint  `json:"source_id,omitempty"`
	CreatedByID *uint  `json:"created_by_id,omitempty"`
	CreatedAt   string `json:"created_at"`
}

type StockMovementListResponse struct {
	Movements  []StockMovementResponse `json:"movements"`
	Pagination Pagination              `json:"pagination"`
}

type LowStockProductResponse struct {
	ProductResponse
	StockShortfall int64 `json:"stock_shortfall"`
}

type LowStockListResponse struct {
	Products   []LowStockProductResponse `json:"products"`
	Pagination Pagination                `json:"pagination"`
}
