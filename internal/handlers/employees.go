package handlers

import (
	"errors"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/yourusername/kapufresh-erp/internal/dto"
	"github.com/yourusername/kapufresh-erp/internal/models"
	"github.com/yourusername/kapufresh-erp/internal/services"
	"github.com/yourusername/kapufresh-erp/internal/utils"
)

type EmployeeHandler struct {
	employees *services.EmployeeService
}

func NewEmployeeHandler(employees *services.EmployeeService) *EmployeeHandler {
	return &EmployeeHandler{employees: employees}
}

func (h *EmployeeHandler) Create(c *gin.Context) {
	var request dto.EmployeeRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		invalidBody(c)
		return
	}
	employee, err := h.employees.Create(
		request.UserID, request.FirstName, request.LastName, request.Email, request.Phone,
		request.Department, request.Position, models.EmployeeStatus(request.Status),
	)
	if err != nil {
		h.respondError(c, err)
		return
	}
	utils.RespondData(c, http.StatusCreated, employeeResponse(employee))
}

func (h *EmployeeHandler) List(c *gin.Context) {
	page, perPage, ok := catalogPageValues(c)
	if !ok {
		return
	}
	result, err := h.employees.List(services.EmployeeFilter{
		Page: page, PerPage: perPage, Search: strings.TrimSpace(c.Query("search")),
		Department: strings.TrimSpace(c.Query("department")),
		Status:     models.EmployeeStatus(strings.TrimSpace(c.Query("status"))),
		Sort:       strings.TrimSpace(c.Query("sort")), Order: strings.ToLower(strings.TrimSpace(c.Query("order"))),
	})
	if err != nil {
		h.respondError(c, err)
		return
	}
	items := make([]dto.EmployeeResponse, 0, len(result.Items))
	for _, employee := range result.Items {
		items = append(items, employeeResponse(employee))
	}
	utils.RespondData(c, http.StatusOK, dto.EmployeeListResponse{
		Employees: items,
		Pagination: dto.Pagination{
			Page: result.Page, PerPage: result.PerPage, Total: result.Total, TotalPages: result.TotalPages,
		},
	})
}

func (h *EmployeeHandler) Get(c *gin.Context) {
	id, ok := catalogPathID(c)
	if !ok {
		return
	}
	employee, err := h.employees.Get(id)
	if err != nil {
		h.respondError(c, err)
		return
	}
	utils.RespondData(c, http.StatusOK, employeeResponse(employee))
}

func (h *EmployeeHandler) Update(c *gin.Context) {
	id, ok := catalogPathID(c)
	if !ok {
		return
	}
	var request dto.UpdateEmployeeRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		invalidBody(c)
		return
	}
	var status *models.EmployeeStatus
	if request.Status != nil {
		value := models.EmployeeStatus(*request.Status)
		status = &value
	}
	employee, err := h.employees.Update(id, services.EmployeeUpdate{
		FirstName: request.FirstName, LastName: request.LastName, Email: request.Email,
		Phone: request.Phone, Department: request.Department, Position: request.Position, Status: status,
	})
	if err != nil {
		h.respondError(c, err)
		return
	}
	utils.RespondData(c, http.StatusOK, employeeResponse(employee))
}

func (h *EmployeeHandler) Delete(c *gin.Context) {
	id, ok := catalogPathID(c)
	if !ok {
		return
	}
	if err := h.employees.Delete(id); err != nil {
		h.respondError(c, err)
		return
	}
	utils.RespondData(c, http.StatusOK, gin.H{"message": "Employee deleted"})
}

func (h *EmployeeHandler) respondError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, services.ErrNotFound):
		utils.RespondError(c, http.StatusNotFound, "not_found", "Employee was not found")
	case errors.Is(err, services.ErrConflict):
		utils.RespondError(c, http.StatusConflict, "conflict", "This user is already linked to an employee")
	case errors.Is(err, services.ErrInvalidInput):
		utils.RespondError(c, http.StatusBadRequest, "invalid_request", "Request values are invalid")
	default:
		log.Printf("employee handler error: %v", err)
		utils.RespondError(c, http.StatusInternalServerError, "internal_error", "An unexpected error occurred")
	}
}

func employeeResponse(employee models.Employee) dto.EmployeeResponse {
	return dto.EmployeeResponse{
		ID: employee.ID, UserID: employee.UserID, FirstName: employee.FirstName, LastName: employee.LastName,
		Email: employee.Email, Phone: employee.Phone, Department: employee.Department,
		Position: employee.Position, Status: string(employee.Status),
		CreatedAt: employee.CreatedAt.Format(time.RFC3339), UpdatedAt: employee.UpdatedAt.Format(time.RFC3339),
	}
}
