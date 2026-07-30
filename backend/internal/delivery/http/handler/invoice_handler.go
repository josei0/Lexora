package handler

import (
	"crypto/subtle"
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/lexora/backend/internal/delivery/http/middleware"
	"github.com/lexora/backend/internal/domain"
	"github.com/lexora/backend/internal/usecase"
)

type InvoiceAPI struct {
	inv           *usecase.Invoice
	audit         *usecase.Audit
	callbackToken string // verifikasi webhook Xendit; kosong = webhook nonaktif
}

func NewInvoiceAPI(inv *usecase.Invoice, audit *usecase.Audit) *InvoiceAPI {
	return &InvoiceAPI{inv: inv, audit: audit}
}

// SetCallbackToken: aktifkan webhook Xendit. Kosong = endpoint tolak semua.
func (a *InvoiceAPI) SetCallbackToken(t string) { a.callbackToken = t }

func (a *InvoiceAPI) Routes(mux *http.ServeMux) {
	mux.HandleFunc("GET /invoices", a.list)                           // org_admin: riwayat org sendiri
	mux.HandleFunc("POST /admin/invoices/{id}/mark-paid", a.markPaid) // super_admin: lunas manual
	mux.HandleFunc("POST /billing/topup", a.topup)                    // org_admin: buat invoice top-up
	mux.HandleFunc("POST /webhooks/xendit", a.xenditWebhook)          // publik: verif x-callback-token
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
	inv, err := a.inv.CreateTopup(r.Context(), id.OrgID, body.PackageCode, time.Now())
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

// POST /webhooks/xendit (publik, server-to-server). Keamanan = x-callback-token
// cocok konstanta-waktu. Bukan JWT, bukan IP allowlist. Rate-limit global tak
// menyentuhnya (suffix tak cocok). Tanpa CORS.
func (a *InvoiceAPI) xenditWebhook(w http.ResponseWriter, r *http.Request) {
	// token kosong (belum dikonfigurasi) -> tolak semua; jangan proses tanpa verif
	if a.callbackToken == "" || subtle.ConstantTimeCompare(
		[]byte(r.Header.Get("x-callback-token")), []byte(a.callbackToken)) != 1 {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	var ev struct {
		ExternalID string `json:"external_id"`
		Status     string `json:"status"`
		Amount     int64  `json:"amount"`
	}
	_ = json.NewDecoder(r.Body).Decode(&ev)

	// non-paid (expired/void/pending): 200 supaya Xendit berhenti retry
	if ev.Status != "PAID" && ev.Status != "SETTLED" {
		w.WriteHeader(http.StatusOK)
		return
	}
	id, err := uuid.Parse(ev.ExternalID)
	if err != nil {
		w.WriteHeader(http.StatusOK) // external_id ngaco: jangan retry-loop
		return
	}

	// cek amount cocok SEBELUM mark-paid (anti tamper / salah nominal).
	// amount 0/absen di payload -> lewati cek (Xendit tak selalu kirim di semua event).
	inv, err := a.inv.ByID(r.Context(), id)
	if err != nil {
		w.WriteHeader(http.StatusOK) // invoice tak ada: jangan retry
		return
	}
	if ev.Amount != 0 && ev.Amount != inv.AmountIDR {
		// nominal beda = mencurigakan. Log keras, JANGAN mark-paid. 200 supaya tak retry.
		a.audit.Record(r.Context(), domain.AuditWebhookPaid, &inv.OrganizationID, nil, &inv.ID, "amount_mismatch")
		w.WriteHeader(http.StatusOK)
		return
	}

	paid, changed, err := a.inv.MarkPaid(r.Context(), id, time.Now())
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError) // DB fail: biar Xendit retry
		return
	}
	if changed { // idempoten: webhook ganda tak audit/extend dobel
		a.audit.Record(r.Context(), domain.AuditWebhookPaid, &paid.OrganizationID, nil, &paid.ID, "xendit")
	}
	w.WriteHeader(http.StatusOK)
}
