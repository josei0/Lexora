package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// allowlist domain web-search (update9-B). Sumber DB, env fallback seed.
type WebSearchDomain struct {
	ID        uuid.UUID
	Host      string
	CreatedAt time.Time
}

type WebDomainRepository interface {
	List(ctx context.Context) ([]WebSearchDomain, error)
	Add(ctx context.Context, host string) (*WebSearchDomain, error) // ErrConflict kalau dobel
	Remove(ctx context.Context, host string) error                  // ErrNotFound kalau tak ada
}
