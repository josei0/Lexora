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
	Messages(ctx context.Context, chatID uuid.UUID) ([]Message, error)
	AddCitations(ctx context.Context, cs []Citation) error
	AddUsage(ctx context.Context, u TokenUsage) error
}

type ChatMessage struct {
	Role    string
	Content string
}

type LLMUsage struct {
	InputTokens  int
	OutputTokens int
}

type LLM interface {
	Stream(ctx context.Context, system string, msgs []ChatMessage, onToken func(string)) (LLMUsage, error)
	Model() string
}
