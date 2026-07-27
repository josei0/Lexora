package usecase

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/lexora/backend/internal/domain"
	"github.com/lexora/backend/pkg/websearch"
)

// fetcher halaman jadi teks; dipisah interface biar bisa di-fake di test
type PageFetcher interface {
	FetchText(ctx context.Context, url string) (title, text string, err error)
}

// cari sumber web + promosikan jadi dokumen pustaka.
// Ingest memakai ulang pipeline ingestion apa adanya: halaman -> .txt -> Enqueue.
type WebIngest struct {
	docs     domain.DocumentRepository
	storage  domain.Storage
	fetcher  PageFetcher
	provider websearch.Provider
	ingest   enqueuer
	billing  *Billing                    // ponytail: nil = tanpa gate langganan
	wsRepo   domain.WebSearchRepository  // ponytail: nil = kandidat tidak tersedia
}

func NewWebIngest(docs domain.DocumentRepository, st domain.Storage, f PageFetcher, p websearch.Provider, ing enqueuer) *WebIngest {
	return &WebIngest{docs: docs, storage: st, fetcher: f, provider: p, ingest: ing}
}

func (w *WebIngest) SetBilling(b *Billing)                       { w.billing = b }
func (w *WebIngest) SetWebSearchRepo(r domain.WebSearchRepository) { w.wsRepo = r }

// URL populer dari pencarian user yang belum ada di pustaka (30 hari, >= 3 hit)
func (w *WebIngest) Candidates(ctx context.Context, orgID uuid.UUID) ([]domain.WebCandidate, error) {
	if w.wsRepo == nil {
		return nil, nil
	}
	since := time.Now().AddDate(0, 0, -30)
	return w.wsRepo.Candidates(ctx, orgID, 3, since)
}

func (w *WebIngest) Search(ctx context.Context, query string, limit int) ([]websearch.Result, error) {
	if strings.TrimSpace(query) == "" {
		return nil, domain.ErrInvalidUpload
	}
	if limit <= 0 || limit > 10 {
		limit = 4
	}
	return w.provider.Search(ctx, query, limit)
}

// pratinjau: teks yang benar-benar akan di-ingest, bukan cuplikan.
// Layar ini titik validasi manusia - tanpa itu fitur berubah jadi scraper.
func (w *WebIngest) Preview(ctx context.Context, url string) (title, text string, err error) {
	title, text, err = w.fetcher.FetchText(ctx, url)
	if err != nil {
		return "", "", err
	}
	if strings.TrimSpace(text) == "" {
		return "", "", fmt.Errorf("halaman tidak memuat teks")
	}
	return title, text, nil
}

// promosikan halaman jadi dokumen pustaka org
func (w *WebIngest) FromURL(ctx context.Context, orgID, userID uuid.UUID, url string) (*domain.Document, error) {
	if w.billing != nil {
		if err := w.billing.GateAccess(ctx, orgID, time.Now()); err != nil {
			return nil, err
		}
	}
	// anti-duplikat: satu URL satu dokumen per org
	if _, err := w.docs.BySourceURL(ctx, orgID, url); err == nil {
		return nil, domain.ErrConflict
	} else if err != domain.ErrNotFound {
		return nil, err
	}

	title, text, err := w.Preview(ctx, url)
	if err != nil {
		return nil, err
	}
	if title == "" {
		title = url
	}

	doc := &domain.Document{
		ID:             uuid.New(),
		OrganizationID: orgID,
		UploadedBy:     userID,
		Scope:          domain.DocScopeKnowledgeBase,
		FileName:       safeFileName(title) + ".txt",
		MimeType:       "text/plain",
		FileSize:       int64(len(text)),
		Status:         domain.DocStatusUploaded,
		SourceURL:      &url,
	}
	// sumber ditulis di badan teks juga: ikut ter-chunk, jadi konteks LLM
	// selalu membawa asal-usulnya
	body := "Sumber: " + url + "\nJudul: " + title + "\n\n" + text

	path, err := w.storage.Save(orgID, doc.ID, doc.FileName, []byte(body))
	if err != nil {
		return nil, err
	}
	doc.StoragePath = path

	if err := w.docs.Create(ctx, doc); err != nil {
		return nil, err
	}
	w.ingest.Enqueue(doc.ID)
	return doc, nil
}

func safeFileName(s string) string {
	s = strings.TrimSpace(strings.ReplaceAll(s, "\n", " "))
	out := make([]rune, 0, 80)
	for _, r := range s {
		if strings.ContainsRune(`/\:*?"<>|`, r) {
			r = '_'
		}
		out = append(out, r)
		if len(out) >= 80 {
			break
		}
	}
	if len(out) == 0 {
		return "sumber-web"
	}
	return strings.TrimSpace(string(out))
}
