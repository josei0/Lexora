package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lexora/backend/internal/domain"
)

type TopupRepo struct{ db *pgxpool.Pool }

func NewTopupRepo(db *pgxpool.Pool) *TopupRepo { return &TopupRepo{db} }

func (r *TopupRepo) Create(ctx context.Context, t *domain.QuotaTopup) error {
	err := r.db.QueryRow(ctx, `
		insert into quota_topups (organization_id, invoice_id, tokens)
		values ($1, $2, $3)
		on conflict (invoice_id) do nothing
		returning id, created_at`,
		t.OrganizationID, t.InvoiceID, t.Tokens,
	).Scan(&t.ID, &t.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil // sudah ada: idempoten (mark-paid dobel)
	}
	return mapErr(err)
}

func (r *TopupRepo) SumTokens(ctx context.Context, orgID uuid.UUID, from, to time.Time) (int64, error) {
	var total int64
	err := r.db.QueryRow(ctx, `
		select coalesce(sum(tokens), 0) from quota_topups
		where organization_id = $1 and created_at >= $2 and created_at < $3`,
		orgID, from, to).Scan(&total)
	return total, err
}
