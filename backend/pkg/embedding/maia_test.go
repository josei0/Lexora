package embedding

import (
	"context"
	"os"
	"testing"
)

// live test - only runs when MAIA_API_KEY is set
func TestMaiaEmbedLive(t *testing.T) {
	key := os.Getenv("MAIA_API_KEY")
	if key == "" {
		t.Skip("MAIA_API_KEY not set")
	}
	url := os.Getenv("EMBEDDING_URL")
	if url == "" {
		url = "https://api.maiarouter.ai/v1"
	}
	m := NewMaia(url, key, "openai/text-embedding-3-large", 3072)
	vecs, err := m.Embed(context.Background(), []string{"pasal satu ketentuan umum", "sanksi administratif"})
	if err != nil {
		t.Fatal(err)
	}
	if len(vecs) != 2 {
		t.Fatalf("want 2 vectors, got %d", len(vecs))
	}
	if len(vecs[0]) != 3072 {
		t.Fatalf("want dim 3072, got %d", len(vecs[0]))
	}
}
