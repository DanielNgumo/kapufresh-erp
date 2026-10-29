package services

import (
	"errors"
	"fmt"
	"net/mail"
	"strings"

	"github.com/yourusername/kapufresh-erp/internal/models"
	"gorm.io/gorm"
)

type CatalogService struct {
	db *gorm.DB
}

type CatalogFilter struct {
	Page     int
	PerPage  int
	Search   string
	Category string
	Sort     string
	Order    string
}

type CustomerUpdate struct {
	Name    *string
	Email   *string
	Phone   *string
	Address *string
	Notes   *string
}

type SupplierUpdate struct {
	Name    *string
	Email   *string
	Phone   *string
	Address *string
	Notes   *string
}

type ProductUpdate struct {
	SKU            *string
	Name           *string
	Category       *string
	Unit           *string
	UnitPriceMinor *int64
	CostPriceMinor *int64
	ReorderLevel   *int64
}

type CustomerPage struct {
	Items      []models.Customer
	Page       int
	PerPage    int
	Total      int64
	TotalPages int
}

type SupplierPage struct {
	Items      []models.Supplier
	Page       int
	PerPage    int
	Total      int64
	TotalPages int
}

type ProductPage struct {
	Items      []models.Product
	Page       int
	PerPage    int
	Total      int64
	TotalPages int
}

func NewCatalogService(db *gorm.DB) (*CatalogService, error) {
	if db == nil {
		return nil, fmt.Errorf("database is required")
	}
	return &CatalogService{db: db}, nil
}

func (s *CatalogService) CreateCustomer(name, email, phone, address, notes string) (models.Customer, error) {
	customer := models.Customer{
		Name: strings.TrimSpace(name), Email: strings.ToLower(strings.TrimSpace(email)),
		Phone: strings.TrimSpace(phone), Address: strings.TrimSpace(address), Notes: strings.TrimSpace(notes),
	}
	if err := validateContact(customer.Name, customer.Email, customer.Phone, customer.Address, customer.Notes); err != nil {
		return models.Customer{}, err
	}
	if err := s.db.Create(&customer).Error; err != nil {
		return models.Customer{}, fmt.Errorf("create customer: %w", err)
	}
	return customer, nil
}

func (s *CatalogService) ListCustomers(filter CatalogFilter) (CustomerPage, error) {
	if err := validateFilter(filter, map[string]bool{"name": true, "email": true, "created_at": true}); err != nil {
		return CustomerPage{}, err
	}
	query := s.db.Model(&models.Customer{})
	if term := strings.TrimSpace(filter.Search); term != "" {
		pattern := "%" + term + "%"
		query = query.Where("name LIKE ? OR email LIKE ? OR phone LIKE ?", pattern, pattern, pattern)
	}
	var result CustomerPage
	result.Page, result.PerPage = filter.Page, filter.PerPage
	if err := query.Count(&result.Total).Error; err != nil {
		return CustomerPage{}, fmt.Errorf("count customers: %w", err)
	}
	if err := query.Order(orderClause(filter)).Offset((filter.Page - 1) * filter.PerPage).Limit(filter.PerPage).Find(&result.Items).Error; err != nil {
		return CustomerPage{}, fmt.Errorf("list customers: %w", err)
	}
	result.TotalPages = pageCount(result.Total, result.PerPage)
	return result, nil
}

func (s *CatalogService) GetCustomer(id uint) (models.Customer, error) {
	var customer models.Customer
	if err := s.db.First(&customer, id).Error; err != nil {
		return customer, catalogReadError("find customer", err)
	}
	return customer, nil
}

func (s *CatalogService) UpdateCustomer(id uint, update CustomerUpdate) (models.Customer, error) {
	changes := make(map[string]any)
	if update.Name != nil {
		value := strings.TrimSpace(*update.Name)
		if value == "" || len(value) > 160 {
			return models.Customer{}, ErrInvalidInput
		}
		changes["name"] = value
	}
	if update.Email != nil {
		value := strings.ToLower(strings.TrimSpace(*update.Email))
		if len(value) > 254 || !validEmailOrEmpty(value) {
			return models.Customer{}, ErrInvalidInput
		}
		changes["email"] = value
	}
	if !addStringChange(changes, "phone", update.Phone, 40) ||
		!addStringChange(changes, "address", update.Address, 500) ||
		!addStringChange(changes, "notes", update.Notes, 2000) || len(changes) == 0 {
		return models.Customer{}, ErrInvalidInput
	}
	var customer models.Customer
	if err := s.db.Model(&customer).Where("id = ?", id).Updates(changes).Error; err != nil {
		return customer, fmt.Errorf("update customer: %w", err)
	}
	return s.GetCustomer(id)
}

func (s *CatalogService) DeleteCustomer(id uint) error {
	result := s.db.Delete(&models.Customer{}, id)
	if result.Error != nil {
		return fmt.Errorf("delete customer: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *CatalogService) CreateSupplier(name, email, phone, address, notes string) (models.Supplier, error) {
	supplier := models.Supplier{
		Name: strings.TrimSpace(name), Email: strings.ToLower(strings.TrimSpace(email)),
		Phone: strings.TrimSpace(phone), Address: strings.TrimSpace(address), Notes: strings.TrimSpace(notes),
	}
	if err := validateContact(supplier.Name, supplier.Email, supplier.Phone, supplier.Address, supplier.Notes); err != nil {
		return models.Supplier{}, err
	}
	if err := s.db.Create(&supplier).Error; err != nil {
		return models.Supplier{}, fmt.Errorf("create supplier: %w", err)
	}
	return supplier, nil
}

func (s *CatalogService) ListSuppliers(filter CatalogFilter) (SupplierPage, error) {
	if err := validateFilter(filter, map[string]bool{"name": true, "email": true, "created_at": true}); err != nil {
		return SupplierPage{}, err
	}
	query := s.db.Model(&models.Supplier{})
	if term := strings.TrimSpace(filter.Search); term != "" {
		pattern := "%" + term + "%"
		query = query.Where("name LIKE ? OR email LIKE ? OR phone LIKE ?", pattern, pattern, pattern)
	}
	var result SupplierPage
	result.Page, result.PerPage = filter.Page, filter.PerPage
	if err := query.Count(&result.Total).Error; err != nil {
		return SupplierPage{}, fmt.Errorf("count suppliers: %w", err)
	}
	if err := query.Order(orderClause(filter)).Offset((filter.Page - 1) * filter.PerPage).Limit(filter.PerPage).Find(&result.Items).Error; err != nil {
		return SupplierPage{}, fmt.Errorf("list suppliers: %w", err)
	}
	result.TotalPages = pageCount(result.Total, result.PerPage)
	return result, nil
}

func (s *CatalogService) GetSupplier(id uint) (models.Supplier, error) {
	var supplier models.Supplier
	if err := s.db.First(&supplier, id).Error; err != nil {
		return supplier, catalogReadError("find supplier", err)
	}
	return supplier, nil
}

func (s *CatalogService) UpdateSupplier(id uint, update SupplierUpdate) (models.Supplier, error) {
	changes := make(map[string]any)
	if update.Name != nil {
		value := strings.TrimSpace(*update.Name)
		if value == "" || len(value) > 160 {
			return models.Supplier{}, ErrInvalidInput
		}
		changes["name"] = value
	}
	if update.Email != nil {
		value := strings.ToLower(strings.TrimSpace(*update.Email))
		if len(value) > 254 || !validEmailOrEmpty(value) {
			return models.Supplier{}, ErrInvalidInput
		}
		changes["email"] = value
	}
	if !addStringChange(changes, "phone", update.Phone, 40) ||
		!addStringChange(changes, "address", update.Address, 500) ||
		!addStringChange(changes, "notes", update.Notes, 2000) || len(changes) == 0 {
		return models.Supplier{}, ErrInvalidInput
	}
	var supplier models.Supplier
	if err := s.db.Model(&supplier).Where("id = ?", id).Updates(changes).Error; err != nil {
		return supplier, fmt.Errorf("update supplier: %w", err)
	}
	return s.GetSupplier(id)
}

func (s *CatalogService) DeleteSupplier(id uint) error {
	result := s.db.Delete(&models.Supplier{}, id)
	if result.Error != nil {
		return fmt.Errorf("delete supplier: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *CatalogService) CreateProduct(sku, name, category, unit string, unitPriceMinor, costPriceMinor, reorderLevel int64) (models.Product, error) {
	product := models.Product{
		SKU: strings.TrimSpace(sku), Name: strings.TrimSpace(name), Category: strings.TrimSpace(category),
		Unit: strings.TrimSpace(unit), UnitPriceMinor: unitPriceMinor, CostPriceMinor: costPriceMinor,
		ReorderLevel: reorderLevel,
	}
	if err := validateProduct(product.SKU, product.Name, product.Category, product.Unit, product.UnitPriceMinor, product.CostPriceMinor, product.ReorderLevel); err != nil {
		return models.Product{}, err
	}
	if err := s.db.Create(&product).Error; err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return models.Product{}, ErrConflict
		}
		return models.Product{}, fmt.Errorf("create product: %w", err)
	}
	return product, nil
}

func (s *CatalogService) ListProducts(filter CatalogFilter) (ProductPage, error) {
	if err := validateFilter(filter, map[string]bool{
		"sku": true, "name": true, "category": true, "unit_price_minor": true,
		"cost_price_minor": true, "stock_on_hand": true, "created_at": true,
	}); err != nil {
		return ProductPage{}, err
	}
	query := s.db.Model(&models.Product{})
	if term := strings.TrimSpace(filter.Search); term != "" {
		pattern := "%" + term + "%"
		query = query.Where("sku LIKE ? OR name LIKE ? OR category LIKE ?", pattern, pattern, pattern)
	}
	if category := strings.TrimSpace(filter.Category); category != "" {
		query = query.Where("category = ?", category)
	}
	var result ProductPage
	result.Page, result.PerPage = filter.Page, filter.PerPage
	if err := query.Count(&result.Total).Error; err != nil {
		return ProductPage{}, fmt.Errorf("count products: %w", err)
	}
	if err := query.Order(orderClause(filter)).Offset((filter.Page - 1) * filter.PerPage).Limit(filter.PerPage).Find(&result.Items).Error; err != nil {
		return ProductPage{}, fmt.Errorf("list products: %w", err)
	}
	result.TotalPages = pageCount(result.Total, result.PerPage)
	return result, nil
}

func (s *CatalogService) GetProduct(id uint) (models.Product, error) {
	var product models.Product
	if err := s.db.First(&product, id).Error; err != nil {
		return product, catalogReadError("find product", err)
	}
	return product, nil
}

func (s *CatalogService) UpdateProduct(id uint, update ProductUpdate) (models.Product, error) {
	changes := make(map[string]any)
	if update.SKU != nil {
		value := strings.TrimSpace(*update.SKU)
		if value == "" || len(value) > 80 {
			return models.Product{}, ErrInvalidInput
		}
		changes["sku"] = value
	}
	if update.Name != nil {
		value := strings.TrimSpace(*update.Name)
		if value == "" || len(value) > 200 {
			return models.Product{}, ErrInvalidInput
		}
		changes["name"] = value
	}
	if update.Category != nil {
		value := strings.TrimSpace(*update.Category)
		if len(value) > 120 {
			return models.Product{}, ErrInvalidInput
		}
		changes["category"] = value
	}
	if update.Unit != nil {
		value := strings.TrimSpace(*update.Unit)
		if value == "" || len(value) > 40 {
			return models.Product{}, ErrInvalidInput
		}
		changes["unit"] = value
	}
	if update.UnitPriceMinor != nil {
		if *update.UnitPriceMinor < 0 {
			return models.Product{}, ErrInvalidInput
		}
		changes["unit_price_minor"] = *update.UnitPriceMinor
	}
	if update.CostPriceMinor != nil {
		if *update.CostPriceMinor < 0 {
			return models.Product{}, ErrInvalidInput
		}
		changes["cost_price_minor"] = *update.CostPriceMinor
	}
	if update.ReorderLevel != nil {
		if *update.ReorderLevel < 0 {
			return models.Product{}, ErrInvalidInput
		}
		changes["reorder_level"] = *update.ReorderLevel
	}
	if len(changes) == 0 {
		return models.Product{}, ErrInvalidInput
	}
	var product models.Product
	if err := s.db.Model(&product).Where("id = ?", id).Updates(changes).Error; err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return models.Product{}, ErrConflict
		}
		return product, fmt.Errorf("update product: %w", err)
	}
	return s.GetProduct(id)
}

func (s *CatalogService) DeleteProduct(id uint) error {
	result := s.db.Delete(&models.Product{}, id)
	if result.Error != nil {
		return fmt.Errorf("delete product: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

func validateContact(name, email, phone, address, notes string) error {
	if name == "" || len(name) > 160 || len(email) > 254 || !validEmailOrEmpty(email) ||
		len(phone) > 40 || len(address) > 500 || len(notes) > 2000 {
		return ErrInvalidInput
	}
	return nil
}

func validEmailOrEmpty(email string) bool {
	if email == "" {
		return true
	}
	address, err := mail.ParseAddress(email)
	return err == nil && address.Address == email
}

func validateProduct(sku, name, category, unit string, unitPrice, costPrice, reorderLevel int64) error {
	if sku == "" || len(sku) > 80 || name == "" || len(name) > 200 ||
		len(category) > 120 || unit == "" || len(unit) > 40 ||
		unitPrice < 0 || costPrice < 0 || reorderLevel < 0 {
		return ErrInvalidInput
	}
	return nil
}

func validateFilter(filter CatalogFilter, allowedSort map[string]bool) error {
	if filter.Page < 1 || filter.PerPage < 1 || filter.PerPage > 100 {
		return ErrInvalidInput
	}
	maxInt := int(^uint(0) >> 1)
	if filter.Page-1 > maxInt/filter.PerPage {
		return ErrInvalidInput
	}
	if filter.Sort != "" && !allowedSort[filter.Sort] {
		return ErrInvalidInput
	}
	if filter.Order != "" && filter.Order != "asc" && filter.Order != "desc" {
		return ErrInvalidInput
	}
	if len(filter.Search) > 200 || len(filter.Category) > 120 {
		return ErrInvalidInput
	}
	return nil
}

func orderClause(filter CatalogFilter) string {
	column := filter.Sort
	if column == "" {
		column = "id"
	}
	direction := strings.ToUpper(filter.Order)
	if direction == "" {
		direction = "ASC"
	}
	return column + " " + direction + ", id ASC"
}

func pageCount(total int64, perPage int) int {
	return int((total + int64(perPage) - 1) / int64(perPage))
}

func catalogReadError(operation string, err error) error {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrNotFound
	}
	return fmt.Errorf("%s: %w", operation, err)
}

func addStringChange(changes map[string]any, column string, value *string, maxLength int) bool {
	if value == nil {
		return true
	}
	normalized := strings.TrimSpace(*value)
	if len(normalized) > maxLength {
		return false
	}
	changes[column] = normalized
	return true
}
