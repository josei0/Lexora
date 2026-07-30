package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/lexora/backend/internal/domain"
	"github.com/lexora/backend/pkg/payment"
)

// Invoice: terbitkan tagihan renewal H-7 + tandai lunas (manual super_admin / webhook).
type Invoice struct {
	repo   domain.InvoiceRepository
	subs   domain.SubscriptionRepository // resolve plan_id org untuk topup
	topups domain.TopupRepository        // nil = topup belum aktif
	pay    payment.Gateway               // nil = mode manual (dev/test); tak buat checkout
}

func NewInvoice(repo domain.InvoiceRepository) *Invoice { return &Invoice{repo: repo} }

func (u *Invoice) SetTopup(subs domain.SubscriptionRepository, r domain.TopupRepository) {
	u.subs, u.topups = subs, r
}

// SetGateway: aktifkan pembuatan checkout Xendit. Nil = mode manual lama.
func (u *Invoice) SetGateway(g payment.Gateway) { u.pay = g }

// createCheckout: best-effort. Gateway nil / gagal -> invoice tetap ada tanpa URL
// (user klik bayar lagi nanti). Tidak pernah menggagalkan pembuatan invoice.
func (u *Invoice) createCheckout(ctx context.Context, inv *domain.Invoice, payerEmail string) {
	if u.pay == nil || inv.Status != domain.InvoicePending {
		return
	}
	c, err := u.pay.CreateInvoice(ctx, payment.Invoice{
		ExternalID:  inv.ID.String(),
		AmountIDR:   inv.AmountIDR,
		PayerEmail:  payerEmail,
		Description: describeInvoice(inv),
	})
	if err != nil {
		return // invoice tetap pending tanpa URL; nyusul saat bayar ulang
	}
	if err := u.repo.SetProvider(ctx, inv.ID, "xendit", c.ProviderID, c.CheckoutURL); err == nil {
		xendit := "xendit"
		inv.Provider, inv.ProviderID, inv.CheckoutURL = &xendit, &c.ProviderID, &c.CheckoutURL
	}
}

func describeInvoice(inv *domain.Invoice) string {
	if inv.Type == domain.InvoiceTypeTopup {
		if pkg, ok := domain.TopupPackages[inv.PackageCode]; ok {
			return "Top-up " + pkg.LabelShort
		}
		return "Top-up kuota"
	}
	return fmt.Sprintf("Perpanjangan langganan (%d seat)", inv.Seats)
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
		u.createCheckout(ctx, inv, "") // best-effort checkout URL
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
	u.createCheckout(ctx, inv, "") // best-effort; isi checkout_url kalau gateway aktif
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
