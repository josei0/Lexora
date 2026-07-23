package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"

	"github.com/lexora/backend/internal/delivery/http/middleware"
	"github.com/lexora/backend/internal/domain"
	"github.com/lexora/backend/internal/usecase"
)

const timeLayout = time.RFC3339

type ChatAPI struct{ rag *usecase.RAG }

func NewChatAPI(rag *usecase.RAG) *ChatAPI { return &ChatAPI{rag: rag} }

func (a *ChatAPI) Routes(mux *http.ServeMux) {
	mux.HandleFunc("POST /chats", a.create)
	mux.HandleFunc("GET /chats", a.list)
	mux.HandleFunc("PATCH /chats/{chatID}", a.update)
	mux.HandleFunc("DELETE /chats/{chatID}", a.remove)
	mux.HandleFunc("GET /chats/{chatID}/messages", a.messages)
	mux.HandleFunc("POST /chats/{chatID}/messages", a.ask)
}

type chatJSON struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	IsPinned  bool   `json:"is_pinned"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

type citationJSON struct {
	Marker     int     `json:"marker"`
	Label      string  `json:"label"`
	DocumentID string  `json:"document_id,omitempty"`
	PageNo     *int    `json:"page_no,omitempty"`
	Score      float32 `json:"score"`
}

type messageJSON struct {
	ID        string         `json:"id"`
	Role      string         `json:"role"`
	Content   string         `json:"content"`
	Model     *string        `json:"model,omitempty"`
	CreatedAt string         `json:"created_at"`
	Citations []citationJSON `json:"citations"`
}

func (a *ChatAPI) create(w http.ResponseWriter, r *http.Request) {
	id, ok := orgIdentity(w, r)
	if !ok {
		return
	}
	var body struct {
		Title string `json:"title"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)

	chat, err := a.rag.CreateChat(r.Context(), id.OrgID, id.UserID, body.Title)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, toChatJSON(chat))
}

func (a *ChatAPI) list(w http.ResponseWriter, r *http.Request) {
	id, ok := orgIdentity(w, r)
	if !ok {
		return
	}
	limit := intParam(r, "limit", 50)
	offset := intParam(r, "offset", 0)

	chats, err := a.rag.ListChats(r.Context(), id.OrgID, id.UserID, r.URL.Query().Get("search"), limit, offset)
	if err != nil {
		writeErr(w, err)
		return
	}
	out := make([]chatJSON, len(chats))
	for i := range chats {
		out[i] = toChatJSON(&chats[i])
	}
	writeJSON(w, http.StatusOK, out)
}

func (a *ChatAPI) update(w http.ResponseWriter, r *http.Request) {
	id, ok := orgIdentity(w, r)
	if !ok {
		return
	}
	chatID, ok := pathUUID(w, r)
	if !ok {
		return
	}
	var body struct {
		Title    *string `json:"title"`
		IsPinned *bool   `json:"is_pinned"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "body tidak valid")
		return
	}
	if body.Title != nil {
		if err := a.rag.Rename(r.Context(), chatID, id.OrgID, id.UserID, *body.Title); err != nil {
			writeErr(w, err)
			return
		}
	}
	if body.IsPinned != nil {
		if err := a.rag.SetPinned(r.Context(), chatID, id.OrgID, id.UserID, *body.IsPinned); err != nil {
			writeErr(w, err)
			return
		}
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *ChatAPI) remove(w http.ResponseWriter, r *http.Request) {
	id, ok := orgIdentity(w, r)
	if !ok {
		return
	}
	chatID, ok := pathUUID(w, r)
	if !ok {
		return
	}
	if err := a.rag.Delete(r.Context(), chatID, id.OrgID, id.UserID); err != nil {
		writeErr(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *ChatAPI) messages(w http.ResponseWriter, r *http.Request) {
	id, ok := orgIdentity(w, r)
	if !ok {
		return
	}
	chatID, ok := pathUUID(w, r)
	if !ok {
		return
	}
	msgs, err := a.rag.Messages(r.Context(), chatID, id.OrgID, id.UserID)
	if err != nil {
		writeErr(w, err)
		return
	}
	out := make([]messageJSON, len(msgs))
	for i := range msgs {
		out[i] = toMessageJSON(&msgs[i])
	}
	writeJSON(w, http.StatusOK, out)
}

func (a *ChatAPI) ask(w http.ResponseWriter, r *http.Request) {
	id, ok := orgIdentity(w, r)
	if !ok {
		return
	}
	chatID, ok := pathUUID(w, r)
	if !ok {
		return
	}
	var body struct {
		Content string `json:"content"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "body tidak valid")
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, "internal", "streaming tidak didukung")
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)

	send := func(v any) {
		b, err := json.Marshal(v)
		if err != nil {
			return
		}
		w.Write([]byte("data: "))
		w.Write(b)
		w.Write([]byte("\n\n"))
		flusher.Flush()
	}

	msg, err := a.rag.Ask(r.Context(), chatID, id.OrgID, id.UserID, body.Content, func(tok string) {
		send(map[string]string{"token": tok})
	})
	if err != nil {
		send(map[string]string{"error": sseError(err)})
		return
	}
	send(map[string]any{"done": true, "message_id": msg.ID, "citations": toCitationsJSON(msg.Citations)})
}

func sseError(err error) string {
	switch {
	case errors.Is(err, domain.ErrNotFound):
		return "percakapan tidak ditemukan"
	case errors.Is(err, domain.ErrInvalidUpload):
		return "pertanyaan kosong"
	default:
		return "gagal memproses pertanyaan"
	}
}

func orgIdentity(w http.ResponseWriter, r *http.Request) (domain.Identity, bool) {
	id, ok := middleware.IdentityFrom(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "sesi tidak valid")
		return id, false
	}
	if id.OrgID == uuid.Nil {
		writeError(w, http.StatusForbidden, "forbidden", "akses ditolak")
		return id, false
	}
	return id, true
}

func pathUUID(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	id, err := uuid.Parse(r.PathValue("chatID"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "id tidak valid")
		return uuid.Nil, false
	}
	return id, true
}

func intParam(r *http.Request, key string, def int) int {
	if v := r.URL.Query().Get(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			return n
		}
	}
	return def
}

func toChatJSON(c *domain.Chat) chatJSON {
	return chatJSON{
		ID:        c.ID.String(),
		Title:     c.Title,
		IsPinned:  c.IsPinned,
		CreatedAt: c.CreatedAt.Format(timeLayout),
		UpdatedAt: c.UpdatedAt.Format(timeLayout),
	}
}

func toMessageJSON(m *domain.Message) messageJSON {
	return messageJSON{
		ID:        m.ID.String(),
		Role:      m.Role,
		Content:   m.Content,
		Model:     m.Model,
		CreatedAt: m.CreatedAt.Format(timeLayout),
		Citations: toCitationsJSON(m.Citations),
	}
}

func toCitationsJSON(cits []domain.Citation) []citationJSON {
	out := make([]citationJSON, len(cits))
	for i, c := range cits {
		out[i] = citationJSON{Marker: c.Marker, Label: c.ReferenceLabel, PageNo: c.PageNo, Score: c.Score}
		if c.DocumentID != nil {
			out[i].DocumentID = c.DocumentID.String()
		}
	}
	return out
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, code, msg string) {
	writeJSON(w, status, map[string]any{"error": map[string]string{"code": code, "message": msg}})
}

func writeErr(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrNotFound):
		writeError(w, http.StatusNotFound, "not_found", "tidak ditemukan")
	case errors.Is(err, domain.ErrInvalidUpload):
		writeError(w, http.StatusBadRequest, "invalid_request", "permintaan tidak valid")
	default:
		writeError(w, http.StatusInternalServerError, "internal", "terjadi kesalahan")
	}
}
