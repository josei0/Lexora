package usecase

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/lexora/backend/internal/domain"
)

// Invoice: terbitkan tagihan renewal H-7 + tandai lunas (manual super_admin / webhook fase 12).
// Gateway (Xendit) belum disentuh di sini - jalur manual dulu, sales-led.
type Invoice struct {
	repo   domain.InvoiceRepository
	subs   domain.SubscriptionRepository // resolve plan_id org untuk topup
	topups domain.TopupRepository        // nil = topup belum aktif
}

func NewInvoice(repo domain.InvoiceRepository) *Invoice { return &Invoice{repo: repo} }

func (u *Invoice) SetTopup(subs domain.SubscriptionRepository, r domain.TopupRepository) {
	u.subs, u.topups = subs, r
}

// terbitkan invoice untuk subscription berbayar yg masa aktifnya <= H-7 dan belum
// punya invoice pending. Dipanggil ticker harian. Balikan = jumlah invoice baru.
func (u *Invoice) CreateRenewals(ctx context.Context, now time.Time) (int, error) {
	cutoff := now.AddDate(0, 0, domain.RenewalLeadDays)
	cands, err := u.repo.DueRenewals(ctx, cutoff)
	if err != nil {
		return 0, err
	}
	n := 0
	for _, c := range cands {
		inv := &domain.Invoice{
			OrganizationID: c.OrganizationID,
			PlanID:         c.PlanID,
			Seats:          c.Seats,
			AmountIDR:      c.PriceIDR * int64(c.Seats), // dihitung server, tak pernah dari FE
			PeriodStart:    c.PeriodEnd,
			PeriodEnd:      c.PeriodEnd.AddDate(0, 1, 0), // extend dari period_end lama
			Status:         domain.InvoicePending,
		}
		// balapan dgn ticker lain / klik user: partial unique menolak dobel -> lewati diam
		if err := u.repo.Create(ctx, inv); err == domain.ErrConflict {
			continue
		} else if err != nil {
			return n, err
		}
		n++
	}
	return n, nil
}

func (u *Invoice) ListByOrg(ctx context.Context, orgID uuid.UUID) ([]domain.Invoice, error) {
	return u.repo.ListByOrg(ctx, orgID)
}

// buat invoice top-up pending. Harga + token dari paket (server), tak pernah dari FE.
// plan_id diambil dari subscription org; period_* tidak bermakna (diisi now, kolom NOT NULL).
func (u *Invoice) CreateTopup(ctx context.Context, orgID uuid.UUID, code string, now time.Time) (*domain.Invoice, error) {
	pkg, ok := domain.TopupPackages[code]
	if !ok {
		return nil, domain.ErrNotFound
	}
	sub, err := u.subs.ByOrg(ctx, orgID)
	if err != nil {
		return nil, err // org tanpa subscription tak bisa top-up
	}
	inv := &domain.Invoice{
		OrganizationID: orgID,
		PlanID:         sub.PlanID,
		Type:           domain.InvoiceTypeTopup,
		PackageCode:    code,
		Seats:          1,
		AmountIDR:      pkg.PriceIDR,
		PeriodStart:    now,
		PeriodEnd:      now,
		Status:         domain.InvoicePending,
	}
	if err := u.repo.Create(ctx, inv); err != nil {
		return nil, err // ErrConflict kalau sudah ada pending -> caller map 409
	}
	return inv, nil
}

func (u *Invoice) ByID(ctx context.Context, id uuid.UUID) (*domain.Invoice, error) {
	return u.repo.ByID(ctx, id)
}

// tandai lunas + extend masa aktif (subscription) atau insert topup row.
// Balikan bool = transisi benar-benar terjadi (false = sudah paid; caller jangan audit ganda).
func (u *Invoice) MarkPaid(ctx context.Context, id uuid.UUID, now time.Time) (*domain.Invoice, bool, error) {
	inv, changed, err := u.repo.MarkPaid(ctx, id, now)
	if err != nil || !changed {
		return inv, changed, err
	}
	if inv.Type == domain.InvoiceTypeTopup && u.topups != nil {
		pkg, ok := domain.TopupPackages[inv.PackageCode]
		if ok {
			t := &domain.QuotaTopup{OrganizationID: inv.OrganizationID, InvoiceID: inv.ID, Tokens: pkg.Tokens}
			if err := u.topups.Create(ctx, t); err != nil {
				return inv, changed, err
			}
		}
	}
	return inv, changed, nil
}
