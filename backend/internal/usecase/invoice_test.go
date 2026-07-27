package usecase

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/lexora/backend/internal/domain"
)

type fakeInvoiceRepo struct {
	invoices   map[uuid.UUID]*domain.Invoice
	candidates []domain.RenewalCandidate
}

func newFakeInvoiceRepo(cands ...domain.RenewalCandidate) *fakeInvoiceRepo {
	return &fakeInvoiceRepo{invoices: make(map[uuid.UUID]*domain.Invoice), candidates: cands}
}

func (f *fakeInvoiceRepo) Create(_ context.Context, inv *domain.Invoice) error {
	if inv.Type == "" {
		inv.Type = domain.InvoiceTypeSubscription
	}
	// dedup pending hanya untuk subscription (cocok dgn partial unique di DB)
	if inv.Type == domain.InvoiceTypeSubscription {
		for _, e := range f.invoices {
			if e.OrganizationID == inv.OrganizationID && e.Status == domain.InvoicePending && e.Type == domain.InvoiceTypeSubscription {
				return domain.ErrConflict
			}
		}
	}
	inv.ID = uuid.New()
	inv.CreatedAt = time.Now()
	cp := *inv
	f.invoices[inv.ID] = &cp
	return nil
}

func (f *fakeInvoiceRepo) ByID(_ context.Context, id uuid.UUID) (*domain.Invoice, error) {
	inv, ok := f.invoices[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	cp := *inv
	return &cp, nil
}

func (f *fakeInvoiceRepo) ListByOrg(_ context.Context, _ uuid.UUID) ([]domain.Invoice, error) {
	return nil, nil
}

func (f *fakeInvoiceRepo) DueRenewals(_ context.Context, _ time.Time) ([]domain.RenewalCandidate, error) {
	return f.candidates, nil
}

func (f *fakeInvoiceRepo) MarkPaid(_ context.Context, id uuid.UUID, at time.Time) (*domain.Invoice, bool, error) {
	inv, ok := f.invoices[id]
	if !ok {
		return nil, false, domain.ErrNotFound
	}
	if inv.Status != domain.InvoicePending {
		cp := *inv
		return &cp, false, nil
	}
	inv.Status = domain.InvoicePaid
	inv.PaidAt = &at
	cp := *inv
	return &cp, true, nil
}

// U6: bayar pending → changed=true, status=paid; bayar ulang → changed=false (idempoten)
func TestMarkPaidIdempotent(t *testing.T) {
	repo := newFakeInvoiceRepo()
	inv := &domain.Invoice{
		OrganizationID: uuid.New(), PlanID: uuid.New(),
		Seats: 1, AmountIDR: 100_000,
		PeriodStart: time.Now(), PeriodEnd: time.Now().AddDate(0, 1, 0),
		Status: domain.InvoicePending,
	}
	if err := repo.Create(context.Background(), inv); err != nil {
		t.Fatal(err)
	}
	uc := NewInvoice(repo)
	paid, changed, err := uc.MarkPaid(context.Background(), inv.ID, time.Now())
	if err != nil || !changed || paid.Status != domain.InvoicePaid || paid.PaidAt == nil {
		t.Fatalf("bayar pertama: changed=%v status=%q err=%v", changed, paid.Status, err)
	}
	_, changed2, err := uc.MarkPaid(context.Background(), inv.ID, time.Now())
	if err != nil || changed2 {
		t.Fatalf("bayar ulang harus changed=false, dapat changed=%v err=%v", changed2, err)
	}
}

// U7: ticker 2x → tetap satu pending (ErrConflict diabaikan diam)
func TestCreateRenewalsIdempotent(t *testing.T) {
	cand := domain.RenewalCandidate{
		OrganizationID: uuid.New(), PlanID: uuid.New(),
		Seats: 1, PriceIDR: 50_000, PeriodEnd: time.Now().AddDate(0, 0, 3),
	}
	uc := NewInvoice(newFakeInvoiceRepo(cand))
	if n, err := uc.CreateRenewals(context.Background(), time.Now()); err != nil || n != 1 {
		t.Fatalf("pertama harus 1, dapat n=%d err=%v", n, err)
	}
	if n, err := uc.CreateRenewals(context.Background(), time.Now()); err != nil || n != 0 {
		t.Fatalf("kedua harus 0 (conflict diabaikan), dapat n=%d err=%v", n, err)
	}
}

// U8: Demo (price_idr=0 → tidak masuk DueRenewals) → tidak ada invoice terbit
func TestDemoNeverGetInvoice(t *testing.T) {
	uc := NewInvoice(newFakeInvoiceRepo()) // tanpa kandidat
	if n, err := uc.CreateRenewals(context.Background(), time.Now()); err != nil || n != 0 {
		t.Fatalf("Demo tidak boleh dapat invoice, dapat n=%d err=%v", n, err)
	}
}

type fakeTopups struct {
	creates int
	sum     int64
}

func (f *fakeTopups) Create(_ context.Context, _ *domain.QuotaTopup) error { f.creates++; return nil }
func (f *fakeTopups) SumTokens(_ context.Context, _ uuid.UUID, _, _ time.Time) (int64, error) {
	return f.sum, nil
}

// U5: top-up lunas → limit efektif naik sebesar SUM tokens window bulan ini
func TestTopupRaisesEffectiveLimit(t *testing.T) {
	// limit dasar 1000; used 900 → tanpa top-up sudah soft (>=80%)
	b := NewBilling(&fakeSubs{sub: subView(1000, 1)}, &fakeUsage{tokens: 900})
	b.SetTopup(&fakeTopups{sum: 1000}) // +1000 → limit 2000
	q, err := b.Usage(context.Background(), uuid.New(), time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if q.Limit != 2000 {
		t.Fatalf("mau limit 2000 (1000 plan + 1000 topup), dapat %d", q.Limit)
	}
	if q.Soft || q.Hard {
		t.Fatalf("900/2000 belum 80%%, tak boleh soft/hard: soft=%v hard=%v", q.Soft, q.Hard)
	}
}

// U6-topup: mark-paid invoice top-up → insert satu QuotaTopup; bayar ulang tak insert lagi
func TestMarkPaidTopupInsertsOnce(t *testing.T) {
	repo := newFakeInvoiceRepo()
	inv := &domain.Invoice{
		OrganizationID: uuid.New(), PlanID: uuid.New(),
		Type: domain.InvoiceTypeTopup, PackageCode: "small",
		Seats: 1, AmountIDR: domain.TopupPackages["small"].PriceIDR,
		PeriodStart: time.Now(), PeriodEnd: time.Now(),
		Status: domain.InvoicePending,
	}
	if err := repo.Create(context.Background(), inv); err != nil {
		t.Fatal(err)
	}
	topups := &fakeTopups{}
	uc := NewInvoice(repo)
	uc.SetTopup(&fakeSubs{sub: subView(1000, 1)}, topups)

	if _, changed, err := uc.MarkPaid(context.Background(), inv.ID, time.Now()); err != nil || !changed {
		t.Fatalf("bayar pertama: changed=%v err=%v", changed, err)
	}
	if _, changed, err := uc.MarkPaid(context.Background(), inv.ID, time.Now()); err != nil || changed {
		t.Fatalf("bayar ulang harus changed=false: changed=%v err=%v", changed, err)
	}
	if topups.creates != 1 {
		t.Fatalf("QuotaTopup harus dibuat tepat sekali, dapat %d", topups.creates)
	}
}
