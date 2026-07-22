package config

import (
	"fmt"
	"os"
)

const (
	StorageMemory   = "memory"
	StoragePostgres = "postgres"
)

type Config struct {
	Port string

	Storage string

	PostgresDSN string
}

func Load() (Config, error) {
	cfg := Config{
		Port:        getEnv("SERVER_PORT", "8080"),
		Storage:     getEnv("STORAGE", StorageMemory),
		PostgresDSN: os.Getenv("POSTGRES_DSN"),
	}

	switch cfg.Storage {
	case StorageMemory:
	case StoragePostgres:
		if cfg.PostgresDSN == "" {
			return Config{}, fmt.Errorf("POSTGRES_DSN is required when STORAGE=%s", StoragePostgres)
		}
	default:
		return Config{}, fmt.Errorf("unknown STORAGE %q: expected %q or %q", cfg.Storage, StorageMemory, StoragePostgres)
	}
	return cfg, nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
