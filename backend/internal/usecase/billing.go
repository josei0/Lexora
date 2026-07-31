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
	// Limit/Used = window MONTHLY (kompat handler + overflow lama). Multi-window di Windows.
	Limit    int64 // 0 = unlimited
	Used     int64
	Soft     bool         // agregat: >= 80% di SALAH SATU window aktif
	Hard     bool         // agregat: >= 100% di SALAH SATU window aktif (gaya Claude)
	Overflow bool         // monthly >= 2x limit: plafon degrade absolut
	Plan     *domain.Plan // nil = tanpa subscription
	Status   string       // active | past_due | expired

	// window multi-limit (update8). Kosong = tanpa subscription. Session/Weekly/Monthly.
	Windows []WindowUsage
}

// MostConstrained: window aktif dengan sisa token terkecil (untuk pesan "limit X, reset Y").
// nil kalau tak ada window aktif.
func (q Quota) MostConstrained() *WindowUsage {
	var best *WindowUsage
	for i := range q.Windows {
		w := &q.Windows[i]
		if !w.Active() {
			continue
		}
		if best == nil || w.Remaining() < best.Remaining() {
			best = w
		}
	}
	return best
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

// week window [Senin 00:00 WIB minggu ini, Senin depan). Reset kalender (update8 §6).
func weekWindow(now time.Time) (from, to time.Time) {
	n := now.In(wib)
	midnight := time.Date(n.Year(), n.Month(), n.Day(), 0, 0, 0, 0, wib)
	// Go: Sunday=0..Saturday=6. Mundur ke Senin: Senin=1 -> 0 hari, Minggu=0 -> 6 hari.
	back := (int(n.Weekday()) + 6) % 7
	from = midnight.AddDate(0, 0, -back)
	to = from.AddDate(0, 0, 7)
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
	seats := int64(sub.Seats)

	// PAYG lintas-window (update8 §2.3): saldo top-up bulan ini angkat plafon SEMUA
	// window aktif. Dihitung sekali, ditambahkan ke tiap window.
	var payg int64
	if b.topups != nil {
		mFrom, mTo := monthWindow(now)
		if payg, err = b.topups.SumTokens(ctx, orgID, mFrom, mTo); err != nil {
			return Quota{}, err
		}
	}

	// bangun 3 window. limit plan 0 = window nonaktif -> PAYG tak mengaktifkannya.
	specs := []struct {
		kind      WindowKind
		planLimit int64
	}{
		{WindowSession, sub.Plan.SessionTokenLimit},
		{WindowWeekly, sub.Plan.WeeklyTokenLimit},
		{WindowMonthly, sub.Plan.MonthlyTokenLimit},
	}
	windows := make([]WindowUsage, 0, len(specs))
	for _, s := range specs {
		base := s.planLimit * seats
		if base == 0 {
			// nonaktif: tetap catat used (info), tapi Limit 0 -> tak menggating
			windows = append(windows, WindowUsage{Kind: s.kind, Limit: 0})
			continue
		}
		from, reset := windowBounds(s.kind, now, sub.SessionStartedAt)
		used, err := b.usage.OrgTokens(ctx, orgID, from, now)
		if err != nil {
			return Quota{}, err
		}
		windows = append(windows, WindowUsage{
			Kind: s.kind, Limit: base + payg, Used: used, ResetAt: reset,
		})
	}

	q := Quota{Plan: &sub.Plan, Status: sub.StatusAt(now), Windows: windows}
	// agregat: soft/hard = true kalau SALAH SATU window aktif mentok (gaya Claude).
	for i := range windows {
		if windows[i].Soft() {
			q.Soft = true
		}
		if windows[i].Hard() {
			q.Hard = true
		}
	}
	// kompat lama: Limit/Used/Overflow = window monthly (dipakai handler + degrade absolut).
	if m := findWindow(windows, WindowMonthly); m != nil && m.Active() {
		q.Limit, q.Used = m.Limit, m.Used
		q.Overflow = m.Used >= m.Limit*overflowFactor
	}
	return q, nil
}

func findWindow(ws []WindowUsage, kind WindowKind) *WindowUsage {
	for i := range ws {
		if ws[i].Kind == kind {
			return &ws[i]
		}
	}
	return nil
}

// dipanggil sebelum Ask. Hard TIDAK error di sini — RAG yang memblokir (update8:
// window manapun mentok = blok). Overflow = plafon absolut, tetap error di sini.
// Efek samping disengaja: memulai window session baru kalau session lama sudah lewat.
func (b *Billing) Check(ctx context.Context, orgID uuid.UUID, now time.Time) (Quota, error) {
	// session window (update8): pesan pertama / pasca-idle >5j -> mulai session baru.
	// Dilakukan SEBELUM usage_ supaya perhitungan pakai window yang benar.
	if err := b.rollSession(ctx, orgID, now); err != nil {
		return Quota{}, err
	}
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

// rollSession: mulai window session baru kalau belum ada / sudah lewat 5 jam.
// No-op kalau plan tak pakai window session (SessionTokenLimit 0) — hemat write.
func (b *Billing) rollSession(ctx context.Context, orgID uuid.UUID, now time.Time) error {
	sub, err := b.subs.ByOrg(ctx, orgID)
	if err == domain.ErrNotFound {
		return nil // tanpa subscription: tak ada window
	}
	if err != nil {
		return err
	}
	if sub.Plan.SessionTokenLimit <= 0 {
		return nil // window session nonaktif untuk plan ini
	}
	if !sessionExpired(now, sub.SessionStartedAt) {
		return nil // session masih jalan
	}
	return b.subs.SetSessionStarted(ctx, orgID, now)
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
