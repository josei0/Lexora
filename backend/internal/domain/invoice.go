package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// status invoice fisik (beda dgn status subscription yg dihitung dari tanggal):
// invoice = kejadian pembayaran diskret, statusnya memang tersimpan.
const (
	InvoicePending = "pending"
	InvoicePaid    = "paid"
	InvoiceExpired = "expired"
	InvoiceVoid    = "void"

	InvoiceTypeSubscription = "subscription"
	InvoiceTypeTopup        = "topup"

	// H-7: invoice renewal terbit tujuh hari sebelum masa aktif habis
	RenewalLeadDays = 7
)

type Invoice struct {
	ID             uuid.UUID  `json:"id"`
	OrganizationID uuid.UUID  `json:"organization_id"`
	PlanID         uuid.UUID  `json:"plan_id"`
	Type           string     `json:"type"` // subscription | topup
	PackageCode    string     `json:"package_code,omitempty"`
	Seats          int        `json:"seats"`
	AmountIDR      int64      `json:"amount_idr"`
	PeriodStart    time.Time  `json:"period_start"`
	PeriodEnd      time.Time  `json:"period_end"`
	Status         string     `json:"status"`
	Provider       *string    `json:"provider,omitempty"`
	ProviderID     *string    `json:"provider_id,omitempty"`
	CheckoutURL    *string    `json:"checkout_url,omitempty"`
	PaidAt         *time.Time `json:"paid_at,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
}

// kandidat renewal: subscription berbayar yg masa aktifnya hampir habis.
// price_idr > 0 memastikan Demo tak pernah ditagih.
type RenewalCandidate struct {
	OrganizationID uuid.UUID
	PlanID         uuid.UUID
	Seats          int
	PriceIDR       int64
	PeriodEnd      time.Time
}

type InvoiceRepository interface {
	Create(ctx context.Context, inv *Invoice) error
	ByID(ctx context.Context, id uuid.UUID) (*Invoice, error)
	ListByOrg(ctx context.Context, orgID uuid.UUID) ([]Invoice, error)
	// subscription berbayar, period_end <= cutoff, belum punya invoice pending
	DueRenewals(ctx context.Context, cutoff time.Time) ([]RenewalCandidate, error)
	// tandai paid + extend subscription.current_period_end = invoice.period_end,
	// satu transaksi. Idempoten: invoice yg sudah paid tak diproses ulang.
	// Balikan bool = true kalau transisi pending->paid benar-benar terjadi.
	MarkPaid(ctx context.Context, id uuid.UUID, at time.Time) (*Invoice, bool, error)
}
