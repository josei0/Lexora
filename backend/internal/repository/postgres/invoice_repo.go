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

type InvoiceRepo struct{ db *pgxpool.Pool }

func NewInvoiceRepo(db *pgxpool.Pool) *InvoiceRepo { return &InvoiceRepo{db} }

func (r *InvoiceRepo) Create(ctx context.Context, inv *domain.Invoice) error {
	typ := inv.Type
	if typ == "" {
		typ = domain.InvoiceTypeSubscription // zero value -> default (renewal)
	}
	// package_code kosong -> NULL (kolom nullable; hanya top-up yang mengisinya)
	var pkg *string
	if inv.PackageCode != "" {
		pkg = &inv.PackageCode
	}
	err := r.db.QueryRow(ctx, `
		insert into invoices (organization_id, plan_id, type, package_code, seats, amount_idr, period_start, period_end, status)
		values ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		returning id, created_at`,
		inv.OrganizationID, inv.PlanID, typ, pkg, inv.Seats, inv.AmountIDR, inv.PeriodStart, inv.PeriodEnd, inv.Status,
	).Scan(&inv.ID, &inv.CreatedAt)
	if err == nil {
		inv.Type = typ // sinkronkan struct dgn yang tersimpan
	}
	return mapErr(err) // unique pending -> ErrConflict
}

func (r *InvoiceRepo) ByID(ctx context.Context, id uuid.UUID) (*domain.Invoice, error) {
	return scanInvoice(r.db.QueryRow(ctx, invoiceCols+` where id = $1`, id))
}

// korelasi webhook: cari invoice by id transaksi gateway (provider_id).
func (r *InvoiceRepo) ByProviderID(ctx context.Context, providerID string) (*domain.Invoice, error) {
	return scanInvoice(r.db.QueryRow(ctx, invoiceCols+` where provider_id = $1`, providerID))
}

func (r *InvoiceRepo) ListByOrg(ctx context.Context, orgID uuid.UUID) ([]domain.Invoice, error) {
	rows, err := r.db.Query(ctx, invoiceCols+` where organization_id = $1 order by created_at desc`, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Invoice
	for rows.Next() {
		inv, err := scanInvoice(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *inv)
	}
	return out, rows.Err()
}

func (r *InvoiceRepo) DueRenewals(ctx context.Context, cutoff time.Time) ([]domain.RenewalCandidate, error) {
	rows, err := r.db.Query(ctx, `
		select s.organization_id, s.plan_id, s.seats, p.price_idr, s.current_period_end
		from subscriptions s
		join plans p on p.id = s.plan_id
		where p.price_idr > 0
		  and s.current_period_end is not null
		  and s.current_period_end <= $1
		  and not exists (
		    select 1 from invoices i
		    where i.organization_id = s.organization_id and i.status = 'pending' and i.type = 'subscription'
		  )`, cutoff)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.RenewalCandidate
	for rows.Next() {
		var c domain.RenewalCandidate
		if err := rows.Scan(&c.OrganizationID, &c.PlanID, &c.Seats, &c.PriceIDR, &c.PeriodEnd); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// paid + extend dalam satu transaksi. Idempoten: invoice non-pending -> no-op.
func (r *InvoiceRepo) MarkPaid(ctx context.Context, id uuid.UUID, at time.Time) (*domain.Invoice, bool, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, false, err
	}
	defer tx.Rollback(ctx)

	inv, err := scanInvoice(tx.QueryRow(ctx, invoiceCols+` where id = $1 for update`, id))
	if err != nil {
		return nil, false, err
	}
	if inv.Status != domain.InvoicePending {
		return inv, false, nil // sudah paid/void: idempoten
	}

	if _, err := tx.Exec(ctx, `update invoices set status = 'paid', paid_at = $2 where id = $1`, id, at); err != nil {
		return nil, false, err
	}
	// HANYA subscription yang extend masa aktif. Top-up isi period_end=now (tak bermakna) -
	// meng-extend pakai itu = nuke langganan (Bug B). Top-up cukup inject token di usecase.
	if inv.Type == domain.InvoiceTypeSubscription {
		// extend absolut ke period_end invoice (= period_end lama + 1 bulan, dihitung saat terbit):
		// bayar cepat tidak memotong masa aktif, dan operasi ini idempoten.
		if _, err := tx.Exec(ctx, `
			update subscriptions set current_period_end = $2, updated_at = now()
			where organization_id = $1`, inv.OrganizationID, inv.PeriodEnd); err != nil {
			return nil, false, err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, false, err
	}
	inv.Status = domain.InvoicePaid
	inv.PaidAt = &at
	return inv, true, nil
}

// simpan hasil gateway (update6 §3.3). Hanya invoice pending yang relevan,
// tapi UPDATE tak dibatasi status - idempoten kalau dipanggil ulang.
func (r *InvoiceRepo) SetProvider(ctx context.Context, id uuid.UUID, provider, providerID, checkoutURL string) error {
	_, err := r.db.Exec(ctx, `
		update invoices set provider = $2, provider_id = $3, checkout_url = $4
		where id = $1`, id, provider, providerID, checkoutURL)
	return err
}

const invoiceCols = `
	select id, organization_id, plan_id, type, coalesce(package_code,''), seats, amount_idr, period_start, period_end,
	       status, provider, provider_id, checkout_url, paid_at, created_at
	from invoices`

func scanInvoice(row pgx.Row) (*domain.Invoice, error) {
	var inv domain.Invoice
	err := row.Scan(&inv.ID, &inv.OrganizationID, &inv.PlanID, &inv.Type, &inv.PackageCode,
		&inv.Seats, &inv.AmountIDR, &inv.PeriodStart, &inv.PeriodEnd,
		&inv.Status, &inv.Provider, &inv.ProviderID, &inv.CheckoutURL, &inv.PaidAt, &inv.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &inv, nil
}
