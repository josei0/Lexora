package handler

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/lexora/backend/internal/delivery/http/middleware"
	"github.com/lexora/backend/internal/domain"
	"github.com/lexora/backend/internal/usecase"
)

type InvoiceAPI struct {
	inv   *usecase.Invoice
	audit *usecase.Audit
}

func NewInvoiceAPI(inv *usecase.Invoice, audit *usecase.Audit) *InvoiceAPI {
	return &InvoiceAPI{inv: inv, audit: audit}
}

func (a *InvoiceAPI) Routes(mux *http.ServeMux) {
	mux.HandleFunc("GET /invoices", a.list)                           // org_admin: riwayat org sendiri
	mux.HandleFunc("POST /admin/invoices/{id}/mark-paid", a.markPaid) // super_admin: lunas manual
	mux.HandleFunc("POST /billing/topup", a.topup)                    // org_admin: buat invoice top-up
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
