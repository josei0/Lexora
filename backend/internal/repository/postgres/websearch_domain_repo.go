package postgres

import (
	"context"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lexora/backend/internal/domain"
)

// allowlist domain web-search (update9-B). Pola PlanRepo.
type WebDomainRepo struct{ db *pgxpool.Pool }

func NewWebDomainRepo(db *pgxpool.Pool) *WebDomainRepo { return &WebDomainRepo{db} }

func (r *WebDomainRepo) List(ctx context.Context) ([]domain.WebSearchDomain, error) {
	rows, err := r.db.Query(ctx, `select id, host, created_at from web_search_domains order by host`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.WebSearchDomain
	for rows.Next() {
		var d domain.WebSearchDomain
		if err := rows.Scan(&d.ID, &d.Host, &d.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

func (r *WebDomainRepo) Add(ctx context.Context, host string) (*domain.WebSearchDomain, error) {
	host = strings.ToLower(strings.TrimSpace(host))
	var d domain.WebSearchDomain
	err := r.db.QueryRow(ctx,
		`insert into web_search_domains (host) values ($1) returning id, host, created_at`, host).
		Scan(&d.ID, &d.Host, &d.CreatedAt)
	if err != nil {
		return nil, mapErr(err) // unique -> ErrConflict
	}
	return &d, nil
}

func (r *WebDomainRepo) Remove(ctx context.Context, host string) error {
	host = strings.ToLower(strings.TrimSpace(host))
	ct, err := r.db.Exec(ctx, `delete from web_search_domains where host = $1`, host)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}
