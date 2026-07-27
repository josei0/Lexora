package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

const (
	RoleUser      = "user"
	RoleAssistant = "assistant"
)

type Chat struct {
	ID             uuid.UUID
	OrganizationID uuid.UUID
	UserID         uuid.UUID
	Title          string
	IsPinned       bool
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type Message struct {
	ID        uuid.UUID
	ChatID    uuid.UUID
	Role      string
	Content   string
	Model     *string
	CreatedAt time.Time
	Citations []Citation
}

type Citation struct {
	ID              uuid.UUID
	MessageID       uuid.UUID
	DocumentChunkID *uuid.UUID
	DocumentID      *uuid.UUID
	ReferenceLabel  string
	Marker          int
	PageNo          *int
	Score           float32
	SourceURL       *string // terisi = sumber web, bukan dokumen pustaka
}

// satu baris log pencarian web: kuota harian + sumber data kandidat pustaka
type WebSearch struct {
	OrganizationID uuid.UUID
	UserID         uuid.UUID
	Query          string
	Provider       string
	ResultsCount   int
	TopURLs        []string
}

// URL populer dari pencarian user yang belum ada di pustaka - kandidat fase 9
type WebCandidate struct {
	URL    string
	Hits   int
	LastAt time.Time
}

type WebSearchRepository interface {
	Log(ctx context.Context, s WebSearch) error
	// jumlah pencarian user hari ini (window WIB) - gate kuota harian
	CountToday(ctx context.Context, orgID, userID uuid.UUID, from time.Time) (int, error)
	DeleteOlderThan(ctx context.Context, t time.Time) (int64, error)
	// URL >= minHits dalam window since, belum ada di documents.source_url org ini
	Candidates(ctx context.Context, orgID uuid.UUID, minHits int, since time.Time) ([]WebCandidate, error)
}

type TokenUsage struct {
	OrganizationID uuid.UUID
	UserID         uuid.UUID
	MessageID      uuid.UUID
	Model          string
	InputTokens    int
	OutputTokens   int
}

type ChatRepository interface {
	Create(ctx context.Context, c *Chat) error
	// scoped org + user - chat privat
	ByID(ctx context.Context, id, orgID, userID uuid.UUID) (*Chat, error)
	ListByUser(ctx context.Context, orgID, userID uuid.UUID, search string, limit, offset int) ([]Chat, error)
	Rename(ctx context.Context, id, orgID, userID uuid.UUID, title string) error
	SetPinned(ctx context.Context, id, orgID, userID uuid.UUID, pinned bool) error
	SoftDelete(ctx context.Context, id, orgID, userID uuid.UUID) error
	Touch(ctx context.Context, id uuid.UUID) error

	AddMessage(ctx context.Context, m *Message) error
	// jumlah pertanyaan user hari ini (window WIB) - cap harian plan gratis
	CountUserMessagesSince(ctx context.Context, orgID, userID uuid.UUID, since time.Time) (int, error)
	Messages(ctx context.Context, chatID uuid.UUID) ([]Message, error)
	AddCitations(ctx context.Context, cs []Citation) error
	AddUsage(ctx context.Context, u TokenUsage) error
}

type ChatMessage struct {
	Role    string
	Content string
	Images  []string // data URL (data:<mime>;base64,...) untuk vision; kosong = teks biasa
}

// lampiran chat sekali-pakai (gambar untuk vision, dokumen untuk konteks). Tidak dipersist.
type Attachment struct {
	Name string
	Mime string
	Data []byte
}

type LLMUsage struct {
	InputTokens  int
	OutputTokens int
}

type LLM interface {
	Stream(ctx context.Context, system string, msgs []ChatMessage, onToken func(string)) (LLMUsage, error)
	Model() string
}
