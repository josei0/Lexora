package config

import (
	"fmt"
	"os"
	"strings"
	"time"
)

type Config struct {
	Port        string
	DatabaseURL string
	QdrantURL   string

	JWTSecret     string
	JWTAccessTTL  time.Duration
	JWTRefreshTTL time.Duration

	ChatModel   string
	RAGTopK     int
	RAGMinScore float32

	EmbeddingProvider string
	EmbeddingURL      string
	EmbeddingModel    string
	EmbeddingDim      int
	MaiaAPIKey        string

	StorageDir string

	CORSOrigins  []string
	CookieSecure bool

	SuperAdminEmail    string
	SuperAdminPassword string
}

func Load() (*Config, error) {
	c := &Config{
		Port:               env("PORT", "8080"),
		DatabaseURL:        env("DATABASE_URL", ""),
		QdrantURL:          env("QDRANT_URL", "http://localhost:6333"),
		JWTSecret:          env("JWT_SECRET", ""),
		ChatModel:          env("CHAT_MODEL", "maia/claude-sonnet-4-5"),
		EmbeddingProvider:  env("EMBEDDING_PROVIDER", "maia"),
		EmbeddingURL:       env("EMBEDDING_URL", "https://api.maiarouter.ai/v1"),
		EmbeddingModel:     env("EMBEDDING_MODEL", "openai/text-embedding-3-large"),
		MaiaAPIKey:         env("MAIA_API_KEY", ""),
		StorageDir:         env("STORAGE_DIR", "./storage"),
		SuperAdminEmail:    env("SUPERADMIN_EMAIL", "admin@lexora.id"),
		SuperAdminPassword: env("SUPERADMIN_PASSWORD", ""),
	}

	c.JWTAccessTTL = envDuration("JWT_ACCESS_TTL", 15*time.Minute)
	c.JWTRefreshTTL = envDuration("JWT_REFRESH_TTL", 720*time.Hour)
	c.EmbeddingDim = envInt("EMBEDDING_DIM", 3072)
	c.RAGTopK = envInt("RAG_TOP_K", 5)
	c.RAGMinScore = float32(envInt("RAG_MIN_SCORE_PCT", 35)) / 100
	c.CORSOrigins = splitCSV(env("CORS_ORIGINS", "http://localhost:3000"))
	c.CookieSecure = env("COOKIE_SECURE", "false") == "true"

	if c.DatabaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL required")
	}
	if c.JWTSecret == "" {
		return nil, fmt.Errorf("JWT_SECRET required")
	}
	return c, nil
}

func splitCSV(s string) []string {
	var out []string
	for _, p := range strings.Split(s, ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
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
