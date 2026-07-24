package usecase

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/lexora/backend/internal/domain"
)

const systemPrompt = `Kamu adalah Lexora, asisten hukum untuk praktisi hukum Indonesia.

Aturan:
- Jawab HANYA berdasarkan KONTEKS dokumen yang diberikan. Dilarang mengarang pasal, nomor, atau kutipan.
- Kalau konteks tidak memuat jawabannya, katakan terus terang bahwa dokumen dalam pustaka tidak memuat informasi tersebut. Jangan menebak.
- Rujuk sumber dengan nomor konteks, contoh: [1], [2].
- Jawab dengan bahasa yang dipakai penanya (default Bahasa Indonesia).
- Ringkas, terstruktur, dan pakai istilah hukum yang tepat.
- Tulis teks biasa tanpa markdown (jangan pakai **, ##, atau bullet -). Untuk daftar, pakai penomoran "1." di baris terpisah.`

// dipakai saat tidak ada dokumen relevan: chat biasa, tetap larang mengarang pasal
const generalPrompt = `Kamu adalah Lexora, asisten hukum untuk praktisi hukum Indonesia.

Balas dengan natural dan membantu, seperti asisten percakapan biasa.
Aturan:
- Untuk sapaan atau obrolan umum, jawab wajar dan ramah.
- Untuk pertanyaan hukum umum, jelaskan sebaik pengetahuanmu.
- JANGAN mengarang nomor pasal, nomor peraturan, tahun, atau kutipan spesifik. Kalau tidak yakin atau tidak ada dokumen rujukan di pustaka, katakan terus terang dan sarankan pengguna mengunggah dokumen terkait agar jawaban bisa dirujuk dengan tepat.
- Jawab dengan bahasa yang dipakai penanya (default Bahasa Indonesia).
- Tulis teks biasa tanpa markdown (jangan pakai **, ##, atau bullet -). Untuk daftar, pakai penomoran "1." di baris terpisah.`

type RAG struct {
	chats     domain.ChatRepository
	embedder  domain.Embedder
	vectors   domain.VectorRepository
	llm       domain.LLM
	topK      int
	minScore  float32
	maxTurns  int
	maxChunks int
	billing   *Billing                // ponytail: nil = no quota check (org belum subscribe)
	prompts   domain.PromptRepository // ponytail: nil = pakai systemPrompt hardcoded
	extractor domain.Extractor        // ponytail: nil = lampiran dokumen diabaikan
}

func (r *RAG) SetBilling(b *Billing)                { r.billing = b }
func (r *RAG) SetPrompts(p domain.PromptRepository) { r.prompts = p }
func (r *RAG) SetExtractor(e domain.Extractor)      { r.extractor = e }

// system prompt dari DB kalau ada, fallback ke konstanta
func (r *RAG) systemPrompt(ctx context.Context) string {
	if r.prompts != nil {
		if p, err := r.prompts.Get(ctx, domain.PromptSystem); err == nil && strings.TrimSpace(p.Content) != "" {
			return p.Content
		}
	}
	return systemPrompt
}

func NewRAG(chats domain.ChatRepository, em domain.Embedder, vec domain.VectorRepository, llm domain.LLM, topK int, minScore float32) *RAG {
	return &RAG{
		chats: chats, embedder: em, vectors: vec, llm: llm,
		topK: topK, minScore: minScore, maxTurns: 10, maxChunks: 5,
	}
}

func (r *RAG) CreateChat(ctx context.Context, orgID, userID uuid.UUID, title string) (*domain.Chat, error) {
	if strings.TrimSpace(title) == "" {
		title = "Percakapan baru"
	}
	c := &domain.Chat{OrganizationID: orgID, UserID: userID, Title: trimTitle(title)}
	if err := r.chats.Create(ctx, c); err != nil {
		return nil, err
	}
	return c, nil
}

func (r *RAG) ListChats(ctx context.Context, orgID, userID uuid.UUID, search string, limit, offset int) ([]domain.Chat, error) {
	return r.chats.ListByUser(ctx, orgID, userID, search, limit, offset)
}

func (r *RAG) Chat(ctx context.Context, id, orgID, userID uuid.UUID) (*domain.Chat, error) {
	return r.chats.ByID(ctx, id, orgID, userID)
}

func (r *RAG) Messages(ctx context.Context, chatID, orgID, userID uuid.UUID) ([]domain.Message, error) {
	if _, err := r.chats.ByID(ctx, chatID, orgID, userID); err != nil {
		return nil, err
	}
	return r.chats.Messages(ctx, chatID)
}

func (r *RAG) Rename(ctx context.Context, id, orgID, userID uuid.UUID, title string) error {
	if strings.TrimSpace(title) == "" {
		return domain.ErrInvalidUpload
	}
	return r.chats.Rename(ctx, id, orgID, userID, trimTitle(title))
}

func (r *RAG) SetPinned(ctx context.Context, id, orgID, userID uuid.UUID, pinned bool) error {
	return r.chats.SetPinned(ctx, id, orgID, userID, pinned)
}

func (r *RAG) Delete(ctx context.Context, id, orgID, userID uuid.UUID) error {
	return r.chats.SoftDelete(ctx, id, orgID, userID)
}

// jawab pertanyaan, token dialirkan lewat onToken
func (r *RAG) Ask(ctx context.Context, chatID, orgID, userID uuid.UUID, question string, atts []domain.Attachment, onToken func(string)) (*domain.Message, error) {
	question = strings.TrimSpace(question)
	if question == "" && len(atts) == 0 {
		return nil, domain.ErrInvalidUpload
	}
	chat, err := r.chats.ByID(ctx, chatID, orgID, userID)
	if err != nil {
		return nil, err
	}

	// hard quota gate
	if r.billing != nil {
		if _, err := r.billing.Check(ctx, orgID, time.Now()); err != nil {
			return nil, err
		}
	}

	history, err := r.chats.Messages(ctx, chat.ID)
	if err != nil {
		return nil, err
	}

	images, docText := r.splitAttachments(atts)

	title := question
	if title == "" {
		title = "Lampiran"
	}
	userMsg := &domain.Message{ChatID: chat.ID, Role: domain.RoleUser, Content: title}
	if err := r.chats.AddMessage(ctx, userMsg); err != nil {
		return nil, err
	}
	if len(history) == 0 {
		_ = r.chats.Rename(ctx, chat.ID, orgID, userID, trimTitle(title))
	}

	// ada lampiran: langsung ke LLM, tanpa RAG
	if len(images) > 0 || docText != "" {
		return r.answerWithAttachments(ctx, chat, orgID, userID, question, docText, images, history, onToken)
	}

	hits, err := r.retrieve(ctx, orgID, question)
	if err != nil {
		return nil, err
	}

	msgs := buildTurns(history, r.maxTurns)
	system := r.systemPrompt(ctx)
	if len(hits) == 0 {
		// tanpa dokumen relevan: chat biasa, tanpa RAG
		system = generalPrompt
		msgs = append(msgs, domain.ChatMessage{Role: domain.RoleUser, Content: question})
	} else {
		msgs = append(msgs, domain.ChatMessage{Role: domain.RoleUser, Content: buildPrompt(hits, question)})
	}

	var sb strings.Builder
	usage, err := r.llm.Stream(ctx, system, msgs, func(tok string) {
		sb.WriteString(tok)
		onToken(tok)
	})
	if err != nil {
		return nil, err
	}
	return r.saveAnswer(ctx, chat, orgID, userID, sb.String(), hits, usage)
}

// gambar -> data URL vision; dokumen -> teks (extract). Lampiran tak dikenal diabaikan.
func (r *RAG) splitAttachments(atts []domain.Attachment) (images []string, docText string) {
	var docs strings.Builder
	for _, a := range atts {
		if strings.HasPrefix(a.Mime, "image/") {
			images = append(images, "data:"+a.Mime+";base64,"+base64.StdEncoding.EncodeToString(a.Data))
			continue
		}
		if r.extractor == nil {
			continue
		}
		if txt := r.extractText(a); strings.TrimSpace(txt) != "" {
			fmt.Fprintf(&docs, "\n\n[Dokumen: %s]\n%s", a.Name, txt)
		}
	}
	return images, docs.String()
}

// tulis ke temp file lalu extract (extractor butuh path)
func (r *RAG) extractText(a domain.Attachment) string {
	f, err := os.CreateTemp("", "att-*")
	if err != nil {
		return ""
	}
	defer os.Remove(f.Name())
	if _, err := f.Write(a.Data); err != nil {
		f.Close()
		return ""
	}
	f.Close()
	pages, err := r.extractor.Extract(f.Name(), a.Mime)
	if err != nil {
		return ""
	}
	return strings.Join(pages, "\n")
}

// jawab dengan lampiran (gambar/dokumen), tanpa RAG
func (r *RAG) answerWithAttachments(ctx context.Context, chat *domain.Chat, orgID, userID uuid.UUID, question, docText string, images []string, history []domain.Message, onToken func(string)) (*domain.Message, error) {
	content := question
	if docText != "" {
		content = "DOKUMEN LAMPIRAN:" + docText + "\n\nPERTANYAAN:\n" + question
	}
	msgs := buildTurns(history, r.maxTurns)
	msgs = append(msgs, domain.ChatMessage{Role: domain.RoleUser, Content: content, Images: images})

	var sb strings.Builder
	usage, err := r.llm.Stream(ctx, generalPrompt, msgs, func(tok string) {
		sb.WriteString(tok)
		onToken(tok)
	})
	if err != nil {
		return nil, err
	}
	return r.saveAnswer(ctx, chat, orgID, userID, sb.String(), nil, usage)
}

func (r *RAG) retrieve(ctx context.Context, orgID uuid.UUID, question string) ([]domain.SearchHit, error) {
	vecs, err := r.embedder.Embed(ctx, []string{question})
	if err != nil {
		return nil, fmt.Errorf("embed query: %w", err)
	}
	collection := collectionName(orgID)
	filter := map[string]any{"organization_id": orgID.String()}

	hits, err := r.vectors.Search(ctx, collection, vecs[0], r.topK, filter)
	if err != nil {
		if strings.Contains(err.Error(), "404") {
			return nil, nil
		}
		return nil, err
	}

	var kept []domain.SearchHit
	for _, h := range hits {
		if h.Score >= r.minScore && len(kept) < r.maxChunks {
			kept = append(kept, h)
		}
	}
	return kept, nil
}

func (r *RAG) saveAnswer(ctx context.Context, chat *domain.Chat, orgID, userID uuid.UUID, content string, hits []domain.SearchHit, usage domain.LLMUsage) (*domain.Message, error) {
	model := r.llm.Model()
	msg := &domain.Message{ChatID: chat.ID, Role: domain.RoleAssistant, Content: content, Model: &model}
	if err := r.chats.AddMessage(ctx, msg); err != nil {
		return nil, err
	}

	cits := citationsFrom(msg.ID, hits, content)
	if err := r.chats.AddCitations(ctx, cits); err != nil {
		return nil, err
	}
	msg.Citations = cits

	if usage.InputTokens > 0 || usage.OutputTokens > 0 {
		_ = r.chats.AddUsage(ctx, domain.TokenUsage{
			OrganizationID: orgID, UserID: userID, MessageID: msg.ID, Model: model,
			InputTokens: usage.InputTokens, OutputTokens: usage.OutputTokens,
		})
	}
	_ = r.chats.Touch(ctx, chat.ID)
	return msg, nil
}

// hanya chunk yang benar-benar dirujuk jawaban
func citationsFrom(messageID uuid.UUID, hits []domain.SearchHit, answer string) []domain.Citation {
	cits := make([]domain.Citation, 0, len(hits))
	for i, h := range hits {
		if !strings.Contains(answer, fmt.Sprintf("[%d]", i+1)) {
			continue
		}
		c := domain.Citation{MessageID: messageID, ReferenceLabel: payloadString(h.Payload, "file_name"), Marker: i + 1, Score: h.Score}
		if c.ReferenceLabel == "" {
			c.ReferenceLabel = "dokumen"
		}
		if id, err := uuid.Parse(payloadString(h.Payload, "document_id")); err == nil {
			c.DocumentID = &id
		}
		if page, ok := payloadInt(h.Payload, "page_no"); ok {
			c.PageNo = &page
		}
		cits = append(cits, c)
	}
	return cits
}

func buildPrompt(hits []domain.SearchHit, question string) string {
	var sb strings.Builder
	sb.WriteString("KONTEKS:\n")
	for i, h := range hits {
		sb.WriteString(fmt.Sprintf("[%d] %s", i+1, payloadString(h.Payload, "file_name")))
		if page, ok := payloadInt(h.Payload, "page_no"); ok {
			sb.WriteString(fmt.Sprintf(" (hal. %d)", page))
		}
		sb.WriteString("\n")
		sb.WriteString(payloadString(h.Payload, "text"))
		sb.WriteString("\n\n")
	}
	sb.WriteString("PERTANYAAN:\n")
	sb.WriteString(question)
	return sb.String()
}

func buildTurns(history []domain.Message, maxTurns int) []domain.ChatMessage {
	if len(history) > maxTurns {
		history = history[len(history)-maxTurns:]
	}
	out := make([]domain.ChatMessage, 0, len(history))
	for _, m := range history {
		out = append(out, domain.ChatMessage{Role: m.Role, Content: m.Content})
	}
	return out
}

func payloadString(p map[string]any, key string) string {
	if v, ok := p[key].(string); ok {
		return v
	}
	return ""
}

func payloadInt(p map[string]any, key string) (int, bool) {
	switch v := p[key].(type) {
	case float64:
		return int(v), true
	case int:
		return v, true
	}
	return 0, false
}

func trimTitle(s string) string {
	s = strings.TrimSpace(strings.ReplaceAll(s, "\n", " "))
	if len(s) > 60 {
		s = strings.TrimSpace(s[:60]) + "…"
	}
	return s
}
