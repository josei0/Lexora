package llm

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/lexora/backend/internal/domain"
)

// Maia Router chat client (OpenAI-compatible /v1/chat/completions)
type Maia struct {
	baseURL string
	apiKey  string
	model   string
	http    *http.Client
}

func NewMaia(baseURL, apiKey, model string) *Maia {
	return &Maia{
		baseURL: baseURL,
		apiKey:  apiKey,
		model:   model,
		http:    &http.Client{Timeout: 5 * time.Minute},
	}
}

func (m *Maia) Model() string { return m.model }

type chatReq struct {
	Model         string        `json:"model"`
	Messages      []chatMsg     `json:"messages"`
	Stream        bool          `json:"stream"`
	StreamOptions streamOptions `json:"stream_options"`
	MaxTokens     int           `json:"max_tokens"`
}

type streamOptions struct {
	IncludeUsage bool `json:"include_usage"`
}

type chatMsg struct {
	Role    string `json:"role"`
	Content any    `json:"content"` // string, atau []part untuk pesan bergambar
}

// OpenAI multimodal: content jadi array {type:text|image_url}
func msgContent(m domain.ChatMessage) any {
	if len(m.Images) == 0 {
		return m.Content
	}
	parts := []any{map[string]any{"type": "text", "text": m.Content}}
	for _, url := range m.Images {
		parts = append(parts, map[string]any{"type": "image_url", "image_url": map[string]any{"url": url}})
	}
	return parts
}

type chatChunk struct {
	Choices []struct {
		Delta struct {
			Content string `json:"content"`
		} `json:"delta"`
	} `json:"choices"`
	Usage *struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
	} `json:"usage"`
}

func (m *Maia) Stream(ctx context.Context, system string, msgs []domain.ChatMessage, onToken func(string)) (domain.LLMUsage, error) {
	var usage domain.LLMUsage

	all := make([]chatMsg, 0, len(msgs)+1)
	if system != "" {
		all = append(all, chatMsg{Role: "system", Content: system})
	}
	for _, msg := range msgs {
		all = append(all, chatMsg{Role: msg.Role, Content: msgContent(msg)})
	}

	body, err := json.Marshal(chatReq{
		Model:         m.model,
		Messages:      all,
		Stream:        true,
		StreamOptions: streamOptions{IncludeUsage: true},
		MaxTokens:     4096,
	})
	if err != nil {
		return usage, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, m.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return usage, err
	}
	req.Header.Set("Authorization", "Bearer "+m.apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")

	resp, err := m.http.Do(req)
	if err != nil {
		return usage, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		var buf bytes.Buffer
		buf.ReadFrom(resp.Body)
		return usage, fmt.Errorf("llm %d: %s", resp.StatusCode, buf.String())
	}

	sc := bufio.NewScanner(resp.Body)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		payload := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if payload == "[DONE]" {
			break
		}
		var chunk chatChunk
		if err := json.Unmarshal([]byte(payload), &chunk); err != nil {
			continue
		}
		for _, c := range chunk.Choices {
			if c.Delta.Content != "" {
				onToken(c.Delta.Content)
			}
		}
		if chunk.Usage != nil {
			usage.InputTokens = chunk.Usage.PromptTokens
			usage.OutputTokens = chunk.Usage.CompletionTokens
		}
	}
	return usage, sc.Err()
}
