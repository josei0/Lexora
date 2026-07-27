package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

const (
	DocStatusUploaded   = "uploaded"
	DocStatusProcessing = "processing"
	DocStatusIndexed    = "indexed"
	DocStatusFailed     = "failed"

	DocScopeKnowledgeBase = "knowledge_base"
	DocScopeUser          = "user"
)

type Document struct {
	ID             uuid.UUID
	OrganizationID uuid.UUID
	UploadedBy     uuid.UUID
	Scope          string
	FileName       string
	MimeType       string
	FileSize       int64
	StoragePath    string
	Status         string
	Error          *string
	SourceURL      *string // asal ingest web; nil = upload biasa
	CreatedAt      time.Time
}

type DocumentChunk struct {
	ID            uuid.UUID
	DocumentID    uuid.UUID
	ChunkIndex    int
	PageNo        *int
	QdrantPointID string
}

type Chunk struct {
	Index int
	Text  string
	Page  *int
}

type VectorPoint struct {
	ID      string
	Vector  []float32
	Payload map[string]any
}

type SearchHit struct {
	ID      string
	Score   float32
	Payload map[string]any
}

type DocumentRepository interface {
	Create(ctx context.Context, d *Document) error
	// scoped by org - anti-IDOR
	ListByOrg(ctx context.Context, orgID uuid.UUID, limit, offset int) ([]Document, error)
	ByID(ctx context.Context, id, orgID uuid.UUID) (*Document, error)
	SetStatus(ctx context.Context, id uuid.UUID, status string, errMsg *string) error
	// startup recovery: docs stuck uploaded/processing
	PendingIDs(ctx context.Context) ([]uuid.UUID, error)
	AnyByID(ctx context.Context, id uuid.UUID) (*Document, error)
	InsertChunks(ctx context.Context, chunks []DocumentChunk) error
	DeleteChunks(ctx context.Context, documentID uuid.UUID) error
	// anti-duplikat ingest web; ErrNotFound kalau belum pernah
	BySourceURL(ctx context.Context, orgID uuid.UUID, url string) (*Document, error)
}

type Extractor interface {
	// returns per-page text; page index is 1-based
	Extract(path, mimeType string) ([]string, error)
}

type Embedder interface {
	Embed(ctx context.Context, texts []string) ([][]float32, error)
	Dim() int
}

type VectorRepository interface {
	EnsureCollection(ctx context.Context, name string, dim int) error
	Upsert(ctx context.Context, collection string, points []VectorPoint) error
	Search(ctx context.Context, collection string, vector []float32, topK int, filter map[string]any) ([]SearchHit, error)
	DeleteByDocument(ctx context.Context, collection, documentID string) error
}

type Storage interface {
	Save(orgID, docID uuid.UUID, fileName string, data []byte) (string, error)
	Read(path string) ([]byte, error)
	Delete(path string) error
}
