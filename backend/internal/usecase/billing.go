package usecase

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/lexora/backend/internal/domain"
)

// WIB = UTC+7; month window dihitung di zona ini
var wib = time.FixedZone("WIB", 7*3600)

const softThreshold = 0.8 // 80%

// Billing: kuota token per org per bulan (limit = plan.MonthlyTokenLimit * seats).
// Org tanpa subscription = tanpa limit (belum onboard billing).
type Billing struct {
	subs  domain.SubscriptionRepository
	usage domain.UsageRepository
}

func NewBilling(subs domain.SubscriptionRepository, usage domain.UsageRepository) *Billing {
	return &Billing{subs: subs, usage: usage}
}

type Quota struct {
	Limit int64 // 0 = unlimited
	Used  int64
	Soft  bool // >= 80%
	Hard  bool // >= 100%
}

func (q Quota) Remaining() int64 {
	if q.Limit == 0 {
		return -1
	}
	if q.Used >= q.Limit {
		return 0
	}
	return q.Limit - q.Used
}

// month window [awal bulan WIB, awal bulan depan WIB)
func monthWindow(now time.Time) (from, to time.Time) {
	n := now.In(wib)
	from = time.Date(n.Year(), n.Month(), 1, 0, 0, 0, 0, wib)
	to = from.AddDate(0, 1, 0)
	return
}

func (b *Billing) usage_(ctx context.Context, orgID uuid.UUID, now time.Time) (Quota, error) {
	sub, err := b.subs.ByOrg(ctx, orgID)
	if err == domain.ErrNotFound {
		return Quota{}, nil // no subscription -> unlimited
	}
	if err != nil {
		return Quota{}, err
	}
	limit := sub.Plan.MonthlyTokenLimit * int64(sub.Seats)

	from, to := monthWindow(now)
	used, err := b.usage.OrgTokens(ctx, orgID, from, to)
	if err != nil {
		return Quota{}, err
	}
	q := Quota{Limit: limit, Used: used}
	if limit > 0 {
		q.Soft = used >= int64(float64(limit)*softThreshold)
		q.Hard = used >= limit
	}
	return q, nil
}

// dipanggil sebelum Ask; error kalau hard limit tembus
func (b *Billing) Check(ctx context.Context, orgID uuid.UUID, now time.Time) (Quota, error) {
	q, err := b.usage_(ctx, orgID, now)
	if err != nil {
		return q, err
	}
	if q.Hard {
		return q, domain.ErrQuotaExceeded
	}
	return q, nil
}

// snapshot kuota untuk ditampilkan (tanpa error hard)
func (b *Billing) Usage(ctx context.Context, orgID uuid.UUID, now time.Time) (Quota, error) {
	return b.usage_(ctx, orgID, now)
}
