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

// ModelUsage: total token per model lintas org, untuk estimasi saldo Maia.
type ModelUsage struct {
	Model        string
	InputTokens  int64
	OutputTokens int64
}

type Plan struct {
	ID   uuid.UUID `json:"id"`
	Code string    `json:"code"`
	Name string    `json:"name"`
	// json:"-" disengaja: nama model tidak pernah sampai ke klien, user lihat tier
	// "AI High"/"AI Normal" saja. Ganti mesin = ganti env, bukan janji marketing.
	Model             string `json:"-"`
	PriceIDR          int64  `json:"price_idr"`
	MonthlyTokenLimit int64  `json:"monthly_token_limit"` // per seat; 0 = unlimited
	IsActive          bool   `json:"is_active"`
	WebSearchEnabled  bool   `json:"web_search_enabled"`
	DailyWebSearches  int    `json:"daily_web_searches"` // 0 = mati
	DailyMessages     int    `json:"daily_messages"`     // 0 = tanpa cap
}

// status langganan dihitung dari tanggal, bukan disimpan: tidak butuh job
// pengubah status, tidak bisa basi, satu sumber kebenaran.
const (
	SubActive  = "active"   // masih dalam periode
	SubPastDue = "past_due" // lewat periode, masih dalam grace
	SubExpired = "expired"  // lewat grace: read-only

	GraceDays = 7
)

type Subscription struct {
	ID             uuid.UUID `json:"id"`
	OrganizationID uuid.UUID `json:"organization_id"`
	PlanID         uuid.UUID `json:"plan_id"`
	Seats          int       `json:"seats"`
	// nil = tanpa masa aktif (Demo / langganan lama) -> selalu active
	CurrentPeriodEnd *time.Time `json:"current_period_end,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

func (s Subscription) StatusAt(now time.Time) string {
	if s.CurrentPeriodEnd == nil {
		return SubActive
	}
	switch {
	case now.Before(*s.CurrentPeriodEnd):
		return SubActive
	case now.Before(s.CurrentPeriodEnd.AddDate(0, 0, GraceDays)):
		return SubPastDue
	default:
		return SubExpired
	}
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

// paket top-up: harga + token dihitung server, tidak pernah dari FE
type TopupPackage struct {
	Code      string
	Tokens    int64
	PriceIDR  int64
	LabelShort string // "500 ribu token"
}

// dua paket sesuai u5 §3.3; harga final = keputusan client
var TopupPackages = map[string]TopupPackage{
	"small": {Code: "small", Tokens: 500_000, PriceIDR: 79_000, LabelShort: "500 ribu token"},
	"large": {Code: "large", Tokens: 1_000_000, PriceIDR: 149_000, LabelShort: "1 juta token"},
}

type QuotaTopup struct {
	ID             uuid.UUID `json:"id"`
	OrganizationID uuid.UUID `json:"organization_id"`
	InvoiceID      uuid.UUID `json:"invoice_id"`
	Tokens         int64     `json:"tokens"`
	CreatedAt      time.Time `json:"created_at"`
}

type TopupRepository interface {
	// insert satu baris; idempoten via unique(invoice_id)
	Create(ctx context.Context, t *QuotaTopup) error
	// SUM tokens window bulan berjalan
	SumTokens(ctx context.Context, orgID uuid.UUID, from, to time.Time) (int64, error)
}
