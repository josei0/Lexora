package handler

import (
	"encoding/json"
	"net/http"
	"net/mail"
	"strings"

	"github.com/lexora/backend/internal/delivery/http/middleware"
	"github.com/lexora/backend/internal/domain"
)

// endpoint publik self-serve (update6 §5.2-5.3). Manual mux (bukan openapi) - hindari regen.
func (a *API) PublicAuthRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /auth/register", a.register)
	mux.HandleFunc("POST /auth/verify-email", a.verifyEmail)
	mux.HandleFunc("POST /auth/google", a.loginGoogle)
}

type registerBody struct {
	FirmaName string `json:"firma_name"`
	FullName  string `json:"full_name"`
	Email     string `json:"email"`
	Password  string `json:"password"`
}

func (a *API) register(w http.ResponseWriter, r *http.Request) {
	var b registerBody
	if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "permintaan tidak valid")
		return
	}
	b.FirmaName, b.FullName = strings.TrimSpace(b.FirmaName), strings.TrimSpace(b.FullName)
	b.Email = strings.TrimSpace(strings.ToLower(b.Email))
	if b.FirmaName == "" || b.FullName == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "nama firma dan nama lengkap wajib diisi")
		return
	}
	if _, err := mail.ParseAddress(b.Email); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "email tidak valid")
		return
	}
	if len(b.Password) < 8 { // samakan aturan existing (change-password): len >= 8
		writeError(w, http.StatusBadRequest, "invalid_request", "password minimal 8 karakter")
		return
	}

	ip := middleware.ClientIPFrom(r.Context())
	u, err := a.org.Register(r.Context(), b.FirmaName, b.FullName, b.Email, b.Password, false, nil)
	if err != nil {
		if err == domain.ErrConflict {
			writeError(w, http.StatusConflict, "conflict", "email sudah terdaftar")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal", "gagal mendaftar")
		return
	}
	uid := u.ID
	a.audit.Record(r.Context(), domain.AuditRegister, nil, &uid, nil, ip)
	// 202: akun dibuat, tunggu verifikasi email (kalau mailer aktif). Tanpa auto-login.
	writeJSON(w, http.StatusAccepted, map[string]any{
		"message":      "Pendaftaran berhasil. Cek email untuk verifikasi.",
		"needs_verify": !u.IsActive,
	})
}

type verifyBody struct {
	Token string `json:"token"`
}

func (a *API) verifyEmail(w http.ResponseWriter, r *http.Request) {
	var b verifyBody
	if err := json.NewDecoder(r.Body).Decode(&b); err != nil || strings.TrimSpace(b.Token) == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "token wajib diisi")
		return
	}
	if err := a.org.VerifyEmail(r.Context(), b.Token); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_token", "tautan verifikasi tidak valid atau sudah kadaluarsa")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"message": "Email terverifikasi. Silakan login."})
}

type googleBody struct {
	IDToken string `json:"id_token"`
}

// POST /auth/google: login / auto-register via Google (u6 §5.3). Set-cookie refresh spt Login.
func (a *API) loginGoogle(w http.ResponseWriter, r *http.Request) {
	var b googleBody
	if err := json.NewDecoder(r.Body).Decode(&b); err != nil || strings.TrimSpace(b.IDToken) == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "id_token wajib diisi")
		return
	}
	ip := middleware.ClientIPFrom(r.Context())
	tok, err := a.auth.LoginGoogle(r.Context(), b.IDToken)
	if err != nil {
		if err == domain.ErrGoogleUnlinked {
			writeError(w, http.StatusConflict, "google_unlinked", "email sudah terdaftar. Gunakan login password.")
			return
		}
		if err == domain.ErrInactive {
			writeError(w, http.StatusForbidden, "inactive", "akun nonaktif")
			return
		}
		a.audit.Record(r.Context(), domain.AuditLoginFail, nil, nil, nil, ip)
		writeError(w, http.StatusUnauthorized, "invalid_token", "login Google gagal")
		return
	}
	uid := tok.UserID
	a.audit.Record(r.Context(), domain.AuditGoogleAuth, nil, &uid, nil, ip)
	w.Header().Set("Set-Cookie", a.cookie(tok.Refresh, a.refreshTTL))
	writeJSON(w, http.StatusOK, map[string]any{
		"access_token": tok.Access,
		"token_type":   "Bearer",
		"expires_in":   tok.ExpiresIn,
	})
}
