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

	JWTSecret      string
	JWTAdminSecret string // kunci admin, terpisah
	JWTAccessTTL   time.Duration
	JWTRefreshTTL  time.Duration

	ChatModelHigh   string // tier High (Pro)
	ChatModelNormal string // tier Normal (Demo + degrade Pro)
	RAGTopK         int
	RAGMinScore     float32

	EmbeddingProvider string
	EmbeddingURL      string
	EmbeddingModel    string
	EmbeddingDim      int
	MaiaAPIKey        string

	StorageDir string

	WebSearchModel   string
	WebSearchDomains []string // allowlist ingest + filter hasil search

	CORSOriginsApp   []string // origin app
	CORSOriginsAdmin []string // origin admin
	CookieSecure     bool

	SuperAdminEmail    string
	SuperAdminPassword string
}

func Load() (*Config, error) {
	c := &Config{
		Port:               env("PORT", "8080"),
		DatabaseURL:        env("DATABASE_URL", ""),
		QdrantURL:          env("QDRANT_URL", "http://localhost:6333"),
		JWTSecret:          env("JWT_SECRET", ""),
		JWTAdminSecret:     env("JWT_ADMIN_SECRET", ""),
		ChatModelHigh:      env("CHAT_MODEL_HIGH", env("CHAT_MODEL", "maia/claude-sonnet-4-5")), // CHAT_MODEL = nama lama
		ChatModelNormal:    env("CHAT_MODEL_NORMAL", "anthropic/claude-haiku-4-5"),
		EmbeddingProvider:  env("EMBEDDING_PROVIDER", "maia"),
		EmbeddingURL:       env("EMBEDDING_URL", "https://api.maiarouter.ai/v1"),
		EmbeddingModel:     env("EMBEDDING_MODEL", "openai/text-embedding-3-large"),
		MaiaAPIKey:         env("MAIA_API_KEY", ""),
		StorageDir:         env("STORAGE_DIR", "./storage"),
		SuperAdminEmail:    env("SUPERADMIN_EMAIL", "admin@mindlaw.web.id"),
		SuperAdminPassword: env("SUPERADMIN_PASSWORD", ""),
	}

	c.JWTAccessTTL = envDuration("JWT_ACCESS_TTL", 15*time.Minute)
	c.JWTRefreshTTL = envDuration("JWT_REFRESH_TTL", 720*time.Hour)
	c.EmbeddingDim = envInt("EMBEDDING_DIM", 3072)
	c.RAGTopK = envInt("RAG_TOP_K", 5)
	c.RAGMinScore = float32(envInt("RAG_MIN_SCORE_PCT", 35)) / 100
	c.WebSearchModel = env("WEB_SEARCH_MODEL", "openai/gpt-4o-mini-search-preview")
	c.WebSearchDomains = splitCSV(env("WEB_SEARCH_DOMAINS", "peraturan.bpk.go.id,peraturan.go.id,jdihn.go.id,putusan3.mahkamahagung.go.id"))
	c.CORSOriginsApp = splitCSV(env("CORS_ORIGINS_APP", "http://localhost:3000"))
	c.CORSOriginsAdmin = splitCSV(env("CORS_ORIGINS_ADMIN", "https://admin.lvh.me"))
	c.CookieSecure = env("COOKIE_SECURE", "false") == "true"

	if c.DatabaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL required")
	}
	if c.JWTSecret == "" {
		return nil, fmt.Errorf("JWT_SECRET required")
	}
	if c.JWTAdminSecret == "" {
		return nil, fmt.Errorf("JWT_ADMIN_SECRET required")
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
