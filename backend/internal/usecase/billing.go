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

// ponytail: 2x kuota = plafon degrade Normal, tebakan awal, kalibrasi dgn data
const overflowFactor = 2

// Billing: kuota token per org per bulan (limit = plan.MonthlyTokenLimit * seats).
// Org tanpa subscription = tanpa limit (belum onboard billing).
type Billing struct {
	subs   domain.SubscriptionRepository
	usage  domain.UsageRepository
	topups domain.TopupRepository // nil = top-up belum aktif
}

func NewBilling(subs domain.SubscriptionRepository, usage domain.UsageRepository) *Billing {
	return &Billing{subs: subs, usage: usage}
}

func (b *Billing) SetTopup(r domain.TopupRepository) { b.topups = r }

type Quota struct {
	Limit     int64 // 0 = unlimited
	Used      int64
	Soft      bool   // >= 80%
	Hard      bool   // >= 100%
	Overflow bool         // >= 2x limit: plafon degrade
	Plan     *domain.Plan // nil = tanpa subscription
	Status   string       // active | past_due | expired
}

// awal hari WIB - window kuota harian (search, pesan Demo)
func dayStart(now time.Time) time.Time {
	n := now.In(wib)
	return time.Date(n.Year(), n.Month(), n.Day(), 0, 0, 0, 0, wib)
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
		return Quota{Status: domain.SubActive}, nil // no subscription -> unlimited
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
	// top-up: tambah limit window bulan ini, hangus akhir bulan
	if limit > 0 && b.topups != nil {
		extra, err := b.topups.SumTokens(ctx, orgID, from, to)
		if err != nil {
			return Quota{}, err
		}
		limit += extra
	}
	q := Quota{Limit: limit, Used: used, Plan: &sub.Plan, Status: sub.StatusAt(now)}
	if limit > 0 {
		q.Soft = used >= int64(float64(limit)*softThreshold)
		q.Hard = used >= limit
		q.Overflow = used >= limit*overflowFactor
	}
	return q, nil
}

// dipanggil sebelum Ask. Hard TIDAK lagi error di sini — kebijakan degrade/blok milik RAG
// (plan High degrade ke Normal; plan Normal blok). Overflow = plafon absolut, tetap error.
func (b *Billing) Check(ctx context.Context, orgID uuid.UUID, now time.Time) (Quota, error) {
	q, err := b.usage_(ctx, orgID, now)
	if err != nil {
		return q, err
	}
	if q.Status == domain.SubExpired {
		return q, domain.ErrSubExpired
	}
	if q.Overflow {
		return q, domain.ErrQuotaExceeded
	}
	return q, nil
}

// gate akses fitur menulis (upload, export): hanya masa aktif, bukan kuota token.
// Membaca (chat lama, dokumen, dashboard) sengaja TIDAK digating - data milik org.
func (b *Billing) GateAccess(ctx context.Context, orgID uuid.UUID, now time.Time) error {
	q, err := b.usage_(ctx, orgID, now)
	if err != nil {
		return err
	}
	if q.Status == domain.SubExpired {
		return domain.ErrSubExpired
	}
	return nil
}

// snapshot kuota untuk ditampilkan (tanpa error hard)
func (b *Billing) Usage(ctx context.Context, orgID uuid.UUID, now time.Time) (Quota, error) {
	return b.usage_(ctx, orgID, now)
}
