package chunk

import (
	"regexp"
	"strings"

	"github.com/lexora/backend/internal/domain"
)

// defaults tuned for ~800 token / 100 overlap
// ponytail: word count approx token count; tiktoken kalau perlu presisi
const (
	targetWords  = 600 // ~800 tokens
	overlapWords = 80  // ~100 tokens
	minWords     = 15  // buang chunk terlalu pendek
)

var sentenceEnd = regexp.MustCompile(`[.!?]+\s+`)

// split per-page text into overlapping chunks, carry page number for citation
func Pages(pages []string) []domain.Chunk {
	var out []domain.Chunk
	idx := 0
	for p, text := range pages {
		page := p + 1
		for _, c := range chunkText(text) {
			pn := page
			out = append(out, domain.Chunk{Index: idx, Text: c, Page: &pn})
			idx++
		}
	}
	return out
}

// greedily pack sentences to ~targetWords with overlapWords carryover
func chunkText(text string) []string {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil
	}
	sentences := splitSentences(text)

	var chunks []string
	var cur []string
	count := 0
	flush := func() {
		if joined := strings.TrimSpace(strings.Join(cur, " ")); joined != "" {
			chunks = append(chunks, joined)
		}
		cur, count = tailWords(cur, overlapWords)
	}
	for _, s := range sentences {
		for _, piece := range splitLongSentence(s) {
			w := len(strings.Fields(piece))
			if count+w > targetWords && count > 0 {
				flush()
			}
			cur = append(cur, piece)
			count += w
		}
	}
	if joined := strings.TrimSpace(strings.Join(cur, " ")); joined != "" {
		if len(strings.Fields(joined)) >= minWords || len(chunks) == 0 {
			chunks = append(chunks, joined)
		}
	}
	return chunks
}

// hard-split a sentence longer than targetWords into word windows
func splitLongSentence(s string) []string {
	w := strings.Fields(s)
	if len(w) <= targetWords {
		return []string{s}
	}
	var out []string
	for start := 0; start < len(w); start += targetWords {
		end := start + targetWords
		if end > len(w) {
			end = len(w)
		}
		out = append(out, strings.Join(w[start:end], " "))
	}
	return out
}

func splitSentences(text string) []string {
	parts := sentenceEnd.Split(text, -1)
	var out []string
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

// return last N words as a slice of one string + its word count
func tailWords(sentences []string, n int) ([]string, int) {
	words := strings.Fields(strings.Join(sentences, " "))
	if len(words) <= n {
		return []string{strings.Join(words, " ")}, len(words)
	}
	tail := words[len(words)-n:]
	return []string{strings.Join(tail, " ")}, len(tail)
}
