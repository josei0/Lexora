package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/lexora/backend/internal/delivery/http/middleware"
	"github.com/lexora/backend/internal/domain"
	"github.com/lexora/backend/internal/usecase"
	"github.com/lexora/backend/pkg/payment"
)

// mayarVerifier: re-fetch status invoice ke API Mayar (webhook Mayar tanpa
// signature -> jangan percaya payload, verifikasi balik). Impl: *payment.Mayar.
type mayarVerifier interface {
	GetInvoice(ctx context.Context, id string) (*payment.FetchedInvoice, error)
}

type InvoiceAPI struct {
	inv   *usecase.Invoice
	audit *usecase.Audit
	mayar mayarVerifier // re-fetch Mayar; nil = webhook Mayar nonaktif
}

func NewInvoiceAPI(inv *usecase.Invoice, audit *usecase.Audit) *InvoiceAPI {
	return &InvoiceAPI{inv: inv, audit: audit}
}

// SetMayarVerifier: aktifkan webhook Mayar (re-fetch). Nil = endpoint tolak.
func (a *InvoiceAPI) SetMayarVerifier(v mayarVerifier) { a.mayar = v }

func (a *InvoiceAPI) Routes(mux *http.ServeMux) {
	mux.HandleFunc("GET /invoices", a.list)                           // org_admin: riwayat org sendiri
	mux.HandleFunc("POST /admin/invoices/{id}/mark-paid", a.markPaid) // super_admin: lunas manual
	mux.HandleFunc("POST /billing/topup", a.topup)                    // org_admin: buat invoice top-up
	mux.HandleFunc("POST /webhooks/mayar", a.mayarWebhook)            // publik: re-fetch verif (tanpa signature)
}

// org_admin lihat tagihan org-nya (org dari JWT, anti-IDOR)
func (a *InvoiceAPI) list(w http.ResponseWriter, r *http.Request) {
	id, ok := orgIdentity(w, r)
	if !ok {
		return
	}
	if !id.IsOrgAdmin() {
		writeError(w, http.StatusForbidden, "forbidden", "akses ditolak")
		return
	}
	list, err := a.inv.ListByOrg(r.Context(), id.OrgID)
	if err != nil {
		writeErr(w, err)
		return
	}
	if list == nil {
		list = []domain.Invoice{}
	}
	writeJSON(w, http.StatusOK, list)
}

// org_admin buat invoice top-up pending. package_code dari body; harga + token dihitung server.
func (a *InvoiceAPI) topup(w http.ResponseWriter, r *http.Request) {
	id, ok := orgIdentity(w, r)
	if !ok {
		return
	}
	if !id.IsOrgAdmin() {
		writeError(w, http.StatusForbidden, "forbidden", "akses ditolak")
		return
	}
	var body struct {
		PackageCode string `json:"package_code"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.PackageCode == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "package_code diperlukan")
		return
	}
	inv, err := a.inv.CreateTopup(r.Context(), id.OrgID, body.PackageCode, id.Email, id.Name, time.Now())
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, inv)
}

// super_admin tandai lunas di luar gateway (transfer langsung / kesepakatan sales).
// Juga fallback operasional kalau webhook macet (fase 12).
func (a *InvoiceAPI) markPaid(w http.ResponseWriter, r *http.Request) {
	id, ok := middleware.IdentityFrom(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "sesi tidak valid")
		return
	}
	if !id.IsSuperAdmin() {
		writeError(w, http.StatusForbidden, "forbidden", "akses ditolak")
		return
	}
	invID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "id tidak valid")
		return
	}
	inv, changed, err := a.inv.MarkPaid(r.Context(), invID, time.Now())
	if err != nil {
		writeErr(w, err)
		return
	}
	if changed { // idempoten: jangan audit ganda kalau sudah paid
		uid, orgID := id.UserID, inv.OrganizationID
		a.audit.Record(r.Context(), domain.AuditManualPaid, &orgID, &uid, &inv.ID, middleware.ClientIPFrom(r.Context()))
	}
	writeJSON(w, http.StatusOK, inv)
}

// POST /webhooks/mayar (publik). Mayar TAK kirim signature (dikonfirmasi dari CLI
// resmi), jadi payload TIDAK dipercaya — verifikasi dgn RE-FETCH ke API Mayar
// pakai API key kita. Keamanan berlapis: (1) event=payment.received, (2) re-fetch
// status=paid ke API, (3) amount cocok, (4) korelasi via provider_id. Semua harus lolos.
func (a *InvoiceAPI) mayarWebhook(w http.ResponseWriter, r *http.Request) {
	if a.mayar == nil { // verifier tak diset = webhook Mayar nonaktif
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}
	var ev struct {
		Event string `json:"event"`
		Data  struct {
			ID            string `json:"id"`
			TransactionID string `json:"transactionId"`
		} `json:"data"`
	}
	_ = json.NewDecoder(r.Body).Decode(&ev)

	// hanya event pembayaran-diterima yang relevan; sisanya 200 (stop retry).
	if ev.Event != "payment.received" {
		w.WriteHeader(http.StatusOK)
		return
	}
	// id transaksi dari payload = KUNCI re-fetch. transactionId dulu (yang kita
	// simpan ke provider_id saat create), fallback id.
	txnID := ev.Data.TransactionID
	if txnID == "" {
		txnID = ev.Data.ID
	}
	if txnID == "" {
		w.WriteHeader(http.StatusOK) // payload tak berguna; jangan retry
		return
	}

	// VERIFIKASI: re-fetch ke API Mayar. Payload webhook TAK dipercaya.
	fetched, err := a.mayar.GetInvoice(r.Context(), txnID)
	if err != nil {
		// gagal cek ke Mayar (network/API) = jangan tandai paid. 5xx -> Mayar retry.
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if !fetched.Paid {
		// re-fetch bilang belum lunas -> abaikan (payload mungkin palsu / belum settle). 200 stop retry.
		w.WriteHeader(http.StatusOK)
		return
	}

	// korelasi ke invoice kita via provider_id (= transactionId saat create).
	inv, err := a.inv.ByProviderID(r.Context(), txnID)
	if err != nil {
		w.WriteHeader(http.StatusOK) // invoice tak ada / bukan punya kita; jangan retry
		return
	}
	// anti-tamper: amount hasil re-fetch harus == amount invoice kita.
	// (fetched.Amount 0 = Mayar tak balik amount -> lewati cek ini, andalkan provider_id+paid)
	if fetched.Amount != 0 && fetched.Amount != inv.AmountIDR {
		a.audit.Record(r.Context(), domain.AuditWebhookPaid, &inv.OrganizationID, nil, &inv.ID, "amount_mismatch")
		w.WriteHeader(http.StatusOK)
		return
	}

	paid, changed, err := a.inv.MarkPaid(r.Context(), inv.ID, time.Now())
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError) // DB fail -> Mayar retry
		return
	}
	if changed {
		a.audit.Record(r.Context(), domain.AuditWebhookPaid, &paid.OrganizationID, nil, &paid.ID, "mayar")
	}
	w.WriteHeader(http.StatusOK)
}
