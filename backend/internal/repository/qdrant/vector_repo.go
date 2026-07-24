package qdrant

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/lexora/backend/internal/domain"
)

// qdrant REST client (no sdk - few endpoints)
// ponytail: REST langsung, cukup buat upsert/search/ensure
type Repo struct {
	baseURL string
	http    *http.Client
}

func New(baseURL string) *Repo {
	return &Repo{baseURL: baseURL, http: &http.Client{Timeout: 30 * time.Second}}
}

func (r *Repo) do(ctx context.Context, method, path string, body any, out any) error {
	var rdr *bytes.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return err
		}
		rdr = bytes.NewReader(b)
	} else {
		rdr = bytes.NewReader(nil)
	}
	req, err := http.NewRequestWithContext(ctx, method, r.baseURL+path, rdr)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := r.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		var buf bytes.Buffer
		buf.ReadFrom(resp.Body)
		return fmt.Errorf("qdrant %s %s: %d %s", method, path, resp.StatusCode, buf.String())
	}
	if out != nil {
		return json.NewDecoder(resp.Body).Decode(out)
	}
	return nil
}

// idempotent: PUT collection with vector size. already-exists = ok
func (r *Repo) EnsureCollection(ctx context.Context, name string, dim int) error {
	var probe struct {
		Status string `json:"status"`
	}
	err := r.do(ctx, http.MethodGet, "/collections/"+name, nil, &probe)
	if err == nil {
		return nil // exists
	}
	body := map[string]any{
		"vectors": map[string]any{"size": dim, "distance": "Cosine"},
	}
	return r.do(ctx, http.MethodPut, "/collections/"+name, body, nil)
}

func (r *Repo) Upsert(ctx context.Context, collection string, points []domain.VectorPoint) error {
	if len(points) == 0 {
		return nil
	}
	pts := make([]map[string]any, len(points))
	for i, p := range points {
		pts[i] = map[string]any{"id": p.ID, "vector": p.Vector, "payload": p.Payload}
	}
	body := map[string]any{"points": pts}
	return r.do(ctx, http.MethodPut, "/collections/"+collection+"/points?wait=true", body, nil)
}

func (r *Repo) Search(ctx context.Context, collection string, vector []float32, topK int, filter map[string]any) ([]domain.SearchHit, error) {
	body := map[string]any{
		"vector":       vector,
		"limit":        topK,
		"with_payload": true,
	}
	if len(filter) > 0 {
		var must []map[string]any
		for k, v := range filter {
			must = append(must, map[string]any{"key": k, "match": map[string]any{"value": v}})
		}
		body["filter"] = map[string]any{"must": must}
	}
	var resp struct {
		Result []struct {
			ID      any            `json:"id"`
			Score   float32        `json:"score"`
			Payload map[string]any `json:"payload"`
		} `json:"result"`
	}
	if err := r.do(ctx, http.MethodPost, "/collections/"+collection+"/points/search", body, &resp); err != nil {
		return nil, err
	}
	hits := make([]domain.SearchHit, len(resp.Result))
	for i, h := range resp.Result {
		hits[i] = domain.SearchHit{ID: fmt.Sprint(h.ID), Score: h.Score, Payload: h.Payload}
	}
	return hits, nil
}

func (r *Repo) DeleteByDocument(ctx context.Context, collection, documentID string) error {
	body := map[string]any{
		"filter": map[string]any{
			"must": []map[string]any{
				{"key": "document_id", "match": map[string]any{"value": documentID}},
			},
		},
	}
	return r.do(ctx, http.MethodPost, "/collections/"+collection+"/points/delete?wait=true", body, nil)
}
