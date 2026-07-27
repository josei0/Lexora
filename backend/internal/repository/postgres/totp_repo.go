package postgres

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lexora/backend/internal/domain"
)

type RecoveryRepo struct{ db *pgxpool.Pool }

func NewRecoveryRepo(db *pgxpool.Pool) *RecoveryRepo { return &RecoveryRepo{db} }

// set baru menggantikan semua kode lama (re-enroll = kode lama hangus)
func (r *RecoveryRepo) Replace(ctx context.Context, userID uuid.UUID, hashes []string) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `delete from admin_recovery_codes where user_id = $1`, userID); err != nil {
		return err
	}
	for _, h := range hashes {
		if _, err := tx.Exec(ctx, `insert into admin_recovery_codes (user_id, code_hash) values ($1, $2)`, userID, h); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func (r *RecoveryRepo) Unused(ctx context.Context, userID uuid.UUID) ([]domain.RecoveryCode, error) {
	rows, err := r.db.Query(ctx, `
		select id, user_id, code_hash from admin_recovery_codes
		where user_id = $1 and used_at is null`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.RecoveryCode
	for rows.Next() {
		var c domain.RecoveryCode
		if err := rows.Scan(&c.ID, &c.UserID, &c.CodeHash); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (r *RecoveryRepo) MarkUsed(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, `update admin_recovery_codes set used_at = now() where id = $1 and used_at is null`, id)
	return err
}
