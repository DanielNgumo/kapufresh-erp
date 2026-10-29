package models

import (
	"time"

	"gorm.io/gorm"
)

type UserRole string

const (
	RoleAdmin   UserRole = "admin"
	RoleManager UserRole = "manager"
	RoleStaff   UserRole = "staff"
)

type StockMovementType string

const (
	StockMovementIn         StockMovementType = "in"
	StockMovementOut        StockMovementType = "out"
	StockMovementAdjustment StockMovementType = "adjustment"
)

type SalesOrderStatus string

const (
	SalesOrderDraft     SalesOrderStatus = "draft"
	SalesOrderConfirmed SalesOrderStatus = "confirmed"
	SalesOrderPaid      SalesOrderStatus = "paid"
	SalesOrderCancelled SalesOrderStatus = "cancelled"
)

type PaymentMethod string

const (
	PaymentMethodCash  PaymentMethod = "cash"
	PaymentMethodBank  PaymentMethod = "bank"
	PaymentMethodOther PaymentMethod = "other"
)

type PurchaseOrderStatus string

const (
	PurchaseOrderDraft     PurchaseOrderStatus = "draft"
	PurchaseOrderOrdered   PurchaseOrderStatus = "ordered"
	PurchaseOrderReceived  PurchaseOrderStatus = "received"
	PurchaseOrderCancelled PurchaseOrderStatus = "cancelled"
)

type EmployeeStatus string

const (
	EmployeeStatusActive   EmployeeStatus = "active"
	EmployeeStatusInactive EmployeeStatus = "inactive"
)

type Model struct {
	ID        uint `gorm:"primaryKey"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

type SoftDeleteModel struct {
	Model
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

type User struct {
	SoftDeleteModel
	Name          string         `gorm:"size:120;not null"`
	Email         string         `gorm:"size:254;not null;uniqueIndex"`
	PasswordHash  string         `gorm:"size:255;not null"`
	Role          UserRole       `gorm:"size:20;not null;default:staff;index"`
	IsActive      bool           `gorm:"not null;default:true"`
	RefreshTokens []RefreshToken `gorm:"foreignKey:UserID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT;" json:"-"`
}

type RefreshToken struct {
	Model
	UserID    uint      `gorm:"not null;index"`
	User      User      `gorm:"foreignKey:UserID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT;" json:"-"`
	TokenHash string    `gorm:"size:64;not null;uniqueIndex"`
	ExpiresAt time.Time `gorm:"not null;index"`
	RevokedAt *time.Time
}

type Customer struct {
	SoftDeleteModel
	Name    string `gorm:"size:160;not null;index"`
	Email   string `gorm:"size:254;index"`
	Phone   string `gorm:"size:40;index"`
	Address string `gorm:"size:500"`
	Notes   string `gorm:"size:2000"`
}

type Supplier struct {
	SoftDeleteModel
	Name    string `gorm:"size:160;not null;index"`
	Email   string `gorm:"size:254;index"`
	Phone   string `gorm:"size:40;index"`
	Address string `gorm:"size:500"`
	Notes   string `gorm:"size:2000"`
}

type Product struct {
	SoftDeleteModel
	SKU            string `gorm:"size:80;not null;uniqueIndex"`
	Name           string `gorm:"size:200;not null;index"`
	Category       string `gorm:"size:120;index"`
	Unit           string `gorm:"size:40;not null"`
	UnitPriceMinor int64  `gorm:"not null"`
	CostPriceMinor int64  `gorm:"not null"`
	StockOnHand    int64  `gorm:"not null;default:0"`
	ReorderLevel   int64  `gorm:"not null;default:0"`
}

type StockMovement struct {
	Model
	ProductID   uint              `gorm:"not null;index"`
	Product     Product           `gorm:"foreignKey:ProductID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT;" json:"-"`
	Type        StockMovementType `gorm:"size:20;not null;index"`
	Quantity    int64             `gorm:"not null"`
	Reason      string            `gorm:"size:500"`
	SourceType  string            `gorm:"size:40;index"`
	SourceID    *uint             `gorm:"index"`
	CreatedByID *uint             `gorm:"index"`
	CreatedBy   *User             `gorm:"foreignKey:CreatedByID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;" json:"-"`
}

type SalesOrder struct {
	Model
	OrderNumber   string           `gorm:"size:40;not null;uniqueIndex"`
	CustomerID    uint             `gorm:"not null;index"`
	Customer      Customer         `gorm:"foreignKey:CustomerID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT;" json:"-"`
	Status        SalesOrderStatus `gorm:"size:20;not null;default:draft;index"`
	CurrencyCode  string           `gorm:"size:3;not null"`
	SubtotalMinor int64            `gorm:"not null"`
	TotalMinor    int64            `gorm:"not null"`
	CreatedByID   *uint            `gorm:"index"`
	CreatedBy     *User            `gorm:"foreignKey:CreatedByID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;" json:"-"`
	Items         []SalesOrderItem `gorm:"foreignKey:SalesOrderID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT;" json:"-"`
	Payments      []Payment        `gorm:"foreignKey:SalesOrderID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT;" json:"-"`
}

type SalesOrderItem struct {
	Model
	SalesOrderID   uint       `gorm:"not null;index"`
	SalesOrder     SalesOrder `gorm:"foreignKey:SalesOrderID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT;" json:"-"`
	ProductID      uint       `gorm:"not null;index"`
	Product        Product    `gorm:"foreignKey:ProductID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT;" json:"-"`
	ProductSKU     string     `gorm:"size:80;not null"`
	ProductName    string     `gorm:"size:200;not null"`
	Quantity       int64      `gorm:"not null"`
	UnitPriceMinor int64      `gorm:"not null"`
	LineTotalMinor int64      `gorm:"not null"`
}

type Payment struct {
	Model
	SalesOrderID uint          `gorm:"not null;index"`
	SalesOrder   SalesOrder    `gorm:"foreignKey:SalesOrderID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT;" json:"-"`
	AmountMinor  int64         `gorm:"not null"`
	Method       PaymentMethod `gorm:"size:20;not null"`
	Reference    string        `gorm:"size:120"`
	PaidAt       time.Time     `gorm:"not null;index"`
	CreatedByID  *uint         `gorm:"index"`
	CreatedBy    *User         `gorm:"foreignKey:CreatedByID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;" json:"-"`
}

type PurchaseOrder struct {
	Model
	OrderNumber  string              `gorm:"size:40;not null;uniqueIndex"`
	SupplierID   uint                `gorm:"not null;index"`
	Supplier     Supplier            `gorm:"foreignKey:SupplierID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT;" json:"-"`
	Status       PurchaseOrderStatus `gorm:"size:20;not null;default:draft;index"`
	CurrencyCode string              `gorm:"size:3;not null"`
	TotalMinor   int64               `gorm:"not null"`
	CreatedByID  *uint               `gorm:"index"`
	CreatedBy    *User               `gorm:"foreignKey:CreatedByID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;" json:"-"`
	Items        []PurchaseOrderItem `gorm:"foreignKey:PurchaseOrderID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT;" json:"-"`
}

type PurchaseOrderItem struct {
	Model
	PurchaseOrderID  uint          `gorm:"not null;index"`
	PurchaseOrder    PurchaseOrder `gorm:"foreignKey:PurchaseOrderID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT;" json:"-"`
	ProductID        uint          `gorm:"not null;index"`
	Product          Product       `gorm:"foreignKey:ProductID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT;" json:"-"`
	Quantity         int64         `gorm:"not null"`
	ReceivedQuantity int64         `gorm:"not null;default:0"`
	UnitCostMinor    int64         `gorm:"not null"`
}

type Employee struct {
	SoftDeleteModel
	UserID     *uint          `gorm:"uniqueIndex"`
	User       *User          `gorm:"foreignKey:UserID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;" json:"-"`
	FirstName  string         `gorm:"size:120;not null"`
	LastName   string         `gorm:"size:120;not null"`
	Email      string         `gorm:"size:254;index"`
	Phone      string         `gorm:"size:40"`
	Department string         `gorm:"size:120;index"`
	Position   string         `gorm:"size:120"`
	Status     EmployeeStatus `gorm:"size:20;not null;default:active;index"`
}
