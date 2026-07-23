package postgres

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lexora/backend/internal/domain"
)

type AuditRepo struct{ db *pgxpool.Pool }

func NewAuditRepo(db *pgxpool.Pool) *AuditRepo { return &AuditRepo{db} }

func (r *AuditRepo) Insert(ctx context.Context, e domain.AuditLog) error {
	_, err := r.db.Exec(ctx,
		`INSERT INTO audit_logs (org_id, user_id, action, resource_id, ip) VALUES ($1,$2,$3,$4,$5)`,
		e.OrgID, e.UserID, e.Action, e.ResourceID, e.IP,
	)
	return err
}

func (r *AuditRepo) Recent(ctx context.Context, limit int) ([]domain.AuditLog, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, org_id, user_id, action, resource_id, ip, created_at
		 FROM audit_logs ORDER BY created_at DESC LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.AuditLog
	for rows.Next() {
		var a domain.AuditLog
		if err := rows.Scan(&a.ID, &a.OrgID, &a.UserID, &a.Action, &a.ResourceID, &a.IP, &a.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}
