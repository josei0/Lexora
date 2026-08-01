package handler

import (
	"encoding/json"
	"net/http"

	"github.com/lexora/backend/internal/delivery/http/middleware"
	"github.com/lexora/backend/internal/domain"
	"github.com/lexora/backend/internal/usecase"
)

// allowlist domain web-search (update9-B). super_admin only, guard inline
// (tak ada middleware RequireSuperAdmin - pola per-handler, sama billing).
type WebDomainAPI struct {
	uc *usecase.WebDomain
}

func NewWebDomainAPI(uc *usecase.WebDomain) *WebDomainAPI {
	return &WebDomainAPI{uc: uc}
}

// dipanggil dari router (dijahit S4). B tidak sentuh router.go.
func (a *WebDomainAPI) Routes(mux *http.ServeMux) {
	mux.HandleFunc("GET /admin/web-domains", a.list)
	mux.HandleFunc("POST /admin/web-domains", a.add)
	mux.HandleFunc("DELETE /admin/web-domains/{host}", a.remove)
}

// super_admin dari JWT, bukan path (anti-IDOR)
func (a *WebDomainAPI) superAdmin(w http.ResponseWriter, r *http.Request) bool {
	id, ok := middleware.IdentityFrom(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "sesi tidak valid")
		return false
	}
	if !id.IsSuperAdmin() {
		writeError(w, http.StatusForbidden, "forbidden", "akses ditolak")
		return false
	}
	return true
}

func (a *WebDomainAPI) list(w http.ResponseWriter, r *http.Request) {
	if !a.superAdmin(w, r) {
		return
	}
	list, err := a.uc.List(r.Context())
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, domainsJSON(list))
}

func (a *WebDomainAPI) add(w http.ResponseWriter, r *http.Request) {
	if !a.superAdmin(w, r) {
		return
	}
	var body struct {
		Host string `json:"host"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "body tidak valid")
		return
	}
	d, err := a.uc.Add(r.Context(), body.Host)
	if err != nil {
		switch err {
		case domain.ErrConflict:
			writeError(w, http.StatusConflict, "conflict", "domain sudah ada di allowlist")
		case domain.ErrInvalidUpload:
			writeError(w, http.StatusBadRequest, "invalid_request", "host tidak boleh kosong")
		default:
			writeErr(w, err)
		}
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"id": d.ID, "host": d.Host})
}

func (a *WebDomainAPI) remove(w http.ResponseWriter, r *http.Request) {
	if !a.superAdmin(w, r) {
		return
	}
	if err := a.uc.Remove(r.Context(), r.PathValue("host")); err != nil {
		if err == domain.ErrNotFound {
			writeError(w, http.StatusNotFound, "not_found", "domain tidak ditemukan")
			return
		}
		writeErr(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func domainsJSON(list []domain.WebSearchDomain) []map[string]any {
	out := make([]map[string]any, len(list))
	for i, d := range list {
		out[i] = map[string]any{"id": d.ID, "host": d.Host}
	}
	return out
}
