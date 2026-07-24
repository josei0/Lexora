package middleware

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRateLimitAdminLoginPerIP(t *testing.T) {
	h := RateLimitLogin(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	var last int
	for i := 0; i < 7; i++ { // limit 5/min → ke-6 dst = 429
		req := httptest.NewRequest(http.MethodPost, "/auth/admin/login", nil)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		last = rec.Code
	}
	if last != http.StatusTooManyRequests {
		t.Fatalf("admin login harus di-throttle per-IP, dapat %d", last)
	}
}

func TestCORSPerGroup(t *testing.T) {
	h := CORS([]string{"https://lexora.com"}, []string{"https://admin.lvh.me"})(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))

	// admin origin ok
	req := httptest.NewRequest(http.MethodOptions, "/auth/admin/login", nil)
	req.Header.Set("Origin", "https://admin.lvh.me")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Header().Get("Access-Control-Allow-Origin") != "https://admin.lvh.me" {
		t.Fatal("admin origin harus diizinkan di /auth/admin/*")
	}
	if rec.Header().Get("Access-Control-Max-Age") != "600" {
		t.Fatal("preflight tak di-cache (Max-Age hilang)")
	}

	// app origin ditolak
	req = httptest.NewRequest(http.MethodOptions, "/auth/admin/login", nil)
	req.Header.Set("Origin", "https://lexora.com")
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Fatal("app origin bocor ke endpoint admin")
	}
}

func TestBodyLimitMultipartGetsBiggerBudget(t *testing.T) {
	var got int
	h := BodyLimit(1<<10, 1<<20)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		got = len(b)
	}))

	body := bytes.Repeat([]byte("x"), 4<<10)

	req := httptest.NewRequest(http.MethodPost, "/documents", bytes.NewReader(body))
	req.Header.Set("Content-Type", "multipart/form-data; boundary=abc")
	h.ServeHTTP(httptest.NewRecorder(), req)
	if got != len(body) {
		t.Fatalf("multipart terpotong: %d dari %d byte", got, len(body))
	}

	req = httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	h.ServeHTTP(httptest.NewRecorder(), req)
	if got == len(body) {
		t.Fatalf("json body tidak dibatasi: %d byte lolos", got)
	}
}
