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

// T1: HSTS + CSP terpasang di respons API (update9-S)
func TestSecureHeadersHSTSandCSP(t *testing.T) {
	h := SecureHeaders(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	req := httptest.NewRequest(http.MethodGet, "/anything", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if hsts := rec.Header().Get("Strict-Transport-Security"); hsts == "" {
		t.Fatal("HSTS hilang")
	} else if !bytesContains(hsts, "max-age=") || !bytesContains(hsts, "includeSubDomains") {
		t.Fatalf("HSTS lemah: %q", hsts)
	}
	if csp := rec.Header().Get("Content-Security-Policy"); csp == "" {
		t.Fatal("CSP hilang")
	} else if !bytesContains(csp, "default-src 'none'") {
		t.Fatalf("CSP API harus kunci default-src: %q", csp)
	}
}

func bytesContains(s, sub string) bool { return bytes.Contains([]byte(s), []byte(sub)) }

// T4/T5: host-check admin (update9-S)
func TestAdminHostOnly(t *testing.T) {
	ok := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })

	// ADMIN_API_HOST diset -> rute admin cek Host
	h := AdminHostOnly("admin-api.mindlaw.web.id")(ok)
	code := func(host, path string) int {
		req := httptest.NewRequest(http.MethodPost, path, nil)
		req.Host = host
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		return rec.Code
	}
	// T4: Host benar -> lolos
	if got := code("admin-api.mindlaw.web.id", "/admin/organizations/x/members"); got != http.StatusOK {
		t.Fatalf("Host benar harus lolos, got %d", got)
	}
	if got := code("admin-api.mindlaw.web.id:443", "/auth/admin/login"); got != http.StatusOK {
		t.Fatalf("Host benar (dgn port) harus lolos, got %d", got)
	}
	// T5: Host asing ke rute admin -> 404
	if got := code("api.mindlaw.web.id", "/admin/organizations/x/members"); got != http.StatusNotFound {
		t.Fatalf("Host asing ke /admin harus ditolak, got %d", got)
	}
	if got := code("api.mindlaw.web.id", "/auth/admin/login"); got != http.StatusNotFound {
		t.Fatalf("Host asing ke /auth/admin harus ditolak, got %d", got)
	}
	// rute non-admin tak terpengaruh
	if got := code("api.mindlaw.web.id", "/auth/login"); got != http.StatusOK {
		t.Fatalf("rute non-admin harus lolos di host mana pun, got %d", got)
	}

	// env kosong = nonaktif (dev): Host asing pun lolos
	dev := AdminHostOnly("")(ok)
	req := httptest.NewRequest(http.MethodPost, "/admin/x", nil)
	req.Host = "evil.example.com"
	rec := httptest.NewRecorder()
	dev.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("ADMIN_API_HOST kosong harus bypass, got %d", rec.Code)
	}
}

// T6: urutan chain - SecureHeaders di luar AdminHostOnly, jadi respons 404
// (Host asing ke /admin) TETAP bawa HSTS+CSP. Cegah regresi urutan middleware.
func TestSecureHeadersWrapsHostBlock(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })
	// tiru urutan router: SecureHeaders(AdminHostOnly(inner))
	chain := SecureHeaders(AdminHostOnly("admin-api.mindlaw.web.id")(inner))

	req := httptest.NewRequest(http.MethodPost, "/admin/organizations/x/members", nil)
	req.Host = "api.mindlaw.web.id" // host salah -> harus 404
	rec := httptest.NewRecorder()
	chain.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("Host asing ke /admin harus 404, got %d", rec.Code)
	}
	if rec.Header().Get("Strict-Transport-Security") == "" {
		t.Fatal("respons 404 tetap harus bawa HSTS (SecureHeaders di luar)")
	}
	if rec.Header().Get("Content-Security-Policy") == "" {
		t.Fatal("respons 404 tetap harus bawa CSP")
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
