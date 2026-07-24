package handler

import (
	"encoding/json"
	"net/http"

	"github.com/lexora/backend/internal/delivery/http/middleware"
	"github.com/lexora/backend/internal/domain"
	"github.com/lexora/backend/internal/usecase"
)

// auth panel admin
func (a *API) AdminAuthRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /auth/admin/login", a.adminLogin)
	mux.HandleFunc("POST /auth/admin/refresh", a.adminRefresh)
	mux.HandleFunc("POST /auth/admin/logout", a.adminLogout)
}

func tokenBody(t *usecase.Tokens) map[string]any {
	return map[string]any{
		"access_token": t.Access,
		"token_type":   "Bearer",
		"expires_in":   t.ExpiresIn,
	}
}

func (a *API) adminLogin(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "email atau password salah")
		return
	}
	ip := middleware.ClientIPFrom(r.Context())
	tok, err := a.auth.AdminLogin(r.Context(), body.Email, body.Password)
	if err != nil {
		// throttle di middleware
		a.audit.Record(r.Context(), domain.AuditAdminLoginFail, nil, nil, nil, ip)
		writeError(w, http.StatusUnauthorized, "invalid_credentials", "email atau password salah")
		return
	}
	uid := tok.UserID
	a.audit.Record(r.Context(), domain.AuditAdminLoginOK, nil, &uid, nil, ip)
	w.Header().Set("Set-Cookie", a.adminCookie(tok.Refresh, a.refreshTTL))
	writeJSON(w, http.StatusOK, tokenBody(tok))
}

func (a *API) adminRefresh(w http.ResponseWriter, r *http.Request) {
	// cek origin server-side
	if !a.adminOrigins[r.Header.Get("Origin")] {
		writeError(w, http.StatusForbidden, "forbidden", "origin tidak diizinkan")
		return
	}
	raw := ""
	if c, err := r.Cookie(middleware.AdminRefreshCookieName(a.secure)); err == nil {
		raw = c.Value
	}
	tok, err := a.auth.AdminRefresh(r.Context(), raw)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "unauthorized", "sesi berakhir")
		return
	}
	w.Header().Set("Set-Cookie", a.adminCookie(tok.Refresh, a.refreshTTL))
	writeJSON(w, http.StatusOK, tokenBody(tok))
}

func (a *API) adminLogout(w http.ResponseWriter, r *http.Request) {
	ip := middleware.ClientIPFrom(r.Context())
	if id, ok := middleware.IdentityFrom(r.Context()); ok {
		uid := id.UserID
		a.audit.Record(r.Context(), domain.AuditLogout, nil, &uid, nil, ip)
	}
	if c, err := r.Cookie(middleware.AdminRefreshCookieName(a.secure)); err == nil {
		_ = a.auth.Logout(r.Context(), c.Value)
	}
	w.Header().Set("Set-Cookie", a.clearAdminCookie())
	w.WriteHeader(http.StatusNoContent)
}
