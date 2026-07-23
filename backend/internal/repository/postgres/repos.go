package postgres

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lexora/backend/internal/domain"
)

type UserRepo struct{ db *pgxpool.Pool }

func NewUserRepo(db *pgxpool.Pool) *UserRepo { return &UserRepo{db} }

func (r *UserRepo) ByEmail(ctx context.Context, email string) (*domain.User, error) {
	return scanUser(r.db.QueryRow(ctx, `
		select id, email, password_hash, full_name, system_role, is_active, must_change_password, created_at
		from users where email = $1`, email))
}

func (r *UserRepo) ByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	return scanUser(r.db.QueryRow(ctx, `
		select id, email, password_hash, full_name, system_role, is_active, must_change_password, created_at
		from users where id = $1`, id))
}

func (r *UserRepo) Create(ctx context.Context, u *domain.User) error {
	err := r.db.QueryRow(ctx, `
		insert into users (email, password_hash, full_name, system_role, is_active, must_change_password)
		values ($1, $2, $3, $4, $5, $6) returning id, created_at`,
		u.Email, u.PasswordHash, u.FullName, u.SystemRole, u.IsActive, u.MustChangePassword,
	).Scan(&u.ID, &u.CreatedAt)
	return mapErr(err)
}

func (r *UserRepo) UpdatePassword(ctx context.Context, id uuid.UUID, hash string) error {
	_, err := r.db.Exec(ctx, `
		update users set password_hash = $2, must_change_password = false, updated_at = now()
		where id = $1`, id, hash)
	return err
}

func (r *UserRepo) SetActive(ctx context.Context, id uuid.UUID, active bool) error {
	_, err := r.db.Exec(ctx, `update users set is_active = $2, updated_at = now() where id = $1`, id, active)
	return err
}

func scanUser(row pgx.Row) (*domain.User, error) {
	var u domain.User
	err := row.Scan(&u.ID, &u.Email, &u.PasswordHash, &u.FullName, &u.SystemRole, &u.IsActive, &u.MustChangePassword, &u.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

type OrgRepo struct{ db *pgxpool.Pool }

func NewOrgRepo(db *pgxpool.Pool) *OrgRepo { return &OrgRepo{db} }

func (r *OrgRepo) Create(ctx context.Context, o *domain.Organization) error {
	err := r.db.QueryRow(ctx, `
		insert into organizations (name, slug) values ($1, $2) returning id, created_at`,
		o.Name, o.Slug).Scan(&o.ID, &o.CreatedAt)
	return mapErr(err)
}

func (r *OrgRepo) List(ctx context.Context) ([]domain.Organization, error) {
	rows, err := r.db.Query(ctx, `select id, name, slug, created_at from organizations order by created_at desc`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Organization
	for rows.Next() {
		var o domain.Organization
		if err := rows.Scan(&o.ID, &o.Name, &o.Slug, &o.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, o)
	}
	return out, rows.Err()
}

func (r *OrgRepo) BySlug(ctx context.Context, slug string) (*domain.Organization, error) {
	var o domain.Organization
	err := r.db.QueryRow(ctx, `select id, name, slug, created_at from organizations where slug = $1`, slug).
		Scan(&o.ID, &o.Name, &o.Slug, &o.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &o, nil
}

type MembershipRepo struct{ db *pgxpool.Pool }

func NewMembershipRepo(db *pgxpool.Pool) *MembershipRepo { return &MembershipRepo{db} }

func (r *MembershipRepo) Create(ctx context.Context, m *domain.Membership) error {
	_, err := r.db.Exec(ctx, `
		insert into memberships (user_id, organization_id, role) values ($1, $2, $3)`,
		m.UserID, m.OrganizationID, m.Role)
	return mapErr(err)
}

// scoped by org - anti-IDOR
func (r *MembershipRepo) ByUserOrg(ctx context.Context, userID, orgID uuid.UUID) (*domain.Membership, error) {
	var m domain.Membership
	err := r.db.QueryRow(ctx, `
		select user_id, organization_id, role from memberships where user_id = $1 and organization_id = $2`,
		userID, orgID).Scan(&m.UserID, &m.OrganizationID, &m.Role)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &m, nil
}

func (r *MembershipRepo) ListByOrg(ctx context.Context, orgID uuid.UUID) ([]domain.Member, error) {
	rows, err := r.db.Query(ctx, `
		select u.id, u.email, u.full_name, m.role, u.is_active
		from memberships m join users u on u.id = m.user_id
		where m.organization_id = $1 order by u.full_name`, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Member
	for rows.Next() {
		var m domain.Member
		if err := rows.Scan(&m.UserID, &m.Email, &m.FullName, &m.Role, &m.IsActive); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

// scoped by org - anti-IDOR
func (r *MembershipRepo) SetRole(ctx context.Context, userID, orgID uuid.UUID, role string) error {
	tag, err := r.db.Exec(ctx, `
		update memberships set role = $3, updated_at = now()
		where user_id = $1 and organization_id = $2`, userID, orgID, role)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *MembershipRepo) Primary(ctx context.Context, userID uuid.UUID) (*domain.Membership, error) {
	var m domain.Membership
	err := r.db.QueryRow(ctx, `
		select user_id, organization_id, role from memberships
		where user_id = $1 order by created_at limit 1`, userID).
		Scan(&m.UserID, &m.OrganizationID, &m.Role)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &m, nil
}

type RefreshRepo struct{ db *pgxpool.Pool }

func NewRefreshRepo(db *pgxpool.Pool) *RefreshRepo { return &RefreshRepo{db} }

func (r *RefreshRepo) Create(ctx context.Context, t *domain.RefreshToken) error {
	return r.db.QueryRow(ctx, `
		insert into refresh_tokens (user_id, token_hash, expires_at) values ($1, $2, $3) returning id`,
		t.UserID, t.TokenHash, t.ExpiresAt).Scan(&t.ID)
}

func (r *RefreshRepo) ByHash(ctx context.Context, hash string) (*domain.RefreshToken, error) {
	var t domain.RefreshToken
	err := r.db.QueryRow(ctx, `
		select id, user_id, token_hash, expires_at, revoked_at from refresh_tokens where token_hash = $1`,
		hash).Scan(&t.ID, &t.UserID, &t.TokenHash, &t.ExpiresAt, &t.RevokedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func (r *RefreshRepo) Revoke(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, `update refresh_tokens set revoked_at = now() where id = $1`, id)
	return err
}

func (r *RefreshRepo) RevokeByHash(ctx context.Context, hash string) error {
	_, err := r.db.Exec(ctx, `update refresh_tokens set revoked_at = now() where token_hash = $1`, hash)
	return err
}

// unique violation -> conflict
func mapErr(err error) error {
	if err == nil {
		return nil
	}
	var pgErr interface{ SQLState() string }
	if errors.As(err, &pgErr) && pgErr.SQLState() == "23505" {
		return domain.ErrConflict
	}
	return err
}
