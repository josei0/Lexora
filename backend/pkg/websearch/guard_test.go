package websearch

import (
	"net"
	"strings"
	"testing"
)

func testGuard() *Guard {
	return NewGuard([]string{"peraturan.bpk.go.id", "jdihn.go.id"})
}

// U8: metadata cloud + IP internal ditolak sebelum request keluar
func TestGuardRejectsInternalTargets(t *testing.T) {
	g := testGuard()
	for _, raw := range []string{
		"http://169.254.169.254/latest/meta-data/", // metadata cloud
		"http://127.0.0.1:8080/admin",
		"http://localhost/",
		"http://10.0.0.5/",
		"http://192.168.1.1/",
	} {
		if err := g.Check(raw); err == nil {
			t.Fatalf("%s harus ditolak", raw)
		}
	}
}

func TestGuardRejectsNonAllowlistedAndBadScheme(t *testing.T) {
	g := testGuard()
	for _, raw := range []string{
		"https://situs-jahat.com/pasal",
		"https://evil-peraturan.bpk.go.id.attacker.com/", // suffix palsu
		"https://xperaturan.bpk.go.id/",                  // bukan subdomain sah
		"file:///etc/passwd",
		"gopher://peraturan.bpk.go.id/",
	} {
		if err := g.Check(raw); err == nil {
			t.Fatalf("%s harus ditolak", raw)
		}
	}
}

func TestGuardAllowsListedDomainAndSubdomain(t *testing.T) {
	g := testGuard()
	if !g.allowedHost("peraturan.bpk.go.id") {
		t.Fatal("domain persis harus lolos")
	}
	if !g.allowedHost("arsip.peraturan.bpk.go.id") {
		t.Fatal("subdomain harus lolos")
	}
	if g.allowedHost("peraturan.bpk.go.id.evil.com") {
		t.Fatal("suffix palsu tidak boleh lolos")
	}
}

func TestIsInternalCoversMappedAndULA(t *testing.T) {
	cases := map[string]bool{
		"127.0.0.1":            true,
		"::1":                  true,
		"::ffff:127.0.0.1":     true, // IPv4-mapped IPv6
		"fd00::1":              true, // unique local
		"169.254.1.1":          true,
		"8.8.8.8":              false,
		"2001:4860:4860::8888": false,
	}
	for s, want := range cases {
		if got := isInternal(net.ParseIP(s)); got != want {
			t.Fatalf("isInternal(%s) = %v, mau %v", s, got, want)
		}
	}
}

// U9: redirect ke host internal ditolak di hop kedua (guard dipanggil tiap hop)
func TestRedirectRevalidated(t *testing.T) {
	next, err := resolveRef("https://peraturan.bpk.go.id/a", "http://127.0.0.1:8080/x")
	if err != nil {
		t.Fatal(err)
	}
	if err := testGuard().Check(next); err == nil {
		t.Fatal("redirect ke loopback harus ditolak saat divalidasi ulang")
	}
	// relatif juga harus resolve ke absolut, bukan gagal diam-diam
	rel, err := resolveRef("https://peraturan.bpk.go.id/a/b", "../c")
	if err != nil || !strings.HasPrefix(rel, "https://peraturan.bpk.go.id/") {
		t.Fatalf("redirect relatif salah resolve: %q (%v)", rel, err)
	}
}

func TestHTMLToTextStripsScript(t *testing.T) {
	raw := `<html><head><title>UU 37/2004</title><style>b{}</style></head>
	<body><script>alert('x')</script><p>Pasal 222 ayat (1)</p><p>Debitor &amp; Kreditor</p></body></html>`
	txt := htmlToText(raw)
	if strings.Contains(txt, "alert") || strings.Contains(txt, "<") {
		t.Fatalf("script/tag masih tersisa: %q", txt)
	}
	if !strings.Contains(txt, "Pasal 222 ayat (1)") || !strings.Contains(txt, "Debitor & Kreditor") {
		t.Fatalf("teks hilang atau entity tak ter-decode: %q", txt)
	}
	if got := htmlTitle(raw); got != "UU 37/2004" {
		t.Fatalf("title = %q", got)
	}
}
