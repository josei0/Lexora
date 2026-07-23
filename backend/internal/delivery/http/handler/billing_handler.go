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

type BillingAPI struct {
	subs      *usecase.Subscription
	dashboard *usecase.Dashboard
	prompt    *usecase.Prompt
	export    *usecase.Export
	billing   *usecase.Billing
}

func NewBillingAPI(
	subs *usecase.Subscription,
	dash *usecase.Dashboard,
	prompt *usecase.Prompt,
	export *usecase.Export,
	billing *usecase.Billing,
) *BillingAPI {
	return &BillingAPI{subs: subs, dashboard: dash, prompt: prompt, export: export, billing: billing}
}

func (a *BillingAPI) Routes(mux *http.ServeMux) {
	// dashboard
	mux.HandleFunc("GET /dashboard", a.dashStats)

	// quota (token meter di FE)
	mux.HandleFunc("GET /quota", a.quota)

	// subscription (super admin)
	mux.HandleFunc("GET /plans", a.listPlans)
	mux.HandleFunc("GET /subscription", a.getSub)
	mux.HandleFunc("PUT /subscription", a.assignSub)

	// prompts (super admin)
	mux.HandleFunc("GET /prompts/{key}", a.getPrompt)
	mux.HandleFunc("PUT /prompts/{key}", a.setPrompt)

	// export
	mux.HandleFunc("GET /chats/{chatID}/export", a.exportChat)
}

// GET /dashboard
func (a *BillingAPI) dashStats(w http.ResponseWriter, r *http.Request) {
	id, ok := orgIdentity(w, r)
	if !ok {
		return
	}
	stats, err := a.dashboard.Stats(r.Context(), id.OrgID, time.Now())
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, stats)
}

// GET /quota
func (a *BillingAPI) quota(w http.ResponseWriter, r *http.Request) {
	id, ok := orgIdentity(w, r)
	if !ok {
		return
	}
	q, err := a.billing.Usage(r.Context(), id.OrgID, time.Now())
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"limit":     q.Limit,
		"used":      q.Used,
		"remaining": q.Remaining(),
		"soft":      q.Soft,
		"hard":      q.Hard,
	})
}

// GET /plans
func (a *BillingAPI) listPlans(w http.ResponseWriter, r *http.Request) {
	id, ok := middleware.IdentityFrom(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "sesi tidak valid")
		return
	}
	if !id.IsSuperAdmin() {
		writeError(w, http.StatusForbidden, "forbidden", "akses ditolak")
		return
	}
	plans, err := a.subs.Plans(r.Context())
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, plans)
}

// GET /subscription  (org admin: lihat subscription org sendiri)
func (a *BillingAPI) getSub(w http.ResponseWriter, r *http.Request) {
	id, ok := orgIdentity(w, r)
	if !ok {
		return
	}
	sub, err := a.subs.ByOrg(r.Context(), id.OrgID)
	if err == domain.ErrNotFound {
		writeJSON(w, http.StatusOK, nil)
		return
	}
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, sub)
}

// PUT /subscription  (super admin: assign plan ke org)
func (a *BillingAPI) assignSub(w http.ResponseWriter, r *http.Request) {
	id, ok := middleware.IdentityFrom(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "sesi tidak valid")
		return
	}
	if !id.IsSuperAdmin() {
		writeError(w, http.StatusForbidden, "forbidden", "akses ditolak")
		return
	}
	var body struct {
		OrgID    string `json:"org_id"`
		PlanCode string `json:"plan_code"`
		Seats    int    `json:"seats"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "body tidak valid")
		return
	}
	orgID, err := uuid.Parse(body.OrgID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "org_id tidak valid")
		return
	}
	sub, err := a.subs.Assign(r.Context(), orgID, body.PlanCode, body.Seats)
	if err != nil {
		switch err {
		case domain.ErrNotFound:
			writeError(w, http.StatusNotFound, "not_found", "plan tidak ditemukan")
		case domain.ErrSeatsFull:
			writeError(w, http.StatusConflict, "seats_full", "anggota melebihi jumlah seat")
		case domain.ErrInvalidUpload:
			writeError(w, http.StatusBadRequest, "invalid_request", "seats minimal 1")
		default:
			writeErr(w, err)
		}
		return
	}
	writeJSON(w, http.StatusOK, sub)
}

// GET /prompts/{key}  (super admin)
func (a *BillingAPI) getPrompt(w http.ResponseWriter, r *http.Request) {
	id, ok := middleware.IdentityFrom(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "sesi tidak valid")
		return
	}
	if !id.IsSuperAdmin() {
		writeError(w, http.StatusForbidden, "forbidden", "akses ditolak")
		return
	}
	p, err := a.prompt.Get(r.Context(), r.PathValue("key"))
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, p)
}

// PUT /prompts/{key}  (super admin)
func (a *BillingAPI) setPrompt(w http.ResponseWriter, r *http.Request) {
	id, ok := middleware.IdentityFrom(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "sesi tidak valid")
		return
	}
	if !id.IsSuperAdmin() {
		writeError(w, http.StatusForbidden, "forbidden", "akses ditolak")
		return
	}
	var body struct {
		Content string `json:"content"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "body tidak valid")
		return
	}
	if err := a.prompt.Set(r.Context(), r.PathValue("key"), body.Content, id.UserID); err != nil {
		if err == domain.ErrInvalidUpload {
			writeError(w, http.StatusBadRequest, "invalid_request", "konten tidak boleh kosong")
			return
		}
		writeErr(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// GET /chats/{chatID}/export?format=word|pdf
func (a *BillingAPI) exportChat(w http.ResponseWriter, r *http.Request) {
	id, ok := orgIdentity(w, r)
	if !ok {
		return
	}
	chatID, ok := pathUUID(w, r)
	if !ok {
		return
	}
	format := r.URL.Query().Get("format")

	title, htmlDoc, err := a.export.ChatHTML(r.Context(), chatID, id.OrgID, id.UserID)
	if err != nil {
		writeErr(w, err)
		return
	}

	safe := sanitizeFilename(title)
	if format == "word" {
		w.Header().Set("Content-Type", "application/msword")
		w.Header().Set("Content-Disposition", `attachment; filename="`+safe+`.doc"`)
	} else {
		// default: print-optimized HTML (user prints to PDF)
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Content-Disposition", `attachment; filename="`+safe+`.html"`)
	}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(htmlDoc))
}

func sanitizeFilename(s string) string {
	out := make([]byte, 0, len(s))
	for i := 0; i < len(s) && i < 60; i++ {
		c := s[i]
		if c == '/' || c == '\\' || c == ':' || c == '*' || c == '?' || c == '"' || c == '<' || c == '>' || c == '|' {
			out = append(out, '_')
		} else {
			out = append(out, c)
		}
	}
	if len(out) == 0 {
		return "percakapan"
	}
	return string(out)
}
