package websearch

import "testing"

// AL2/AL3: SetDomains ubah allowlist tanpa restart. Pakai allowedHost (tanpa DNS)
// biar hermetik; jalur Check+DNS sudah diuji guard_test.go.
func TestGuardSetDomainsTogglesAllowlist(t *testing.T) {
	g := NewGuard(nil)
	if g.allowedHost("jdihn.go.id") {
		t.Fatal("allowlist kosong harus tolak semua")
	}
	// AL2: tambah -> lolos (persis + subdomain)
	g.SetDomains([]string{" JDIHN.go.id "}) // trim+lower
	if !g.allowedHost("jdihn.go.id") || !g.allowedHost("sub.jdihn.go.id") {
		t.Fatal("domain baru harus lolos setelah SetDomains")
	}
	// AL3: hapus (set tanpa host itu) -> ditolak
	g.SetDomains([]string{"lain.go.id"})
	if g.allowedHost("jdihn.go.id") {
		t.Fatal("domain yang dicabut harus ditolak")
	}
}

// search allowlist ikut berubah: prompt + allowedURL baca snapshot terbaru
func TestSearchSetDomainsTogglesFilter(t *testing.T) {
	m := NewMaiaSearch("", "", "model", nil)
	if !m.allowedURL("https://apa-saja.com/x") {
		t.Fatal("allowlist kosong = terima semua (fallback)")
	}
	m.SetDomains([]string{"jdihn.go.id"})
	if !m.allowedURL("https://jdihn.go.id/uu") {
		t.Fatal("URL dari domain allowlist harus lolos")
	}
	if m.allowedURL("https://jahat.com/x") {
		t.Fatal("URL di luar allowlist harus ditolak")
	}
}
