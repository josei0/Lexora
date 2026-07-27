package websearch

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Result struct {
	Title   string
	URL     string
	Content string
}

type Provider interface {
	Search(ctx context.Context, query string, limit int) ([]Result, error)
	Name() string
}

// Maia (OpenAI-compatible) dgn model server-side search. Provider yang fetch,
// backend cuma parse annotations -> tidak perlu SSRF guard di jalur ini.
// Spike 2026-07-26: openai/gpt-4o-mini-search-preview balikin url_citation terstruktur.
type MaiaSearch struct {
	baseURL, apiKey, model string
	allowed                []string // allowlist domain, disuntik ke prompt
	http                   *http.Client
}

func NewMaiaSearch(baseURL, apiKey, model string, allowed []string) *MaiaSearch {
	return &MaiaSearch{
		baseURL: baseURL, apiKey: apiKey, model: model, allowed: allowed,
		http: &http.Client{Timeout: 60 * time.Second},
	}
}

func (m *MaiaSearch) Name() string { return m.model }

type searchResp struct {
	Choices []struct {
		Message struct {
			Content     string `json:"content"`
			Annotations []struct {
				Type        string `json:"type"`
				URLCitation struct {
					Title string `json:"title"`
					URL   string `json:"url"`
				} `json:"url_citation"`
			} `json:"annotations"`
		} `json:"message"`
	} `json:"choices"`
}

func (m *MaiaSearch) Search(ctx context.Context, query string, limit int) ([]Result, error) {
	body, err := json.Marshal(map[string]any{
		"model":      m.model,
		"max_tokens": 1200,
		"messages": []map[string]string{
			{"role": "user", "content": m.prompt(query)},
		},
	})
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, m.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+m.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := m.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		var buf bytes.Buffer
		buf.ReadFrom(resp.Body)
		return nil, fmt.Errorf("search %d: %s", resp.StatusCode, buf.String())
	}
	var sr searchResp
	if err := json.NewDecoder(resp.Body).Decode(&sr); err != nil {
		return nil, err
	}
	if len(sr.Choices) == 0 {
		return nil, nil
	}
	msg := sr.Choices[0].Message

	// ringkasan tersintesis dilekatkan ke tiap sumber; delimiter + label
	// "tidak tepercaya" dipasang di sisi RAG (update5 §6.2)
	var out []Result
	seen := map[string]bool{}
	for _, a := range msg.Annotations {
		u := a.URLCitation.URL
		if a.Type != "url_citation" || u == "" || seen[u] || !m.allowedURL(u) {
			continue
		}
		seen[u] = true
		out = append(out, Result{Title: a.URLCitation.Title, URL: u, Content: msg.Content})
		if len(out) >= limit {
			break
		}
	}
	return out, nil
}

// allowlist di level search: presisi hukum naik, permukaan injeksi mengecil
func (m *MaiaSearch) prompt(query string) string {
	var sb strings.Builder
	sb.WriteString("Cari di web dan jawab ringkas berdasarkan sumber hukum Indonesia.\n")
	if len(m.allowed) > 0 {
		sb.WriteString("Utamakan sumber dari domain berikut: ")
		sb.WriteString(strings.Join(m.allowed, ", "))
		sb.WriteString(".\n")
	}
	sb.WriteString("Sebutkan dasar hukum yang relevan beserta tautan sumbernya.\n\nPertanyaan: ")
	sb.WriteString(query)
	return sb.String()
}

func (m *MaiaSearch) allowedURL(raw string) bool {
	if len(m.allowed) == 0 {
		return true
	}
	u, err := url.Parse(raw)
	if err != nil {
		return false
	}
	host := strings.ToLower(u.Hostname())
	for _, d := range m.allowed {
		if host == d || strings.HasSuffix(host, "."+d) {
			return true
		}
	}
	return false
}
