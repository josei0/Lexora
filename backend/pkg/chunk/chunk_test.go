package chunk

import (
	"strings"
	"testing"
)

// build a page with ~n words
func words(n int) string {
	return strings.TrimSpace(strings.Repeat("kata ", n))
}

func TestPagesEmpty(t *testing.T) {
	if got := Pages([]string{"", "   "}); len(got) != 0 {
		t.Fatalf("empty pages must yield 0 chunks, got %d", len(got))
	}
}

func TestPagePreservesPageNumber(t *testing.T) {
	chunks := Pages([]string{"Halaman satu berisi teks hukum yang cukup untuk satu chunk.", words(700)})
	if len(chunks) < 2 {
		t.Fatalf("expected multiple chunks, got %d", len(chunks))
	}
	// first chunk from page 1
	if chunks[0].Page == nil || *chunks[0].Page != 1 {
		t.Fatalf("first chunk page = %v, want 1", chunks[0].Page)
	}
	// a large page 2 must split into >1 chunk and stay on page 2
	page2 := 0
	for _, c := range chunks {
		if c.Page != nil && *c.Page == 2 {
			page2++
		}
	}
	if page2 < 2 {
		t.Fatalf("700-word page should split into >=2 chunks, got %d", page2)
	}
}

func TestChunkIndexMonotonic(t *testing.T) {
	chunks := Pages([]string{words(700), words(700)})
	for i, c := range chunks {
		if c.Index != i {
			t.Fatalf("chunk index gap at %d: got %d", i, c.Index)
		}
	}
}

func TestOverlapBetweenChunks(t *testing.T) {
	// long single-sentence-ish page forces splitting with carryover
	chunks := chunkText(words(1500))
	if len(chunks) < 2 {
		t.Fatalf("expected split, got %d", len(chunks))
	}
	// each chunk should be under a hard ceiling (target + overlap slack)
	for _, c := range chunks {
		if wc := len(strings.Fields(c)); wc > targetWords+200 {
			t.Fatalf("chunk too big: %d words", wc)
		}
	}
}
