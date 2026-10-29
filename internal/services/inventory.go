package services

import (
	"errors"
	"fmt"
	"math"
	"strings"

	"github.com/yourusername/kapufresh-erp/internal/models"
	"gorm.io/gorm"
)

type InventoryService struct {
	db *gorm.DB
}

type MovementFilter struct {
	ProductID uint
	Type      models.StockMovementType
	Page      int
	PerPage   int
}

type MovementPage struct {
	Items      []models.StockMovement
	Page       int
	PerPage    int
	Total      int64
	TotalPages int
}

type LowStockPage struct {
	Items      []models.Product
	Page       int
	PerPage    int
	Total      int64
	TotalPages int
}

func NewInventoryService(db *gorm.DB) (*InventoryService, error) {
	if db == nil {
		return nil, fmt.Errorf("database is required")
	}
	return &InventoryService{db: db}, nil
}

// RecordMovement takes a signed delta for adjustment, and a positive quantity for in/out.
// The stored movement quantity is always the signed change to stock.
func (s *InventoryService) RecordMovement(productID, actorID uint, movementType models.StockMovementType, quantity int64, reason string) (models.StockMovement, error) {
	if productID == 0 || actorID == 0 || len(strings.TrimSpace(reason)) > 500 {
		return models.StockMovement{}, ErrInvalidInput
	}
	reason = strings.TrimSpace(reason)
	var delta int64
	switch movementType {
	case models.StockMovementIn:
		if quantity <= 0 {
			return models.StockMovement{}, ErrInvalidInput
		}
		delta = quantity
	case models.StockMovementOut:
		if quantity <= 0 || quantity == math.MinInt64 {
			return models.StockMovement{}, ErrInvalidInput
		}
		delta = -quantity
	case models.StockMovementAdjustment:
		if quantity == 0 || quantity == math.MinInt64 {
			return models.StockMovement{}, ErrInvalidInput
		}
		delta = quantity
	default:
		return models.StockMovement{}, ErrInvalidInput
	}

	var movement models.StockMovement
	err := s.db.Transaction(func(tx *gorm.DB) error {
		update := tx.Model(&models.Product{}).
			Where("id = ? AND deleted_at IS NULL", productID)
		switch {
		case delta < 0:
			update = update.Where("stock_on_hand >= ?", -delta)
		case delta > 0:
			update = update.Where("stock_on_hand <= ?", int64(math.MaxInt64)-delta)
		}
		result := update.Update("stock_on_hand", gorm.Expr("stock_on_hand + ?", delta))
		if result.Error != nil {
			return fmt.Errorf("update product stock: %w", result.Error)
		}
		if result.RowsAffected == 0 {
			var product models.Product
			if err := tx.First(&product, productID).Error; err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return ErrNotFound
				}
				return fmt.Errorf("find product after stock update: %w", err)
			}
			return ErrInsufficientStock
		}

		movement = models.StockMovement{
			ProductID: productID, Type: movementType, Quantity: delta,
			Reason: reason, SourceType: "manual", CreatedByID: &actorID,
		}
		if err := tx.Create(&movement).Error; err != nil {
			return fmt.Errorf("create stock movement: %w", err)
		}
		return tx.Preload("Product").First(&movement, movement.ID).Error
	})
	if err != nil {
		return models.StockMovement{}, err
	}
	return movement, nil
}

func (s *InventoryService) ListMovements(filter MovementFilter) (MovementPage, error) {
	if filter.Page < 1 || filter.PerPage < 1 || filter.PerPage > 100 ||
		(filter.Type != "" && filter.Type != models.StockMovementIn &&
			filter.Type != models.StockMovementOut && filter.Type != models.StockMovementAdjustment) {
		return MovementPage{}, ErrInvalidInput
	}
	maxInt := int(^uint(0) >> 1)
	if filter.Page-1 > maxInt/filter.PerPage {
		return MovementPage{}, ErrInvalidInput
	}
	query := s.db.Model(&models.StockMovement{})
	if filter.ProductID > 0 {
		query = query.Where("product_id = ?", filter.ProductID)
	}
	if filter.Type != "" {
		query = query.Where("type = ?", filter.Type)
	}
	var result MovementPage
	result.Page, result.PerPage = filter.Page, filter.PerPage
	if err := query.Count(&result.Total).Error; err != nil {
		return MovementPage{}, fmt.Errorf("count stock movements: %w", err)
	}
	if err := query.Preload("Product", func(db *gorm.DB) *gorm.DB { return db.Unscoped() }).
		Order("created_at DESC, id DESC").
		Offset((filter.Page - 1) * filter.PerPage).Limit(filter.PerPage).Find(&result.Items).Error; err != nil {
		return MovementPage{}, fmt.Errorf("list stock movements: %w", err)
	}
	result.TotalPages = pageCount(result.Total, result.PerPage)
	return result, nil
}

func (s *InventoryService) GetProduct(id uint) (models.Product, error) {
	var product models.Product
	if err := s.db.First(&product, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return product, ErrNotFound
		}
		return product, fmt.Errorf("find product: %w", err)
	}
	return product, nil
}

func (s *InventoryService) ListLowStock(page, perPage int) (LowStockPage, error) {
	if page < 1 || perPage < 1 || perPage > 100 {
		return LowStockPage{}, ErrInvalidInput
	}
	maxInt := int(^uint(0) >> 1)
	if page-1 > maxInt/perPage {
		return LowStockPage{}, ErrInvalidInput
	}
	query := s.db.Model(&models.Product{}).Where("stock_on_hand <= reorder_level")
	var result LowStockPage
	result.Page, result.PerPage = page, perPage
	if err := query.Count(&result.Total).Error; err != nil {
		return LowStockPage{}, fmt.Errorf("count low-stock products: %w", err)
	}
	if err := query.Order("stock_on_hand ASC, name ASC, id ASC").
		Offset((page - 1) * perPage).Limit(perPage).Find(&result.Items).Error; err != nil {
		return LowStockPage{}, fmt.Errorf("list low-stock products: %w", err)
	}
	result.TotalPages = pageCount(result.Total, result.PerPage)
	return result, nil
}
