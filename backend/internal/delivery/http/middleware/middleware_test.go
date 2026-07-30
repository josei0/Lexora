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

// CORS bukan security boundary (otorisasi via JWT audience + role). Kedua origin
// first-party (app + admin) diizinkan di semua path; origin asing ditolak.
func TestCORSAllowsFirstPartyOrigins(t *testing.T) {
	h := CORS([]string{"https://mindlaw.web.id"}, []string{"https://admin.lvh.me"})(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))

	allow := func(origin, path string) string {
		req := httptest.NewRequest(http.MethodOptions, path, nil)
		req.Header.Set("Origin", origin)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		return rec.Header().Get("Access-Control-Allow-Origin")
	}

	// kedua origin first-party lolos di path mana pun
	if got := allow("https://admin.lvh.me", "/auth/admin/login"); got != "https://admin.lvh.me" {
		t.Fatalf("admin origin harus diizinkan, dapat %q", got)
	}
	if got := allow("https://mindlaw.web.id", "/auth/login"); got != "https://mindlaw.web.id" {
		t.Fatalf("app origin harus diizinkan, dapat %q", got)
	}
	// preflight di-cache
	req := httptest.NewRequest(http.MethodOptions, "/auth/login", nil)
	req.Header.Set("Origin", "https://mindlaw.web.id")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Header().Get("Access-Control-Max-Age") != "600" {
		t.Fatal("preflight tak di-cache (Max-Age hilang)")
	}
	// origin asing DITOLAK
	if got := allow("https://evil.example.com", "/auth/login"); got != "" {
		t.Fatalf("origin asing harus ditolak, dapat %q", got)
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
