package main

import (
	"log"

	"github.com/gin-gonic/gin"
	"github.com/yourusername/kapufresh-erp/internal/config"
	"github.com/yourusername/kapufresh-erp/internal/database"
	"github.com/yourusername/kapufresh-erp/internal/routes"
	"github.com/yourusername/kapufresh-erp/internal/services"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load configuration: %v", err)
	}

	db, err := database.Open(cfg.DatabasePath)
	if err != nil {
		log.Fatalf("open database: %v", err)
	}

	if err := database.Migrate(db); err != nil {
		log.Fatalf("migrate database: %v", err)
	}

	if err := database.SeedAdmin(db, cfg.AdminEmail, cfg.AdminPassword, cfg.AdminName); err != nil {
		log.Fatalf("seed admin user: %v", err)
	}

	if cfg.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	authService, err := services.NewAuthService(db, cfg.JWTSecret, cfg.AccessTokenTTL, cfg.RefreshTokenTTL)
	if err != nil {
		log.Fatalf("configure authentication: %v", err)
	}
	catalogService, err := services.NewCatalogService(db)
	if err != nil {
		log.Fatalf("configure catalog: %v", err)
	}
	r := routes.New(authService, catalogService)

	if err := r.Run(":" + cfg.Port); err != nil {
		log.Fatalf("run server: %v", err)
	}
}
