package usecase

import (
	"context"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/lexora/backend/internal/domain"
)

type fakeChats struct {
	chat     *domain.Chat
	messages []domain.Message
	cits     []domain.Citation
	usage    []domain.TokenUsage
}

func (f *fakeChats) Create(_ context.Context, c *domain.Chat) error { c.ID = uuid.New(); return nil }
func (f *fakeChats) ByID(_ context.Context, id, orgID, userID uuid.UUID) (*domain.Chat, error) {
	if f.chat == nil || f.chat.ID != id || f.chat.OrganizationID != orgID || f.chat.UserID != userID {
		return nil, domain.ErrNotFound
	}
	return f.chat, nil
}
func (f *fakeChats) ListByUser(context.Context, uuid.UUID, uuid.UUID, string, int, int) ([]domain.Chat, error) {
	return nil, nil
}
func (f *fakeChats) Rename(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, string) error {
	return nil
}
func (f *fakeChats) SetPinned(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, bool) error {
	return nil
}
func (f *fakeChats) SoftDelete(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) error { return nil }
func (f *fakeChats) Touch(context.Context, uuid.UUID) error                            { return nil }
func (f *fakeChats) AddMessage(_ context.Context, m *domain.Message) error {
	m.ID = uuid.New()
	f.messages = append(f.messages, *m)
	return nil
}
func (f *fakeChats) Messages(context.Context, uuid.UUID) ([]domain.Message, error) {
	return f.messages, nil
}
func (f *fakeChats) AddCitations(_ context.Context, cs []domain.Citation) error {
	f.cits = append(f.cits, cs...)
	return nil
}
func (f *fakeChats) AddUsage(_ context.Context, u domain.TokenUsage) error {
	f.usage = append(f.usage, u)
	return nil
}

type stubVectors struct{ hits []domain.SearchHit }

func (s *stubVectors) EnsureCollection(context.Context, string, int) error        { return nil }
func (s *stubVectors) Upsert(context.Context, string, []domain.VectorPoint) error { return nil }
func (s *stubVectors) DeleteByDocument(context.Context, string, string) error     { return nil }
func (s *stubVectors) Search(context.Context, string, []float32, int, map[string]any) ([]domain.SearchHit, error) {
	return s.hits, nil
}

type stubLLM struct{ answer string }

func (s *stubLLM) Model() string { return "test-model" }
func (s *stubLLM) Stream(_ context.Context, _ string, _ []domain.ChatMessage, onToken func(string)) (domain.LLMUsage, error) {
	onToken(s.answer)
	return domain.LLMUsage{InputTokens: 100, OutputTokens: 20}, nil
}

func hit(file string, page int, score float32) domain.SearchHit {
	return domain.SearchHit{Payload: map[string]any{
		"file_name": file, "page_no": float64(page), "text": "isi pasal",
		"document_id": uuid.New().String(),
	}, Score: score}
}

func setup(hits []domain.SearchHit, answer string) (*RAG, *fakeChats, *domain.Chat) {
	chat := &domain.Chat{ID: uuid.New(), OrganizationID: uuid.New(), UserID: uuid.New()}
	chats := &fakeChats{chat: chat}
	rag := NewRAG(chats, fakeEmbedder{}, &stubVectors{hits: hits}, &stubLLM{answer: answer}, 5, 0.35)
	return rag, chats, chat
}

func TestAskKeepsOnlyCitedChunks(t *testing.T) {
	hits := []domain.SearchHit{hit("uu-a.pdf", 3, 0.8), hit("uu-b.pdf", 9, 0.6)}
	rag, chats, chat := setup(hits, "Syaratnya diatur di Pasal 222 [2].")

	msg, err := rag.Ask(context.Background(), chat.ID, chat.OrganizationID, chat.UserID, "syarat PKPU?", func(string) {})
	if err != nil {
		t.Fatal(err)
	}
	if len(msg.Citations) != 1 {
		t.Fatalf("citation = %d, mau 1 (cuma [2] yang dirujuk)", len(msg.Citations))
	}
	if msg.Citations[0].ReferenceLabel != "uu-b.pdf" || *msg.Citations[0].PageNo != 9 {
		t.Fatalf("citation salah sasaran: %+v", msg.Citations[0])
	}
	if msg.Citations[0].Marker != 2 {
		t.Fatalf("marker = %d, mau 2 (nomor harus ikut urutan konteks, bukan urutan simpan)", msg.Citations[0].Marker)
	}
	if len(chats.usage) != 1 || chats.usage[0].InputTokens != 100 {
		t.Fatalf("token usage tidak tercatat: %+v", chats.usage)
	}
}

func TestAskWithoutCitationMarkerSavesNone(t *testing.T) {
	hits := []domain.SearchHit{hit("uu-a.pdf", 3, 0.8)}
	rag, _, chat := setup(hits, "Dokumen tidak memuat informasi tersebut.")

	msg, err := rag.Ask(context.Background(), chat.ID, chat.OrganizationID, chat.UserID, "resep kue?", func(string) {})
	if err != nil {
		t.Fatal(err)
	}
	if len(msg.Citations) != 0 {
		t.Fatalf("citation palsu: %+v", msg.Citations)
	}
}

func TestAskNoRelevantChunkAnswersHonestly(t *testing.T) {
	hits := []domain.SearchHit{hit("uu-a.pdf", 3, 0.2)}
	rag, _, chat := setup(hits, "jawaban model yang seharusnya tidak dipakai")

	var streamed strings.Builder
	msg, err := rag.Ask(context.Background(), chat.ID, chat.OrganizationID, chat.UserID, "resep kue?", func(tok string) {
		streamed.WriteString(tok)
	})
	if err != nil {
		t.Fatal(err)
	}
	if msg.Content != noContextAnswer || streamed.String() != noContextAnswer {
		t.Fatalf("model dipanggil padahal skor di bawah threshold: %q", msg.Content)
	}
	if len(msg.Citations) != 0 {
		t.Fatalf("citation muncul tanpa chunk relevan: %+v", msg.Citations)
	}
}

func TestAskRejectsOtherTenantChat(t *testing.T) {
	rag, _, chat := setup([]domain.SearchHit{hit("uu-a.pdf", 1, 0.9)}, "jawab [1]")

	_, err := rag.Ask(context.Background(), chat.ID, uuid.New(), chat.UserID, "halo", func(string) {})
	if err != domain.ErrNotFound {
		t.Fatalf("chat org lain harusnya ErrNotFound, dapat %v", err)
	}
}
