package services

import (
	"errors"
	"fmt"
	"net/mail"
	"strings"

	"github.com/yourusername/kapufresh-erp/internal/models"
	"gorm.io/gorm"
)

type EmployeeService struct {
	db *gorm.DB
}

type EmployeeFilter struct {
	Page       int
	PerPage    int
	Search     string
	Department string
	Status     models.EmployeeStatus
	Sort       string
	Order      string
}

type EmployeeUpdate struct {
	FirstName  *string
	LastName   *string
	Email      *string
	Phone      *string
	Department *string
	Position   *string
	Status     *models.EmployeeStatus
}

type EmployeePage struct {
	Items      []models.Employee
	Page       int
	PerPage    int
	Total      int64
	TotalPages int
}

func NewEmployeeService(db *gorm.DB) (*EmployeeService, error) {
	if db == nil {
		return nil, fmt.Errorf("database is required")
	}
	return &EmployeeService{db: db}, nil
}

func (s *EmployeeService) Create(userID *uint, firstName, lastName, email, phone, department, position string, status models.EmployeeStatus) (models.Employee, error) {
	employee := models.Employee{
		UserID: userID, FirstName: strings.TrimSpace(firstName), LastName: strings.TrimSpace(lastName),
		Email: strings.ToLower(strings.TrimSpace(email)), Phone: strings.TrimSpace(phone),
		Department: strings.TrimSpace(department), Position: strings.TrimSpace(position), Status: status,
	}
	if employee.Status == "" {
		employee.Status = models.EmployeeStatusActive
	}
	if err := validateEmployee(employee.FirstName, employee.LastName, employee.Email, employee.Phone, employee.Department, employee.Position, employee.Status); err != nil {
		return models.Employee{}, err
	}
	if err := s.db.Create(&employee).Error; err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return models.Employee{}, ErrConflict
		}
		return models.Employee{}, fmt.Errorf("create employee: %w", err)
	}
	return employee, nil
}

func (s *EmployeeService) List(filter EmployeeFilter) (EmployeePage, error) {
	if err := validateFilter(CatalogFilter{
		Page: filter.Page, PerPage: filter.PerPage, Search: filter.Search,
		Sort: filter.Sort, Order: filter.Order,
	}, map[string]bool{
		"first_name": true, "last_name": true, "email": true, "department": true,
		"position": true, "status": true, "created_at": true,
	}); err != nil {
		return EmployeePage{}, err
	}
	if filter.Status != "" && filter.Status != models.EmployeeStatusActive && filter.Status != models.EmployeeStatusInactive {
		return EmployeePage{}, ErrInvalidInput
	}
	if len(filter.Department) > 120 {
		return EmployeePage{}, ErrInvalidInput
	}
	query := s.db.Model(&models.Employee{})
	if term := strings.TrimSpace(filter.Search); term != "" {
		pattern := "%" + term + "%"
		query = query.Where("first_name LIKE ? OR last_name LIKE ? OR email LIKE ?", pattern, pattern, pattern)
	}
	if department := strings.TrimSpace(filter.Department); department != "" {
		query = query.Where("department = ?", department)
	}
	if filter.Status != "" {
		query = query.Where("status = ?", filter.Status)
	}
	var result EmployeePage
	result.Page, result.PerPage = filter.Page, filter.PerPage
	if err := query.Count(&result.Total).Error; err != nil {
		return EmployeePage{}, fmt.Errorf("count employees: %w", err)
	}
	if err := query.Order(orderClause(CatalogFilter{Sort: filter.Sort, Order: filter.Order})).
		Offset((filter.Page - 1) * filter.PerPage).Limit(filter.PerPage).Find(&result.Items).Error; err != nil {
		return EmployeePage{}, fmt.Errorf("list employees: %w", err)
	}
	result.TotalPages = pageCount(result.Total, result.PerPage)
	return result, nil
}

func (s *EmployeeService) Get(id uint) (models.Employee, error) {
	var employee models.Employee
	if err := s.db.First(&employee, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return employee, ErrNotFound
		}
		return employee, fmt.Errorf("find employee: %w", err)
	}
	return employee, nil
}

func (s *EmployeeService) Update(id uint, update EmployeeUpdate) (models.Employee, error) {
	changes := make(map[string]any)
	addRequiredEmployeeChange(changes, "first_name", update.FirstName, 120)
	addRequiredEmployeeChange(changes, "last_name", update.LastName, 120)
	addOptionalEmployeeChange(changes, "phone", update.Phone, 40)
	addOptionalEmployeeChange(changes, "department", update.Department, 120)
	addOptionalEmployeeChange(changes, "position", update.Position, 120)
	if update.Email != nil {
		email := strings.ToLower(strings.TrimSpace(*update.Email))
		if len(email) > 254 || !validEmployeeEmail(email) {
			return models.Employee{}, ErrInvalidInput
		}
		changes["email"] = email
	}
	if update.Status != nil {
		if *update.Status != models.EmployeeStatusActive && *update.Status != models.EmployeeStatusInactive {
			return models.Employee{}, ErrInvalidInput
		}
		changes["status"] = *update.Status
	}
	if len(changes) == 0 || hasEmployeeStringError(changes) {
		return models.Employee{}, ErrInvalidInput
	}
	result := s.db.Model(&models.Employee{}).Where("id = ?", id).Updates(changes)
	if result.Error != nil {
		return models.Employee{}, fmt.Errorf("update employee: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return models.Employee{}, ErrNotFound
	}
	return s.Get(id)
}

func (s *EmployeeService) Delete(id uint) error {
	result := s.db.Delete(&models.Employee{}, id)
	if result.Error != nil {
		return fmt.Errorf("delete employee: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

func validateEmployee(firstName, lastName, email, phone, department, position string, status models.EmployeeStatus) error {
	if firstName == "" || len(firstName) > 120 || lastName == "" || len(lastName) > 120 ||
		len(email) > 254 || !validEmployeeEmail(email) || len(phone) > 40 ||
		len(department) > 120 || len(position) > 120 ||
		(status != models.EmployeeStatusActive && status != models.EmployeeStatusInactive) {
		return ErrInvalidInput
	}
	return nil
}

func validEmployeeEmail(email string) bool {
	if email == "" {
		return true
	}
	address, err := mail.ParseAddress(email)
	return err == nil && address.Address == email
}

func addRequiredEmployeeChange(changes map[string]any, key string, value *string, max int) {
	if value == nil {
		return
	}
	normalized := strings.TrimSpace(*value)
	if normalized == "" || len(normalized) > max {
		changes["invalid_"+key] = normalized
		return
	}
	changes[key] = normalized
}

func addOptionalEmployeeChange(changes map[string]any, key string, value *string, max int) {
	if value == nil {
		return
	}
	normalized := strings.TrimSpace(*value)
	if len(normalized) > max {
		changes["invalid_"+key] = normalized
		return
	}
	changes[key] = normalized
}

func hasEmployeeStringError(changes map[string]any) bool {
	for key := range changes {
		if strings.HasPrefix(key, "invalid_") {
			return true
		}
	}
	return false
}
