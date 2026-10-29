package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/yourusername/kapufresh-erp/internal/handlers"
	"github.com/yourusername/kapufresh-erp/internal/middleware"
	"github.com/yourusername/kapufresh-erp/internal/models"
	"github.com/yourusername/kapufresh-erp/internal/services"
)

func New(auth *services.AuthService, catalog *services.CatalogService) *gin.Engine {
	router := gin.Default()
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok", "service": "kapufresh-erp"})
	})

	handler := handlers.NewAuthHandler(auth)
	api := router.Group("/api/v1")
	api.POST("/auth/login", handler.Login)
	api.POST("/auth/refresh", handler.Refresh)

	protected := api.Group("")
	protected.Use(middleware.RequireAuth(auth))
	protected.POST("/auth/register", middleware.RequireRoles(models.RoleAdmin), handler.Register)
	protected.POST("/auth/logout", handler.Logout)
	protected.GET("/auth/me", handler.Me)

	users := protected.Group("/users")
	users.Use(middleware.RequireRoles(models.RoleAdmin))
	users.GET("", handler.ListUsers)
	users.GET("/:id", handler.GetUser)
	users.PATCH("/:id", handler.UpdateUser)
	users.DELETE("/:id", handler.DeleteUser)

	catalogHandler := handlers.NewCatalogHandler(catalog)
	protected.GET("/customers", catalogHandler.ListCustomers)
	protected.POST("/customers", middleware.RequireRoles(models.RoleAdmin, models.RoleManager), catalogHandler.CreateCustomer)
	protected.GET("/customers/:id", catalogHandler.GetCustomer)
	protected.PATCH("/customers/:id", middleware.RequireRoles(models.RoleAdmin, models.RoleManager), catalogHandler.UpdateCustomer)
	protected.DELETE("/customers/:id", middleware.RequireRoles(models.RoleAdmin, models.RoleManager), catalogHandler.DeleteCustomer)
	protected.GET("/suppliers", catalogHandler.ListSuppliers)
	protected.POST("/suppliers", middleware.RequireRoles(models.RoleAdmin, models.RoleManager), catalogHandler.CreateSupplier)
	protected.GET("/suppliers/:id", catalogHandler.GetSupplier)
	protected.PATCH("/suppliers/:id", middleware.RequireRoles(models.RoleAdmin, models.RoleManager), catalogHandler.UpdateSupplier)
	protected.DELETE("/suppliers/:id", middleware.RequireRoles(models.RoleAdmin, models.RoleManager), catalogHandler.DeleteSupplier)
	protected.GET("/products", catalogHandler.ListProducts)
	protected.POST("/products", middleware.RequireRoles(models.RoleAdmin, models.RoleManager), catalogHandler.CreateProduct)
	protected.GET("/products/:id", catalogHandler.GetProduct)
	protected.PATCH("/products/:id", middleware.RequireRoles(models.RoleAdmin, models.RoleManager), catalogHandler.UpdateProduct)
	protected.DELETE("/products/:id", middleware.RequireRoles(models.RoleAdmin, models.RoleManager), catalogHandler.DeleteProduct)

	return router
}
