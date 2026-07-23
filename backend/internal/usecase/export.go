package usecase

import (
	"context"
	"fmt"
	"html"
	"strings"

	"github.com/google/uuid"
	"github.com/lexora/backend/internal/domain"
)

// Export: render chat ke HTML mandiri. Word buka application/msword;
// PDF lewat print browser (window.print) di FE. Nol dependency baru.
type Export struct {
	rag *RAG
}

func NewExport(rag *RAG) *Export { return &Export{rag: rag} }

// HTML lengkap satu percakapan, scoped org + user
func (e *Export) ChatHTML(ctx context.Context, chatID, orgID, userID uuid.UUID) (title, htmlDoc string, err error) {
	chat, err := e.rag.Chat(ctx, chatID, orgID, userID)
	if err != nil {
		return "", "", err
	}
	msgs, err := e.rag.Messages(ctx, chatID, orgID, userID)
	if err != nil {
		return "", "", err
	}
	return chat.Title, renderChat(chat.Title, msgs), nil
}

func renderChat(title string, msgs []domain.Message) string {
	var b strings.Builder
	b.WriteString(`<!DOCTYPE html><html lang="id"><head><meta charset="utf-8"><title>`)
	b.WriteString(html.EscapeString(title))
	b.WriteString(`</title><style>
body{font-family:Georgia,'Times New Roman',serif;max-width:720px;margin:40px auto;color:#1a1a1a;line-height:1.6;padding:0 24px}
h1{font-size:22px;border-bottom:2px solid #333;padding-bottom:8px}
.msg{margin:18px 0}
.role{font-weight:bold;font-size:12px;text-transform:uppercase;letter-spacing:.05em;color:#555}
.content{white-space:pre-wrap;margin-top:4px}
.cits{margin-top:8px;font-size:12px;color:#555;border-left:3px solid #ccc;padding-left:10px}
.cite{margin:2px 0}
.brand{color:#888;font-size:11px;margin-top:40px;text-align:center}
@media print{body{margin:0}}
</style></head><body>`)
	b.WriteString(`<h1>` + html.EscapeString(title) + `</h1>`)

	for _, m := range msgs {
		role := "Anda"
		if m.Role == domain.RoleAssistant {
			role = "Lexora"
		}
		b.WriteString(`<div class="msg"><div class="role">` + role + `</div>`)
		b.WriteString(`<div class="content">` + html.EscapeString(m.Content) + `</div>`)
		if len(m.Citations) > 0 {
			b.WriteString(`<div class="cits">Sumber:`)
			for _, c := range m.Citations {
				line := fmt.Sprintf("[%d] %s", c.Marker, c.ReferenceLabel)
				if c.PageNo != nil {
					line += fmt.Sprintf(" (hal. %d)", *c.PageNo)
				}
				b.WriteString(`<div class="cite">` + html.EscapeString(line) + `</div>`)
			}
			b.WriteString(`</div>`)
		}
		b.WriteString(`</div>`)
	}
	b.WriteString(`<div class="brand">Diekspor dari Lexora — asisten hukum Indonesia</div>`)
	b.WriteString(`</body></html>`)
	return b.String()
}
