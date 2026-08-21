package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Environment      string
	HTTPAddress      string
	DatabaseURL      string
	StoreMode        string
	AttachmentRoot   string
	RequestTimeout   time.Duration
	ShutdownTimeout  time.Duration
	MaxRequestBytes  int64
	AllowedOrigins   []string
	QueryTokenPepper string
	SeedDemoData     bool
}

func Load() (Config, error) {
	cfg := Config{
		Environment:      value("APP_ENV", "development"),
		HTTPAddress:      value("HTTP_ADDRESS", ":8080"),
		DatabaseURL:      strings.TrimSpace(os.Getenv("DATABASE_URL")),
		StoreMode:        value("STORE_MODE", "memory"),
		AttachmentRoot:   value("ATTACHMENT_ROOT", "./var/attachments"),
		RequestTimeout:   duration("REQUEST_TIMEOUT", 5*time.Second),
		ShutdownTimeout:  duration("SHUTDOWN_TIMEOUT", 10*time.Second),
		MaxRequestBytes:  integer("MAX_REQUEST_BYTES", 12<<20),
		AllowedOrigins:   split(value("ALLOWED_ORIGINS", "http://localhost:5173")),
		QueryTokenPepper: value("QUERY_TOKEN_PEPPER", "local-development-only"),
		SeedDemoData:     boolean("SEED_DEMO_DATA", true),
	}
	if cfg.StoreMode != "memory" && cfg.StoreMode != "postgres" {
		return Config{}, fmt.Errorf("STORE_MODE must be memory or postgres")
	}
	if cfg.StoreMode == "postgres" && cfg.DatabaseURL == "" {
		return Config{}, fmt.Errorf("DATABASE_URL is required for postgres store")
	}
	if cfg.RequestTimeout <= 0 || cfg.ShutdownTimeout <= 0 {
		return Config{}, fmt.Errorf("timeouts must be positive")
	}
	if cfg.MaxRequestBytes < 1024 {
		return Config{}, fmt.Errorf("MAX_REQUEST_BYTES is too small")
	}
	return cfg, nil
}

func value(key, fallback string) string {
	if current := strings.TrimSpace(os.Getenv(key)); current != "" {
		return current
	}
	return fallback
}

func duration(key string, fallback time.Duration) time.Duration {
	current := strings.TrimSpace(os.Getenv(key))
	if current == "" {
		return fallback
	}
	parsed, err := time.ParseDuration(current)
	if err != nil {
		return fallback
	}
	return parsed
}

func integer(key string, fallback int64) int64 {
	current := strings.TrimSpace(os.Getenv(key))
	if current == "" {
		return fallback
	}
	parsed, err := strconv.ParseInt(current, 10, 64)
	if err != nil {
		return fallback
	}
	return parsed
}

func boolean(key string, fallback bool) bool {
	current := strings.TrimSpace(os.Getenv(key))
	if current == "" {
		return fallback
	}
	parsed, err := strconv.ParseBool(current)
	if err != nil {
		return fallback
	}
	return parsed
}

func split(value string) []string {
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}
