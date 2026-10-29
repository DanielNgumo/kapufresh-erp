package config

import (
	"fmt"
	"net/mail"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	Environment     string
	Port            string
	DatabasePath    string
	AdminName       string
	AdminEmail      string
	AdminPassword   string
	CurrencyCode    string
	JWTSecret       string
	AccessTokenTTL  time.Duration
	RefreshTokenTTL time.Duration
}

func Load() (Config, error) {
	if err := godotenv.Load(); err != nil && !os.IsNotExist(err) {
		return Config{}, fmt.Errorf("load .env: %w", err)
	}

	cfg := Config{
		Environment:   envOrDefault("APP_ENV", "development"),
		Port:          envOrDefault("APP_PORT", "8080"),
		DatabasePath:  envOrDefault("DATABASE_PATH", "data/kapufresh.db"),
		AdminName:     envOrDefault("ADMIN_NAME", "System Administrator"),
		AdminEmail:    strings.ToLower(strings.TrimSpace(os.Getenv("ADMIN_EMAIL"))),
		AdminPassword: os.Getenv("ADMIN_PASSWORD"),
		CurrencyCode:  strings.ToUpper(envOrDefault("CURRENCY_CODE", "KES")),
		JWTSecret:     os.Getenv("JWT_SECRET"),
	}

	port, err := strconv.Atoi(cfg.Port)
	if err != nil || port < 1 || port > 65535 {
		return Config{}, fmt.Errorf("APP_PORT must be a valid TCP port")
	}
	if cfg.AdminEmail == "" {
		return Config{}, fmt.Errorf("ADMIN_EMAIL is required to seed the first admin user")
	}
	address, err := mail.ParseAddress(cfg.AdminEmail)
	if err != nil || address.Address != cfg.AdminEmail {
		return Config{}, fmt.Errorf("ADMIN_EMAIL must be a valid email address")
	}
	if len(cfg.AdminPassword) < 12 {
		return Config{}, fmt.Errorf("ADMIN_PASSWORD must be at least 12 characters")
	}
	if len(cfg.AdminPassword) > 72 {
		return Config{}, fmt.Errorf("ADMIN_PASSWORD must be no more than 72 bytes for bcrypt")
	}
	if len(cfg.CurrencyCode) != 3 {
		return Config{}, fmt.Errorf("CURRENCY_CODE must be a three-letter code")
	}
	for _, character := range cfg.CurrencyCode {
		if character < 'A' || character > 'Z' {
			return Config{}, fmt.Errorf("CURRENCY_CODE must contain only letters")
		}
	}
	if len(cfg.JWTSecret) < 32 {
		return Config{}, fmt.Errorf("JWT_SECRET must be at least 32 bytes")
	}
	cfg.AccessTokenTTL, err = durationOrDefault("ACCESS_TOKEN_TTL", 15*time.Minute)
	if err != nil {
		return Config{}, err
	}
	cfg.RefreshTokenTTL, err = durationOrDefault("REFRESH_TOKEN_TTL", 7*24*time.Hour)
	if err != nil {
		return Config{}, err
	}

	return cfg, nil
}

func envOrDefault(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func durationOrDefault(key string, fallback time.Duration) (time.Duration, error) {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback, nil
	}
	duration, err := time.ParseDuration(value)
	if err != nil || duration <= 0 {
		return 0, fmt.Errorf("%s must be a positive Go duration", key)
	}
	return duration, nil
}
