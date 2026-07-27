package usecase

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/lexora/backend/internal/domain"
)

// in-memory fakes

type fakeDocs struct {
	doc     *domain.Document
	chunks  []domain.DocumentChunk
	status  []string
	deletes int
}

func (f *fakeDocs) Create(context.Context, *domain.Document) error { return nil }
func (f *fakeDocs) ListByOrg(context.Context, uuid.UUID, int, int) ([]domain.Document, error) {
	return nil, nil
}
func (f *fakeDocs) ByID(context.Context, uuid.UUID, uuid.UUID) (*domain.Document, error) {
	return f.doc, nil
}
func (f *fakeDocs) SetStatus(_ context.Context, _ uuid.UUID, s string, _ *string) error {
	f.status = append(f.status, s)
	return nil
}
func (f *fakeDocs) PendingIDs(context.Context) ([]uuid.UUID, error) { return nil, nil }
func (f *fakeDocs) AnyByID(context.Context, uuid.UUID) (*domain.Document, error) {
	return f.doc, nil
}
func (f *fakeDocs) InsertChunks(_ context.Context, c []domain.DocumentChunk) error {
	f.chunks = append(f.chunks, c...)
	return nil
}
func (f *fakeDocs) BySourceURL(context.Context, uuid.UUID, string) (*domain.Document, error) {
	return nil, domain.ErrNotFound
}
func (f *fakeDocs) DeleteChunks(_ context.Context, _ uuid.UUID) error {
	f.deletes++
	f.chunks = nil
	return nil
}

type fakeExtractor struct{ pages []string }

func (f *fakeExtractor) Extract(string, string) ([]string, error) { return f.pages, nil }

type fakeEmbedder struct{}

func (fakeEmbedder) Embed(_ context.Context, texts []string) ([][]float32, error) {
	out := make([][]float32, len(texts))
	for i := range texts {
		out[i] = []float32{float32(i), 0.1, 0.2}
	}
	return out, nil
}
func (fakeEmbedder) Dim() int { return 3 }

type fakeVectors struct {
	points  []domain.VectorPoint
	deletes int
}

func (f *fakeVectors) EnsureCollection(context.Context, string, int) error { return nil }
func (f *fakeVectors) Upsert(_ context.Context, _ string, p []domain.VectorPoint) error {
	f.points = append(f.points, p...)
	return nil
}
func (f *fakeVectors) Search(context.Context, string, []float32, int, map[string]any) ([]domain.SearchHit, error) {
	return nil, nil
}
func (f *fakeVectors) DeleteByDocument(_ context.Context, _, _ string) error {
	f.deletes++
	f.points = nil
	return nil
}

type fakeStorage struct{}

func (fakeStorage) Save(uuid.UUID, uuid.UUID, string, []byte) (string, error) { return "", nil }
func (fakeStorage) Read(string) ([]byte, error)                               { return nil, nil }
func (fakeStorage) Delete(string) error                                       { return nil }

func TestIngestionReprocessIsIdempotent(t *testing.T) {
	doc := &domain.Document{ID: uuid.New(), OrganizationID: uuid.New(), Scope: domain.DocScopeKnowledgeBase}
	docs := &fakeDocs{doc: doc}
	vecs := &fakeVectors{}
	ing := NewIngestion(docs, fakeStorage{}, &fakeExtractor{pages: []string{"Pasal 1 cukup jelas.", "Pasal 2 cukup jelas."}}, fakeEmbedder{}, vecs)

	ing.process(context.Background(), doc.ID)
	first := len(vecs.points)
	if first == 0 || len(docs.chunks) != first {
		t.Fatalf("run pertama: %d point, %d chunk", first, len(docs.chunks))
	}

	ing.process(context.Background(), doc.ID) // recovery
	if len(vecs.points) != first || len(docs.chunks) != first {
		t.Fatalf("reprocess duplikat: %d point, %d chunk (harusnya %d)", len(vecs.points), len(docs.chunks), first)
	}
	if vecs.deletes != 2 || docs.deletes != 2 {
		t.Fatalf("cleanup tidak jalan: vectors=%d chunks=%d", vecs.deletes, docs.deletes)
	}
	if last := docs.status[len(docs.status)-1]; last != domain.DocStatusIndexed {
		t.Fatalf("status akhir = %s", last)
	}
}
