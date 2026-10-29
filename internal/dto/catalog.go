package dto

type CustomerRequest struct {
	Name    string `json:"name" binding:"required,min=1,max=160"`
	Email   string `json:"email" binding:"omitempty,email,max=254"`
	Phone   string `json:"phone" binding:"omitempty,max=40"`
	Address string `json:"address" binding:"omitempty,max=500"`
	Notes   string `json:"notes" binding:"omitempty,max=2000"`
}

type UpdateCustomerRequest struct {
	Name    *string `json:"name" binding:"omitempty,min=1,max=160"`
	Email   *string `json:"email" binding:"omitempty,email,max=254"`
	Phone   *string `json:"phone" binding:"omitempty,max=40"`
	Address *string `json:"address" binding:"omitempty,max=500"`
	Notes   *string `json:"notes" binding:"omitempty,max=2000"`
}

type CustomerResponse struct {
	ID        uint   `json:"id"`
	Name      string `json:"name"`
	Email     string `json:"email"`
	Phone     string `json:"phone"`
	Address   string `json:"address"`
	Notes     string `json:"notes"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

type CustomerListResponse struct {
	Customers  []CustomerResponse `json:"customers"`
	Pagination Pagination         `json:"pagination"`
}

type SupplierRequest struct {
	Name    string `json:"name" binding:"required,min=1,max=160"`
	Email   string `json:"email" binding:"omitempty,email,max=254"`
	Phone   string `json:"phone" binding:"omitempty,max=40"`
	Address string `json:"address" binding:"omitempty,max=500"`
	Notes   string `json:"notes" binding:"omitempty,max=2000"`
}

type UpdateSupplierRequest struct {
	Name    *string `json:"name" binding:"omitempty,min=1,max=160"`
	Email   *string `json:"email" binding:"omitempty,email,max=254"`
	Phone   *string `json:"phone" binding:"omitempty,max=40"`
	Address *string `json:"address" binding:"omitempty,max=500"`
	Notes   *string `json:"notes" binding:"omitempty,max=2000"`
}

type SupplierResponse struct {
	ID        uint   `json:"id"`
	Name      string `json:"name"`
	Email     string `json:"email"`
	Phone     string `json:"phone"`
	Address   string `json:"address"`
	Notes     string `json:"notes"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

type SupplierListResponse struct {
	Suppliers  []SupplierResponse `json:"suppliers"`
	Pagination Pagination         `json:"pagination"`
}

type ProductRequest struct {
	SKU            string `json:"sku" binding:"required,min=1,max=80"`
	Name           string `json:"name" binding:"required,min=1,max=200"`
	Category       string `json:"category" binding:"omitempty,max=120"`
	Unit           string `json:"unit" binding:"required,min=1,max=40"`
	UnitPriceMinor int64  `json:"unit_price_minor" binding:"gte=0"`
	CostPriceMinor int64  `json:"cost_price_minor" binding:"gte=0"`
	ReorderLevel   int64  `json:"reorder_level" binding:"gte=0"`
}

type UpdateProductRequest struct {
	SKU            *string `json:"sku" binding:"omitempty,min=1,max=80"`
	Name           *string `json:"name" binding:"omitempty,min=1,max=200"`
	Category       *string `json:"category" binding:"omitempty,max=120"`
	Unit           *string `json:"unit" binding:"omitempty,min=1,max=40"`
	UnitPriceMinor *int64  `json:"unit_price_minor" binding:"omitempty,gte=0"`
	CostPriceMinor *int64  `json:"cost_price_minor" binding:"omitempty,gte=0"`
	ReorderLevel   *int64  `json:"reorder_level" binding:"omitempty,gte=0"`
}

type ProductResponse struct {
	ID             uint   `json:"id"`
	SKU            string `json:"sku"`
	Name           string `json:"name"`
	Category       string `json:"category"`
	Unit           string `json:"unit"`
	UnitPriceMinor int64  `json:"unit_price_minor"`
	CostPriceMinor int64  `json:"cost_price_minor"`
	StockOnHand    int64  `json:"stock_on_hand"`
	ReorderLevel   int64  `json:"reorder_level"`
	CreatedAt      string `json:"created_at"`
	UpdatedAt      string `json:"updated_at"`
}

type ProductListResponse struct {
	Products   []ProductResponse `json:"products"`
	Pagination Pagination        `json:"pagination"`
}
