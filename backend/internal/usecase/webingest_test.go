package usecase

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/lexora/backend/internal/domain"
	"github.com/lexora/backend/pkg/websearch"
)

type fakeFetcher struct {
	title, text string
	err         error
	calls       int
}

func (f *fakeFetcher) FetchText(context.Context, string) (string, string, error) {
	f.calls++
	return f.title, f.text, f.err
}

type fakeProvider struct{ res []websearch.Result }

func (f *fakeProvider) Name() string { return "fake" }
func (f *fakeProvider) Search(context.Context, string, int) ([]websearch.Result, error) {
	return f.res, nil
}

// docs repo minimal untuk WebIngest
type webDocs struct {
	created []domain.Document
	bySrc   map[string]bool
}

func (d *webDocs) Create(_ context.Context, doc *domain.Document) error {
	d.created = append(d.created, *doc)
	if doc.SourceURL != nil {
		d.bySrc[*doc.SourceURL] = true
	}
	return nil
}
func (d *webDocs) BySourceURL(_ context.Context, _ uuid.UUID, url string) (*domain.Document, error) {
	if d.bySrc[url] {
		return &domain.Document{}, nil
	}
	return nil, domain.ErrNotFound
}
func (d *webDocs) ListByOrg(context.Context, uuid.UUID, int, int) ([]domain.Document, error) {
	return nil, nil
}
func (d *webDocs) ByID(context.Context, uuid.UUID, uuid.UUID) (*domain.Document, error) {
	return nil, domain.ErrNotFound
}
func (d *webDocs) SetStatus(context.Context, uuid.UUID, string, *string) error { return nil }
func (d *webDocs) PendingIDs(context.Context) ([]uuid.UUID, error)             { return nil, nil }
func (d *webDocs) AnyByID(context.Context, uuid.UUID) (*domain.Document, error) {
	return nil, domain.ErrNotFound
}
func (d *webDocs) InsertChunks(context.Context, []domain.DocumentChunk) error { return nil }
func (d *webDocs) DeleteChunks(context.Context, uuid.UUID) error              { return nil }

type memStorage struct{ saved map[string][]byte }

func (m *memStorage) Save(_ uuid.UUID, docID uuid.UUID, name string, data []byte) (string, error) {
	p := docID.String() + "/" + name
	m.saved[p] = data
	return p, nil
}
func (m *memStorage) Read(p string) ([]byte, error) { return m.saved[p], nil }
func (m *memStorage) Delete(string) error           { return nil }

type spyQueue struct{ ids []uuid.UUID }

func (s *spyQueue) Enqueue(id uuid.UUID) { s.ids = append(s.ids, id) }

func setupWeb(text string) (*WebIngest, *webDocs, *memStorage, *spyQueue) {
	docs := &webDocs{bySrc: map[string]bool{}}
	store := &memStorage{saved: map[string][]byte{}}
	q := &spyQueue{}
	w := NewWebIngest(docs, store, &fakeFetcher{title: "UU 37/2004", text: text}, &fakeProvider{}, q)
	return w, docs, store, q
}

const srcURL = "https://peraturan.bpk.go.id/Details/40784"

// ingest = dokumen pustaka + masuk antrean pipeline yang sudah ada
func TestWebIngestCreatesKBDocAndEnqueues(t *testing.T) {
	w, docs, store, q := setupWeb("Pasal 222 ayat (1) mengatur PKPU.")

	doc, err := w.FromURL(context.Background(), uuid.New(), uuid.New(), srcURL)
	if err != nil {
		t.Fatal(err)
	}
	if doc.Scope != domain.DocScopeKnowledgeBase || doc.MimeType != "text/plain" {
		t.Fatalf("dokumen harus KB .txt: %+v", doc)
	}
	if doc.SourceURL == nil || *doc.SourceURL != srcURL {
		t.Fatal("source_url tidak tersimpan")
	}
	if len(q.ids) != 1 || q.ids[0] != doc.ID {
		t.Fatalf("harus masuk antrean ingestion, got %v", q.ids)
	}
	if len(docs.created) != 1 {
		t.Fatalf("dokumen = %d", len(docs.created))
	}
	// asal-usul ikut ter-chunk supaya konteks LLM selalu bawa sumbernya
	body := string(store.saved[doc.StoragePath])
	if !strings.Contains(body, srcURL) || !strings.Contains(body, "Pasal 222") {
		t.Fatalf("badan dokumen tidak lengkap: %q", body)
	}
}

// U14: URL yang sudah pernah di-ingest tidak menggandakan dokumen
func TestWebIngestRejectsDuplicate(t *testing.T) {
	w, _, _, q := setupWeb("isi")
	org, user := uuid.New(), uuid.New()

	if _, err := w.FromURL(context.Background(), org, user, srcURL); err != nil {
		t.Fatal(err)
	}
	if _, err := w.FromURL(context.Background(), org, user, srcURL); err != domain.ErrConflict {
		t.Fatalf("URL duplikat harus ErrConflict, dapat %v", err)
	}
	if len(q.ids) != 1 {
		t.Fatalf("duplikat tidak boleh ikut antre: %v", q.ids)
	}
}

// guard/fetch gagal -> tidak ada dokumen setengah jadi
func TestWebIngestFetchFailureLeavesNothing(t *testing.T) {
	docs := &webDocs{bySrc: map[string]bool{}}
	store := &memStorage{saved: map[string][]byte{}}
	q := &spyQueue{}
	w := NewWebIngest(docs, store, &fakeFetcher{err: fmt.Errorf("domain di luar allowlist")}, &fakeProvider{}, q)

	if _, err := w.FromURL(context.Background(), uuid.New(), uuid.New(), "https://jahat.com/x"); err == nil {
		t.Fatal("fetch gagal harus error")
	}
	if len(docs.created) != 0 || len(store.saved) != 0 || len(q.ids) != 0 {
		t.Fatal("tidak boleh ada sisa dokumen/file/antrean saat fetch gagal")
	}
}

// halaman kosong ditolak: jangan indeks dokumen tanpa teks
func TestWebIngestRejectsEmptyPage(t *testing.T) {
	w, docs, _, _ := setupWeb("   \n  ")
	if _, err := w.FromURL(context.Background(), uuid.New(), uuid.New(), srcURL); err == nil {
		t.Fatal("halaman tanpa teks harus ditolak")
	}
	if len(docs.created) != 0 {
		t.Fatal("dokumen kosong tidak boleh dibuat")
	}
}
