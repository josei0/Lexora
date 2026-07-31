package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/lexora/backend/internal/domain"
	"github.com/lexora/backend/internal/usecase"
	"github.com/lexora/backend/pkg/payment"
)

// fake verifier: kontrol hasil re-fetch (paid, amount, err) untuk uji tiap cabang.
type fakeVerifier struct {
	paid   bool
	amount int64
	err    error
	calls  int
}

func (f *fakeVerifier) GetInvoice(_ context.Context, id string) (*payment.FetchedInvoice, error) {
	f.calls++
	if f.err != nil {
		return nil, f.err
	}
	return &payment.FetchedInvoice{ID: id, Paid: f.paid, Amount: f.amount}, nil
}

// invoice pending dengan provider_id terisi (korelasi webhook)
func mayarInvoice(providerID string, amount int64) *domain.Invoice {
	pid := providerID
	return &domain.Invoice{
		ID: uuid.New(), OrganizationID: uuid.New(), Type: domain.InvoiceTypeTopup,
		AmountIDR: amount, Status: domain.InvoicePending, ProviderID: &pid,
	}
}

func mayarAPI(inv *domain.Invoice, v mayarVerifier) (*InvoiceAPI, *whRepo) {
	repo := &whRepo{inv: inv}
	uc := usecase.NewInvoice(repo)
	uc.SetTopup(nil, nil) // top-up inject dilewati kalau topups nil; cukup untuk uji MarkPaid transisi
	api := NewInvoiceAPI(uc, usecase.NewAudit(&noopAuditRepo{}))
	api.SetMayarVerifier(v)
	return api, repo
}

func postMayar(api *InvoiceAPI, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest("POST", "/webhooks/mayar", strings.NewReader(body))
	w := httptest.NewRecorder()
	api.mayarWebhook(w, req)
	return w
}

// M3: event valid + re-fetch paid + amount cocok -> MarkPaid
func TestMayarWebhookValid(t *testing.T) {
	inv := mayarInvoice("txn-1", 79000)
	api, repo := mayarAPI(inv, &fakeVerifier{paid: true, amount: 79000})
	w := postMayar(api, `{"event":"payment.received","data":{"transactionId":"txn-1"}}`)
	if w.Code != http.StatusOK {
		t.Fatalf("valid harus 200, dapat %d", w.Code)
	}
	if inv.Status != domain.InvoicePaid {
		t.Fatal("invoice harus paid setelah webhook valid")
	}
	_ = repo
}

// M4 (KEAMANAN): payload palsu -> re-fetch bilang BELUM paid -> TIDAK MarkPaid
func TestMayarWebhookForgedRejected(t *testing.T) {
	inv := mayarInvoice("txn-1", 79000)
	v := &fakeVerifier{paid: false, amount: 79000} // Mayar sebenarnya bilang belum bayar
	api, _ := mayarAPI(inv, v)
	w := postMayar(api, `{"event":"payment.received","data":{"transactionId":"txn-1"}}`)
	if w.Code != http.StatusOK {
		t.Fatalf("payload palsu harus 200 (stop retry), dapat %d", w.Code)
	}
	if v.calls != 1 {
		t.Fatal("harus re-fetch ke API (jangan percaya payload)")
	}
	if inv.Status != domain.InvoicePending {
		t.Fatal("KEAMANAN BOCOR: payload palsu menandai invoice paid tanpa re-fetch lolos")
	}
}

// re-fetch ke API gagal (network) -> 5xx (Mayar retry), TIDAK paid
func TestMayarWebhookRefetchError(t *testing.T) {
	inv := mayarInvoice("txn-1", 79000)
	api, _ := mayarAPI(inv, &fakeVerifier{err: context.DeadlineExceeded})
	w := postMayar(api, `{"event":"payment.received","data":{"transactionId":"txn-1"}}`)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("re-fetch gagal harus 5xx (retry), dapat %d", w.Code)
	}
	if inv.Status != domain.InvoicePending {
		t.Fatal("re-fetch gagal: jangan tandai paid")
	}
}

// M6: amount hasil re-fetch beda dari invoice -> TIDAK paid
func TestMayarWebhookAmountMismatch(t *testing.T) {
	inv := mayarInvoice("txn-1", 79000)
	api, _ := mayarAPI(inv, &fakeVerifier{paid: true, amount: 999999}) // amount beda
	w := postMayar(api, `{"event":"payment.received","data":{"transactionId":"txn-1"}}`)
	if w.Code != http.StatusOK {
		t.Fatalf("amount beda harus 200 (stop retry), dapat %d", w.Code)
	}
	if inv.Status != domain.InvoicePending {
		t.Fatal("amount beda: invoice tak boleh paid")
	}
}

// M5: webhook 2x -> idempoten (MarkPaid transisi sekali)
func TestMayarWebhookIdempotent(t *testing.T) {
	inv := mayarInvoice("txn-1", 79000)
	api, _ := mayarAPI(inv, &fakeVerifier{paid: true, amount: 79000})
	body := `{"event":"payment.received","data":{"transactionId":"txn-1"}}`
	if w := postMayar(api, body); w.Code != http.StatusOK {
		t.Fatalf("pertama 200, dapat %d", w.Code)
	}
	if w := postMayar(api, body); w.Code != http.StatusOK {
		t.Fatalf("kedua tetap 200, dapat %d", w.Code)
	}
	if inv.Status != domain.InvoicePaid {
		t.Fatal("status tetap paid (idempoten)")
	}
}

// event non-payment.received -> 200 skip, tak re-fetch
func TestMayarWebhookIgnoresOtherEvents(t *testing.T) {
	inv := mayarInvoice("txn-1", 79000)
	v := &fakeVerifier{paid: true, amount: 79000}
	api, _ := mayarAPI(inv, v)
	w := postMayar(api, `{"event":"membership.memberExpired","data":{"memberId":"x"}}`)
	if w.Code != http.StatusOK || v.calls != 0 || inv.Status != domain.InvoicePending {
		t.Fatalf("event lain: 200 tanpa re-fetch/paid; code=%d calls=%d status=%s", w.Code, v.calls, inv.Status)
	}
}

// provider_id tak dikenal -> 200, tak crash
func TestMayarWebhookUnknownProvider(t *testing.T) {
	inv := mayarInvoice("txn-KNOWN", 79000)
	api, _ := mayarAPI(inv, &fakeVerifier{paid: true, amount: 79000})
	w := postMayar(api, `{"event":"payment.received","data":{"transactionId":"txn-OTHER"}}`)
	if w.Code != http.StatusOK || inv.Status != domain.InvoicePending {
		t.Fatalf("provider tak dikenal: 200 tanpa sentuh invoice; code=%d status=%s", w.Code, inv.Status)
	}
}

// verifier nil (Mayar nonaktif) -> 503
func TestMayarWebhookDisabled(t *testing.T) {
	repo := &whRepo{inv: mayarInvoice("t", 1000)}
	api := NewInvoiceAPI(usecase.NewInvoice(repo), usecase.NewAudit(&noopAuditRepo{}))
	// SetMayarVerifier tak dipanggil -> nil
	w := postMayar(api, `{"event":"payment.received","data":{"transactionId":"t"}}`)
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("verifier nil harus 503, dapat %d", w.Code)
	}
}
