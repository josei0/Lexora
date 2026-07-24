package usecase

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"github.com/google/uuid"
	"github.com/lexora/backend/internal/domain"
	"github.com/lexora/backend/pkg/chunk"
)

// ingestion worker pool: extract -> chunk -> embed -> upsert qdrant
type Ingestion struct {
	docs      domain.DocumentRepository
	storage   domain.Storage
	extractor domain.Extractor
	embedder  domain.Embedder
	vectors   domain.VectorRepository
	queue     chan uuid.UUID
	wg        sync.WaitGroup
}

func NewIngestion(docs domain.DocumentRepository, st domain.Storage, ex domain.Extractor, em domain.Embedder, vec domain.VectorRepository) *Ingestion {
	return &Ingestion{
		docs: docs, storage: st, extractor: ex, embedder: em, vectors: vec,
		queue: make(chan uuid.UUID, 256),
	}
}

func (i *Ingestion) Start(ctx context.Context, workers int) {
	for w := 0; w < workers; w++ {
		i.wg.Add(1)
		go func() {
			defer i.wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				case id := <-i.queue:
					i.process(ctx, id)
				}
			}
		}()
	}
}

// enqueue a doc for processing (non-blocking best-effort)
func (i *Ingestion) Enqueue(id uuid.UUID) {
	select {
	case i.queue <- id:
	default:
		slog.Warn("ingestion queue full", "doc", id)
		go func() { i.queue <- id }()
	}
}

// startup recovery: re-enqueue docs stuck uploaded/processing
func (i *Ingestion) Recover(ctx context.Context) {
	ids, err := i.docs.PendingIDs(ctx)
	if err != nil {
		slog.Error("ingestion recover", "err", err)
		return
	}
	for _, id := range ids {
		i.Enqueue(id)
	}
	if len(ids) > 0 {
		slog.Info("ingestion recovered pending docs", "count", len(ids))
	}
}

func collectionName(orgID uuid.UUID) string { return "kb_" + orgID.String() }

// full pipeline for one document. failure -> status=failed (never silent)
func (i *Ingestion) process(ctx context.Context, id uuid.UUID) {
	doc, err := i.docs.AnyByID(ctx, id)
	if err != nil {
		slog.Error("ingestion load doc", "doc", id, "err", err)
		return
	}
	if err := i.docs.SetStatus(ctx, id, domain.DocStatusProcessing, nil); err != nil {
		slog.Error("ingestion set processing", "doc", id, "err", err)
		return
	}

	if err := i.run(ctx, doc); err != nil {
		msg := err.Error()
		slog.Error("ingestion failed", "doc", id, "err", msg)
		_ = i.docs.SetStatus(ctx, id, domain.DocStatusFailed, &msg)
		return
	}
	_ = i.docs.SetStatus(ctx, id, domain.DocStatusIndexed, nil)
	slog.Info("ingestion indexed", "doc", id)
}

func (i *Ingestion) run(ctx context.Context, doc *domain.Document) error {
	pages, err := i.extractor.Extract(doc.StoragePath, doc.MimeType)
	if err != nil {
		return fmt.Errorf("extract: %w", err)
	}
	chunks := chunk.Pages(pages)
	if len(chunks) == 0 {
		return fmt.Errorf("no text extracted")
	}

	texts := make([]string, len(chunks))
	for j, c := range chunks {
		texts[j] = c.Text
	}
	vectors, err := i.embedInBatches(ctx, texts)
	if err != nil {
		return fmt.Errorf("embed: %w", err)
	}

	collection := collectionName(doc.OrganizationID)
	if err := i.vectors.EnsureCollection(ctx, collection, i.embedder.Dim()); err != nil {
		return fmt.Errorf("ensure collection: %w", err)
	}

	points := make([]domain.VectorPoint, len(chunks))
	dbChunks := make([]domain.DocumentChunk, len(chunks))
	for j, c := range chunks {
		pointID := uuid.New().String()
		payload := map[string]any{
			"document_id":     doc.ID.String(),
			"organization_id": doc.OrganizationID.String(),
			"scope":           doc.Scope,
			"chunk_index":     c.Index,
			"text":            c.Text,
			"file_name":       doc.FileName,
		}
		if c.Page != nil {
			payload["page_no"] = *c.Page
		}
		points[j] = domain.VectorPoint{ID: pointID, Vector: vectors[j], Payload: payload}
		dbChunks[j] = domain.DocumentChunk{DocumentID: doc.ID, ChunkIndex: c.Index, PageNo: c.Page, QdrantPointID: pointID}
	}

	// hindari duplikat
	if err := i.vectors.DeleteByDocument(ctx, collection, doc.ID.String()); err != nil {
		return fmt.Errorf("delete old points: %w", err)
	}
	if err := i.docs.DeleteChunks(ctx, doc.ID); err != nil {
		return fmt.Errorf("delete old chunks: %w", err)
	}

	if err := i.vectors.Upsert(ctx, collection, points); err != nil {
		return fmt.Errorf("upsert: %w", err)
	}
	if err := i.docs.InsertChunks(ctx, dbChunks); err != nil {
		return fmt.Errorf("insert chunks: %w", err)
	}
	return nil
}

// embed in batches of 96 (provider limit safety)
func (i *Ingestion) embedInBatches(ctx context.Context, texts []string) ([][]float32, error) {
	const batch = 96
	var out [][]float32
	for start := 0; start < len(texts); start += batch {
		end := start + batch
		if end > len(texts) {
			end = len(texts)
		}
		vecs, err := i.embedder.Embed(ctx, texts[start:end])
		if err != nil {
			return nil, err
		}
		out = append(out, vecs...)
	}
	return out, nil
}

func (i *Ingestion) Wait() { i.wg.Wait() }
