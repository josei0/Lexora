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

type PlanRepo struct{ db *pgxpool.Pool }

func NewPlanRepo(db *pgxpool.Pool) *PlanRepo { return &PlanRepo{db} }

// kolom plan dipakai bareng di semua SELECT (hindari drift). Urutan = scanPlan.
const planCols = `id, code, name, model, price_idr, monthly_token_limit,
	session_token_limit, weekly_token_limit,
	is_active, web_search_enabled, daily_web_searches, daily_messages`

func (r *PlanRepo) List(ctx context.Context) ([]domain.Plan, error) {
	rows, err := r.db.Query(ctx, `select `+planCols+` from plans where is_active = true order by price_idr`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Plan
	for rows.Next() {
		p, err := scanPlan(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *p)
	}
	return out, rows.Err()
}

func (r *PlanRepo) ByCode(ctx context.Context, code string) (*domain.Plan, error) {
	return scanPlan(r.db.QueryRow(ctx, `select `+planCols+` from plans where code = $1`, code))
}

// idempotent seed upsert
func (r *PlanRepo) Upsert(ctx context.Context, p *domain.Plan) error {
	return r.db.QueryRow(ctx, `
		insert into plans (code, name, model, price_idr, monthly_token_limit, session_token_limit, weekly_token_limit, is_active, web_search_enabled, daily_web_searches, daily_messages)
		values ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		on conflict (code) do update set
			name = excluded.name, model = excluded.model, price_idr = excluded.price_idr,
			web_search_enabled = excluded.web_search_enabled, daily_web_searches = excluded.daily_web_searches,
			daily_messages = excluded.daily_messages,
			monthly_token_limit = excluded.monthly_token_limit,
			session_token_limit = excluded.session_token_limit, weekly_token_limit = excluded.weekly_token_limit,
			is_active = excluded.is_active, updated_at = now()
		returning id`,
		p.Code, p.Name, p.Model, p.PriceIDR, p.MonthlyTokenLimit, p.SessionTokenLimit, p.WeeklyTokenLimit,
		p.IsActive, p.WebSearchEnabled, p.DailyWebSearches, p.DailyMessages).Scan(&p.ID)
}

// ubah limit window tanpa menyentuh kolom lain (admin, update8 F4). nil = tak diubah.
func (r *PlanRepo) UpdateLimits(ctx context.Context, code string, monthly, session, weekly *int64) error {
	ct, err := r.db.Exec(ctx, `
		update plans set
			monthly_token_limit = coalesce($2, monthly_token_limit),
			session_token_limit = coalesce($3, session_token_limit),
			weekly_token_limit  = coalesce($4, weekly_token_limit),
			updated_at = now()
		where code = $1`,
		code, monthly, session, weekly)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func scanPlan(row pgx.Row) (*domain.Plan, error) {
	var p domain.Plan
	err := row.Scan(&p.ID, &p.Code, &p.Name, &p.Model, &p.PriceIDR, &p.MonthlyTokenLimit,
		&p.SessionTokenLimit, &p.WeeklyTokenLimit,
		&p.IsActive, &p.WebSearchEnabled, &p.DailyWebSearches, &p.DailyMessages)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

type SubscriptionRepo struct{ db *pgxpool.Pool }

func NewSubscriptionRepo(db *pgxpool.Pool) *SubscriptionRepo { return &SubscriptionRepo{db} }

func (r *SubscriptionRepo) ByOrg(ctx context.Context, orgID uuid.UUID) (*domain.SubscriptionView, error) {
	var v domain.SubscriptionView
	err := r.db.QueryRow(ctx, `
		select s.id, s.organization_id, s.plan_id, s.seats, s.current_period_end, s.session_started_at, s.created_at, s.updated_at,
		       p.id, p.code, p.name, p.model, p.price_idr, p.monthly_token_limit, p.session_token_limit, p.weekly_token_limit, p.is_active, p.web_search_enabled, p.daily_web_searches, p.daily_messages
		from subscriptions s join plans p on p.id = s.plan_id
		where s.organization_id = $1`, orgID).Scan(
		&v.ID, &v.OrganizationID, &v.PlanID, &v.Seats, &v.CurrentPeriodEnd, &v.SessionStartedAt, &v.CreatedAt, &v.UpdatedAt,
		&v.Plan.ID, &v.Plan.Code, &v.Plan.Name, &v.Plan.Model, &v.Plan.PriceIDR, &v.Plan.MonthlyTokenLimit, &v.Plan.SessionTokenLimit, &v.Plan.WeeklyTokenLimit, &v.Plan.IsActive, &v.Plan.WebSearchEnabled, &v.Plan.DailyWebSearches, &v.Plan.DailyMessages)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &v, nil
}

// mulai window session baru (update8). Idempoten: dipanggil hanya saat session expired.
func (r *SubscriptionRepo) SetSessionStarted(ctx context.Context, orgID uuid.UUID, at time.Time) error {
	_, err := r.db.Exec(ctx, `
		update subscriptions set session_started_at = $2, updated_at = now()
		where organization_id = $1`, orgID, at)
	return err
}

func (r *SubscriptionRepo) Upsert(ctx context.Context, s *domain.Subscription) error {
	return r.db.QueryRow(ctx, `
		insert into subscriptions (organization_id, plan_id, seats, current_period_end)
		values ($1, $2, $3, $4)
		on conflict (organization_id) do update set
			plan_id = excluded.plan_id, seats = excluded.seats,
			current_period_end = excluded.current_period_end, updated_at = now()
		returning id, created_at, updated_at`,
		s.OrganizationID, s.PlanID, s.Seats, s.CurrentPeriodEnd).Scan(&s.ID, &s.CreatedAt, &s.UpdatedAt)
}

type PromptRepo struct{ db *pgxpool.Pool }

func NewPromptRepo(db *pgxpool.Pool) *PromptRepo { return &PromptRepo{db} }

func (r *PromptRepo) Get(ctx context.Context, key string) (*domain.Prompt, error) {
	var p domain.Prompt
	err := r.db.QueryRow(ctx, `
		select key, content, updated_by, updated_at from prompts where key = $1`, key).
		Scan(&p.Key, &p.Content, &p.UpdatedBy, &p.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *PromptRepo) Set(ctx context.Context, key, content string, updatedBy uuid.UUID) error {
	_, err := r.db.Exec(ctx, `
		insert into prompts (key, content, updated_by) values ($1, $2, $3)
		on conflict (key) do update set content = excluded.content, updated_by = excluded.updated_by, updated_at = now()`,
		key, content, updatedBy)
	return err
}

type UsageRepo struct{ db *pgxpool.Pool }

func NewUsageRepo(db *pgxpool.Pool) *UsageRepo { return &UsageRepo{db} }

func (r *UsageRepo) OrgTokens(ctx context.Context, orgID uuid.UUID, from, to time.Time) (int64, error) {
	var total int64
	err := r.db.QueryRow(ctx, `
		select coalesce(sum(input_tokens + output_tokens), 0)
		from token_usage where organization_id = $1 and created_at >= $2 and created_at < $3`,
		orgID, from, to).Scan(&total)
	return total, err
}

// ModelTokens: total input/output per model LINTAS SEMUA org (saldo Maia global,
// 1 API key). Dipakai estimasi saldo (update6 §4.1).
func (r *UsageRepo) ModelTokens(ctx context.Context) ([]domain.ModelUsage, error) {
	rows, err := r.db.Query(ctx, `
		select model, coalesce(sum(input_tokens), 0), coalesce(sum(output_tokens), 0)
		from token_usage group by model`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.ModelUsage
	for rows.Next() {
		var m domain.ModelUsage
		if err := rows.Scan(&m.Model, &m.InputTokens, &m.OutputTokens); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func (r *UsageRepo) CountMembers(ctx context.Context, orgID uuid.UUID) (int, error) {
	var n int
	err := r.db.QueryRow(ctx, `select count(*) from memberships where organization_id = $1`, orgID).Scan(&n)
	return n, err
}

func (r *UsageRepo) ChatsSince(ctx context.Context, orgID uuid.UUID, t time.Time) (int, error) {
	var n int
	err := r.db.QueryRow(ctx, `
		select count(*) from chats where organization_id = $1 and created_at >= $2 and deleted_at is null`,
		orgID, t).Scan(&n)
	return n, err
}

func (r *UsageRepo) DocCounts(ctx context.Context, orgID uuid.UUID) (indexed, total int, err error) {
	err = r.db.QueryRow(ctx, `
		select count(*) filter (where status = 'indexed'), count(*)
		from documents where organization_id = $1`, orgID).Scan(&indexed, &total)
	return
}
