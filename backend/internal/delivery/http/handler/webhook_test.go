package handler

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/lexora/backend/internal/domain"
)

// fake invoice repo minimal untuk test webhook (subset InvoiceRepository).
// Dipakai bersama oleh test webhook Mayar (mayar_webhook_test.go).
type whRepo struct {
	inv       *domain.Invoice
	markCalls int
}

func (r *whRepo) Create(context.Context, *domain.Invoice) error { return nil }
func (r *whRepo) ByID(_ context.Context, id uuid.UUID) (*domain.Invoice, error) {
	if r.inv != nil && r.inv.ID == id {
		cp := *r.inv
		return &cp, nil
	}
	return nil, domain.ErrNotFound
}
func (r *whRepo) ListByOrg(context.Context, uuid.UUID) ([]domain.Invoice, error) { return nil, nil }
func (r *whRepo) DueRenewals(context.Context, time.Time) ([]domain.RenewalCandidate, error) {
	return nil, nil
}
func (r *whRepo) SetProvider(context.Context, uuid.UUID, string, string, string) error { return nil }
func (r *whRepo) ByProviderID(_ context.Context, providerID string) (*domain.Invoice, error) {
	if r.inv != nil && r.inv.ProviderID != nil && *r.inv.ProviderID == providerID {
		cp := *r.inv
		return &cp, nil
	}
	return nil, domain.ErrNotFound
}
func (r *whRepo) MarkPaid(_ context.Context, id uuid.UUID, at time.Time) (*domain.Invoice, bool, error) {
	r.markCalls++
	if r.inv == nil || r.inv.ID != id {
		return nil, false, domain.ErrNotFound
	}
	if r.inv.Status != domain.InvoicePending {
		cp := *r.inv
		return &cp, false, nil // idempoten
	}
	r.inv.Status = domain.InvoicePaid
	r.inv.PaidAt = &at
	cp := *r.inv
	return &cp, true, nil
}

type noopAuditRepo struct{ n int }

func (a *noopAuditRepo) Insert(context.Context, domain.AuditLog) error          { a.n++; return nil }
func (a *noopAuditRepo) Recent(context.Context, int) ([]domain.AuditLog, error) { return nil, nil }
