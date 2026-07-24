package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

const (
	PlanDemo = "demo"
	PlanPro  = "pro"

	PromptSystem = "system"
)

type Plan struct {
	ID                uuid.UUID `json:"id"`
	Code              string    `json:"code"`
	Name              string    `json:"name"`
	Model             string    `json:"model"`
	PriceIDR          int64     `json:"price_idr"`
	MonthlyTokenLimit int64     `json:"monthly_token_limit"` // per seat; 0 = unlimited
	IsActive          bool      `json:"is_active"`
}

type Subscription struct {
	ID             uuid.UUID `json:"id"`
	OrganizationID uuid.UUID `json:"organization_id"`
	PlanID         uuid.UUID `json:"plan_id"`
	Seats          int       `json:"seats"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type SubscriptionView struct {
	Subscription
	Plan Plan `json:"plan"`
}

type Prompt struct {
	Key       string     `json:"key"`
	Content   string     `json:"content"`
	UpdatedBy *uuid.UUID `json:"updated_by,omitempty"`
	UpdatedAt time.Time  `json:"updated_at"`
}

type PlanRepository interface {
	List(ctx context.Context) ([]Plan, error)
	ByCode(ctx context.Context, code string) (*Plan, error)
	Upsert(ctx context.Context, p *Plan) error
}

type SubscriptionRepository interface {
	ByOrg(ctx context.Context, orgID uuid.UUID) (*SubscriptionView, error)
	Upsert(ctx context.Context, s *Subscription) error
}

type PromptRepository interface {
	Get(ctx context.Context, key string) (*Prompt, error)
	Set(ctx context.Context, key, content string, updatedBy uuid.UUID) error
}

type DashboardStats struct {
	ChatsToday  int   `json:"chats_today"`
	TokensMonth int64 `json:"tokens_month"`
	TokenLimit  int64 `json:"token_limit"` // 0 = unlimited
	DocsIndexed int   `json:"docs_indexed"`
	DocsTotal   int   `json:"docs_total"`
	Members     int   `json:"members"`
	Seats       int   `json:"seats"`
}

type UsageRepository interface {
	OrgTokens(ctx context.Context, orgID uuid.UUID, from, to time.Time) (int64, error)
	CountMembers(ctx context.Context, orgID uuid.UUID) (int, error)
	ChatsSince(ctx context.Context, orgID uuid.UUID, t time.Time) (int, error)
	DocCounts(ctx context.Context, orgID uuid.UUID) (indexed, total int, err error)
}
