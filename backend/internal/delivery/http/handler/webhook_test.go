package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/lexora/backend/internal/domain"
	"github.com/lexora/backend/internal/usecase"
)

// fake invoice repo minimal untuk test webhook (subset InvoiceRepository)
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

func (a *noopAuditRepo) Insert(context.Context, domain.AuditLog) error { a.n++; return nil }
func (a *noopAuditRepo) Recent(context.Context, int) ([]domain.AuditLog, error) { return nil, nil }

func webhookAPI(inv *domain.Invoice, token string) (*InvoiceAPI, *whRepo) {
	repo := &whRepo{inv: inv}
	uc := usecase.NewInvoice(repo)
	api := NewInvoiceAPI(uc, usecase.NewAudit(&noopAuditRepo{}))
	api.SetCallbackToken(token)
	return api, repo
}

func post(api *InvoiceAPI, token, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest("POST", "/webhooks/xendit", strings.NewReader(body))
	if token != "" {
		req.Header.Set("x-callback-token", token)
	}
	w := httptest.NewRecorder()
	api.xenditWebhook(w, req)
	return w
}

func pendingInvoice() *domain.Invoice {
	return &domain.Invoice{
		ID: uuid.New(), OrganizationID: uuid.New(), Type: domain.InvoiceTypeSubscription,
		AmountIDR: 275_000, Status: domain.InvoicePending,
		PeriodStart: time.Now(), PeriodEnd: time.Now().AddDate(0, 1, 0),
	}
}

// U10: token salah -> 401, MarkPaid tak dipanggil
func TestWebhookBadToken(t *testing.T) {
	inv := pendingInvoice()
	api, repo := webhookAPI(inv, "secret-token")
	w := post(api, "wrong-token", `{"external_id":"`+inv.ID.String()+`","status":"PAID","amount":275000}`)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("token salah harus 401, dapat %d", w.Code)
	}
	if repo.markCalls != 0 {
		t.Fatal("token salah tak boleh sampai MarkPaid")
	}
}

// token kosong (belum dikonfigurasi) -> tolak
func TestWebhookNoTokenConfigured(t *testing.T) {
	inv := pendingInvoice()
	api, _ := webhookAPI(inv, "") // callbackToken kosong
	w := post(api, "apa-saja", `{"external_id":"`+inv.ID.String()+`","status":"PAID"}`)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("token kosong harus tolak semua, dapat %d", w.Code)
	}
}

// U11: webhook 2x -> extend/token sekali (idempoten via changed=false)
func TestWebhookIdempotent(t *testing.T) {
	inv := pendingInvoice()
	api, _ := webhookAPI(inv, "tok")
	body := `{"external_id":"` + inv.ID.String() + `","status":"PAID","amount":275000}`

	if w := post(api, "tok", body); w.Code != http.StatusOK {
		t.Fatalf("bayar pertama harus 200, dapat %d", w.Code)
	}
	if inv.Status != domain.InvoicePaid {
		t.Fatal("invoice harus paid setelah webhook pertama")
	}
	if w := post(api, "tok", body); w.Code != http.StatusOK {
		t.Fatalf("webhook kedua harus tetap 200, dapat %d", w.Code)
	}
	// MarkPaid boleh dipanggil 2x, tapi transisi (changed) cuma sekali -> status tetap paid
	if inv.Status != domain.InvoicePaid {
		t.Fatal("status harus tetap paid (tak berubah di webhook kedua)")
	}
}

// U12: amount beda -> TIDAK paid
func TestWebhookAmountMismatch(t *testing.T) {
	inv := pendingInvoice()
	api, repo := webhookAPI(inv, "tok")
	// amount 999999 != 275000
	w := post(api, "tok", `{"external_id":"`+inv.ID.String()+`","status":"PAID","amount":999999}`)
	if w.Code != http.StatusOK {
		t.Fatalf("amount beda harus 200 (stop retry), dapat %d", w.Code)
	}
	if repo.markCalls != 0 {
		t.Fatal("amount beda tak boleh MarkPaid")
	}
	if inv.Status != domain.InvoicePending {
		t.Fatal("amount beda: invoice harus tetap pending")
	}
}

// status non-paid (expired) -> 200, tak proses
func TestWebhookNonPaidStatus(t *testing.T) {
	inv := pendingInvoice()
	api, repo := webhookAPI(inv, "tok")
	w := post(api, "tok", `{"external_id":"`+inv.ID.String()+`","status":"EXPIRED"}`)
	if w.Code != http.StatusOK || repo.markCalls != 0 {
		t.Fatalf("status non-paid: harus 200 tanpa MarkPaid; code=%d calls=%d", w.Code, repo.markCalls)
	}
}
