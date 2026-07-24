package usecase

import (
	"context"
	"net/http"
	"strings"

	"github.com/google/uuid"

	"github.com/lexora/backend/internal/domain"
)

const MaxUploadBytes = 20 << 20 // 20MB

// allowed mime prefixes (sniffed from content)
var allowedMIME = map[string]string{
	"application/pdf": "pdf",
	"text/plain":      "txt",
	// docx sniffs as zip; handled specially below
	"application/zip": "docx",
	"application/vnd.openxmlformats-officedocument.wordprocessingml.document": "docx",
}

type enqueuer interface{ Enqueue(id uuid.UUID) }

type Document struct {
	docs    domain.DocumentRepository
	storage domain.Storage
	ingest  enqueuer
}

func NewDocument(docs domain.DocumentRepository, st domain.Storage, ingest enqueuer) *Document {
	return &Document{docs: docs, storage: st, ingest: ingest}
}

// upload: validate -> store -> insert(uploaded) -> enqueue. returns 202 doc
func (d *Document) Upload(ctx context.Context, orgID, userID uuid.UUID, fileName string, data []byte) (*domain.Document, error) {
	if len(data) == 0 {
		return nil, domain.ErrInvalidUpload
	}
	if len(data) > MaxUploadBytes {
		return nil, domain.ErrTooLarge
	}

	// sniff content, not extension
	sniff := http.DetectContentType(data)
	mime := normalizeMIME(sniff, fileName)
	if mime == "" {
		return nil, domain.ErrUnsupportedType
	}

	doc := &domain.Document{
		ID:             uuid.New(),
		OrganizationID: orgID,
		UploadedBy:     userID,
		Scope:          domain.DocScopeKnowledgeBase,
		FileName:       fileName,
		MimeType:       mime,
		FileSize:       int64(len(data)),
		Status:         domain.DocStatusUploaded,
	}

	path, err := d.storage.Save(orgID, doc.ID, fileName, data)
	if err != nil {
		return nil, err
	}
	doc.StoragePath = path

	if err := d.docs.Create(ctx, doc); err != nil {
		return nil, err
	}
	d.ingest.Enqueue(doc.ID)
	return doc, nil
}

func (d *Document) List(ctx context.Context, orgID uuid.UUID, limit, offset int) ([]domain.Document, error) {
	return d.docs.ListByOrg(ctx, orgID, limit, offset)
}

// map sniffed type (+ extension for docx/txt edge cases) to canonical mime
func normalizeMIME(sniff, fileName string) string {
	base := strings.SplitN(sniff, ";", 2)[0]
	lower := strings.ToLower(fileName)
	switch {
	case strings.HasPrefix(base, "application/pdf"):
		return "application/pdf"
	case strings.HasPrefix(base, "text/plain"):
		return "text/plain"
	case strings.HasPrefix(base, "application/zip") && strings.HasSuffix(lower, ".docx"):
		return "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
	case strings.HasSuffix(lower, ".txt"):
		return "text/plain"
	}
	return ""
}
