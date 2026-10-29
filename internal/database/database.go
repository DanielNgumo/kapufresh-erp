package database

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/glebarez/sqlite"
	"github.com/yourusername/kapufresh-erp/internal/models"
	"gorm.io/gorm"
)

func Open(path string) (*gorm.DB, error) {
	if path == "" {
		return nil, fmt.Errorf("database path is required")
	}
	if path != ":memory:" {
		if err := os.MkdirAll(filepath.Dir(path), 0750); err != nil {
			return nil, fmt.Errorf("create database directory: %w", err)
		}
	}

	db, err := gorm.Open(sqlite.Open(path), &gorm.Config{TranslateError: true})
	if err != nil {
		return nil, fmt.Errorf("open SQLite database: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("get SQLite connection pool: %w", err)
	}
	sqlDB.SetMaxOpenConns(1)
	sqlDB.SetMaxIdleConns(1)

	if err := db.Exec("PRAGMA foreign_keys = ON").Error; err != nil {
		return nil, fmt.Errorf("enable SQLite foreign keys: %w", err)
	}
	if err := db.Exec("PRAGMA busy_timeout = 5000").Error; err != nil {
		return nil, fmt.Errorf("set SQLite busy timeout: %w", err)
	}

	return db, nil
}

func Migrate(db *gorm.DB) error {
	if db == nil {
		return fmt.Errorf("database is required")
	}

	if err := db.AutoMigrate(
		&models.User{},
		&models.RefreshToken{},
		&models.Customer{},
		&models.Supplier{},
		&models.Product{},
		&models.StockMovement{},
		&models.SalesOrder{},
		&models.SalesOrderItem{},
		&models.Payment{},
		&models.PurchaseOrder{},
		&models.PurchaseOrderItem{},
		&models.Employee{},
	); err != nil {
		return fmt.Errorf("auto-migrate models: %w", err)
	}
	return nil
}
