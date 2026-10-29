package dto

type EmployeeRequest struct {
	UserID     *uint  `json:"user_id"`
	FirstName  string `json:"first_name" binding:"required,min=1,max=120"`
	LastName   string `json:"last_name" binding:"required,min=1,max=120"`
	Email      string `json:"email" binding:"omitempty,email,max=254"`
	Phone      string `json:"phone" binding:"omitempty,max=40"`
	Department string `json:"department" binding:"omitempty,max=120"`
	Position   string `json:"position" binding:"omitempty,max=120"`
	Status     string `json:"status" binding:"omitempty,oneof=active inactive"`
}

type UpdateEmployeeRequest struct {
	FirstName  *string `json:"first_name" binding:"omitempty,min=1,max=120"`
	LastName   *string `json:"last_name" binding:"omitempty,min=1,max=120"`
	Email      *string `json:"email" binding:"omitempty,email,max=254"`
	Phone      *string `json:"phone" binding:"omitempty,max=40"`
	Department *string `json:"department" binding:"omitempty,max=120"`
	Position   *string `json:"position" binding:"omitempty,max=120"`
	Status     *string `json:"status" binding:"omitempty,oneof=active inactive"`
}

type EmployeeResponse struct {
	ID         uint   `json:"id"`
	UserID     *uint  `json:"user_id,omitempty"`
	FirstName  string `json:"first_name"`
	LastName   string `json:"last_name"`
	Email      string `json:"email"`
	Phone      string `json:"phone"`
	Department string `json:"department"`
	Position   string `json:"position"`
	Status     string `json:"status"`
	CreatedAt  string `json:"created_at"`
	UpdatedAt  string `json:"updated_at"`
}

type EmployeeListResponse struct {
	Employees  []EmployeeResponse `json:"employees"`
	Pagination Pagination         `json:"pagination"`
}

type SalesSummaryResponse struct {
	StartDate          string              `json:"start_date"`
	EndDate            string              `json:"end_date"`
	CurrencyCode       string              `json:"currency_code"`
	OrderCount         int64               `json:"order_count"`
	SalesTotalMinor    int64               `json:"sales_total_minor"`
	PaymentsTotalMinor int64               `json:"payments_total_minor"`
	Daily              []DailySalesSummary `json:"daily"`
}

type DailySalesSummary struct {
	Date               string `json:"date"`
	OrderCount         int64  `json:"order_count"`
	SalesTotalMinor    int64  `json:"sales_total_minor"`
	PaymentsTotalMinor int64  `json:"payments_total_minor"`
}

type TopProductResponse struct {
	ProductID       uint   `json:"product_id"`
	ProductSKU      string `json:"product_sku"`
	ProductName     string `json:"product_name"`
	QuantitySold    int64  `json:"quantity_sold"`
	SalesTotalMinor int64  `json:"sales_total_minor"`
}

type TopProductsReportResponse struct {
	StartDate    string               `json:"start_date"`
	EndDate      string               `json:"end_date"`
	CurrencyCode string               `json:"currency_code"`
	Products     []TopProductResponse `json:"products"`
}

type StockValuationProduct struct {
	ProductID        uint   `json:"product_id"`
	SKU              string `json:"sku"`
	Name             string `json:"name"`
	StockOnHand      int64  `json:"stock_on_hand"`
	CostPriceMinor   int64  `json:"cost_price_minor"`
	UnitPriceMinor   int64  `json:"unit_price_minor"`
	CostValueMinor   int64  `json:"cost_value_minor"`
	RetailValueMinor int64  `json:"retail_value_minor"`
}

type StockValuationResponse struct {
	Products         []StockValuationProduct `json:"products"`
	Pagination       Pagination              `json:"pagination"`
	CurrencyCode     string                  `json:"currency_code"`
	TotalCostMinor   int64                   `json:"total_cost_minor"`
	TotalRetailMinor int64                   `json:"total_retail_minor"`
}
