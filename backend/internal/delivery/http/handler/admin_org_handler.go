package handler

import (
	"encoding/json"
	"net/http"
	"net/mail"
	"strings"

	"github.com/google/uuid"
	"github.com/lexora/backend/internal/delivery/http/middleware"
	"github.com/lexora/backend/internal/domain"
)

// route super_admin manual (bukan openapi). AddMember usecase sudah generik -
// expose ke org mana saja lewat orgId di path (u6 §5.4). Nol usecase baru.
func (a *API) AdminOrgRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /admin/organizations/{orgId}/members", a.adminAssignMember)
}

type assignMemberBody struct {
	Email    string `json:"email"`
	FullName string `json:"full_name"`
	Role     string `json:"role"`
}

// POST /admin/organizations/{orgId}/members - assign 1 akun ke org existing (super_admin).
func (a *API) adminAssignMember(w http.ResponseWriter, r *http.Request) {
	id, ok := middleware.IdentityFrom(r.Context())
	if !ok || !id.IsSuperAdmin() {
		writeError(w, http.StatusForbidden, "forbidden", "akses ditolak")
		return
	}
	orgID, err := uuid.Parse(r.PathValue("orgId"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "org tidak valid")
		return
	}
	var b assignMemberBody
	if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "permintaan tidak valid")
		return
	}
	b.Email, b.FullName = strings.TrimSpace(strings.ToLower(b.Email)), strings.TrimSpace(b.FullName)
	if _, err := mail.ParseAddress(b.Email); err != nil || b.FullName == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "email dan nama wajib diisi")
		return
	}
	if b.Role == "" {
		b.Role = domain.OrgRoleMember
	}

	nm, err := a.org.AddMember(r.Context(), orgID, b.Email, b.FullName, b.Role)
	if err != nil {
		switch err {
		case domain.ErrConflict:
			writeError(w, http.StatusConflict, "conflict", "email sudah terdaftar")
		case domain.ErrSeatsFull:
			writeError(w, http.StatusConflict, "seats_full", "jumlah seat langganan sudah penuh")
		case domain.ErrForbidden:
			writeError(w, http.StatusBadRequest, "invalid_role", "role tidak valid")
		default:
			writeError(w, http.StatusInternalServerError, "internal", "gagal menambah anggota")
		}
		return
	}
	adminUID := id.UserID
	newUID := nm.UserID
	a.audit.Record(r.Context(), domain.AuditMemberAdd, &orgID, &adminUID, &newUID, middleware.ClientIPFrom(r.Context()))
	writeJSON(w, http.StatusCreated, map[string]any{
		"user_id":       nm.UserID,
		"email":         nm.Email,
		"temp_password": nm.TempPassword,
	})
}
