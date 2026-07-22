package config

import (
	"fmt"
	"os"
	"time"
)

// app config
type Config struct {
	Port        string
	DatabaseURL string
	QdrantURL   string

	JWTSecret     string
	JWTAccessTTL  time.Duration
	JWTRefreshTTL time.Duration

	AnthropicAPIKey string
	ChatModel       string

	EmbeddingProvider string
	EmbeddingURL      string
	EmbeddingModel    string
	EmbeddingDim      int

	StorageDir string

	SuperAdminEmail    string
	SuperAdminPassword string
}

// read env
func Load() (*Config, error) {
	c := &Config{
		Port:               env("PORT", "8080"),
		DatabaseURL:        env("DATABASE_URL", ""),
		QdrantURL:          env("QDRANT_URL", "http://localhost:6333"),
		JWTSecret:          env("JWT_SECRET", ""),
		AnthropicAPIKey:    env("ANTHROPIC_API_KEY", ""),
		ChatModel:          env("CHAT_MODEL", "claude-sonnet-4-5"),
		EmbeddingProvider:  env("EMBEDDING_PROVIDER", "local"),
		EmbeddingURL:       env("EMBEDDING_URL", "http://localhost:11434"),
		EmbeddingModel:     env("EMBEDDING_MODEL", "nomic-embed-text"),
		StorageDir:         env("STORAGE_DIR", "./storage"),
		SuperAdminEmail:    env("SUPERADMIN_EMAIL", "admin@lexora.id"),
		SuperAdminPassword: env("SUPERADMIN_PASSWORD", ""),
	}

	c.JWTAccessTTL = envDuration("JWT_ACCESS_TTL", 15*time.Minute)
	c.JWTRefreshTTL = envDuration("JWT_REFRESH_TTL", 720*time.Hour)
	c.EmbeddingDim = envInt("EMBEDDING_DIM", 768)

	if c.DatabaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL required")
	}
	return c, nil
}

func env(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func envDuration(key string, def time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return def
}

func envInt(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		var n int
		if _, err := fmt.Sscanf(v, "%d", &n); err == nil {
			return n
		}
	}
	return def
}
