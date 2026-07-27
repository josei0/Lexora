package handler

import (
	"encoding/json"
	"net/http"

	"github.com/lexora/backend/internal/delivery/http/middleware"
	"github.com/lexora/backend/internal/domain"
	"github.com/lexora/backend/internal/usecase"
)

// cari sumber web + promosi ke pustaka. org_admin (pengelola pustaka), bukan super_admin.
type WebAPI struct {
	web   *usecase.WebIngest
	audit *usecase.Audit
}

func NewWebAPI(web *usecase.WebIngest, audit *usecase.Audit) *WebAPI {
	return &WebAPI{web: web, audit: audit}
}

func (a *WebAPI) Routes(mux *http.ServeMux) {
	mux.HandleFunc("POST /web-sources/search", a.search)
	mux.HandleFunc("POST /web-sources/preview", a.preview)
	mux.HandleFunc("POST /web-sources/ingest", a.ingest)
	mux.HandleFunc("GET /web-sources/candidates", a.candidates)
}

// org_admin only; org dari JWT (anti-IDOR)
func (a *WebAPI) orgAdmin(w http.ResponseWriter, r *http.Request) (domain.Identity, bool) {
	id, ok := orgIdentity(w, r)
	if !ok {
		return id, false
	}
	if !id.IsOrgAdmin() {
		writeError(w, http.StatusForbidden, "forbidden", "akses ditolak")
		return id, false
	}
	return id, true
}

func (a *WebAPI) search(w http.ResponseWriter, r *http.Request) {
	if _, ok := a.orgAdmin(w, r); !ok {
		return
	}
	var body struct {
		Query string `json:"query"`
		Limit int    `json:"limit"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "body tidak valid")
		return
	}
	res, err := a.web.Search(r.Context(), body.Query, body.Limit)
	if err != nil {
		writeErr(w, err)
		return
	}
	out := make([]map[string]string, len(res))
	for i, x := range res {
		out[i] = map[string]string{"title": x.Title, "url": x.URL, "snippet": x.Content}
	}
	writeJSON(w, http.StatusOK, out)
}

func (a *WebAPI) preview(w http.ResponseWriter, r *http.Request) {
	if _, ok := a.orgAdmin(w, r); !ok {
		return
	}
	var body struct {
		URL string `json:"url"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "body tidak valid")
		return
	}
	title, text, err := a.web.Preview(r.Context(), body.URL)
	if err != nil {
		// pesan guard/fetch informatif: admin perlu tahu kenapa URL ditolak
		writeError(w, http.StatusBadRequest, "fetch_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"title": title, "text": text, "chars": len(text)})
}

func (a *WebAPI) candidates(w http.ResponseWriter, r *http.Request) {
	id, ok := a.orgAdmin(w, r)
	if !ok {
		return
	}
	list, err := a.web.Candidates(r.Context(), id.OrgID)
	if err != nil {
		writeErr(w, err)
		return
	}
	type item struct {
		URL    string `json:"url"`
		Hits   int    `json:"hits"`
		LastAt string `json:"last_at"`
	}
	out := make([]item, len(list))
	for i, c := range list {
		out[i] = item{URL: c.URL, Hits: c.Hits, LastAt: c.LastAt.Format("2006-01-02")}
	}
	writeJSON(w, http.StatusOK, out)
}

func (a *WebAPI) ingest(w http.ResponseWriter, r *http.Request) {
	id, ok := a.orgAdmin(w, r)
	if !ok {
		return
	}
	var body struct {
		URL string `json:"url"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "body tidak valid")
		return
	}
	doc, err := a.web.FromURL(r.Context(), id.OrgID, id.UserID, body.URL)
	if err != nil {
		switch err {
		case domain.ErrConflict:
			writeError(w, http.StatusConflict, "conflict", "sumber ini sudah ada di pustaka")
		case domain.ErrSubExpired:
			writeErr(w, err)
		default:
			writeError(w, http.StatusBadRequest, "ingest_failed", err.Error())
		}
		return
	}
	uid, orgID, docID := id.UserID, id.OrgID, doc.ID
	a.audit.Record(r.Context(), domain.AuditKBWebIngest, &orgID, &uid, &docID, middleware.ClientIPFrom(r.Context()))
	writeJSON(w, http.StatusAccepted, map[string]any{
		"id": doc.ID, "file_name": doc.FileName, "status": doc.Status, "source_url": doc.SourceURL,
	})
}
