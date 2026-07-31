package config

import (
	"bufio"
	"os"
	"strings"
	"testing"
)

// Kunci invariant: .env.example harus MEMUAT semua env yang bikin Load() gagal.
// Nangkep bug "env wajib absen dari .env.example" -> clone baru gagal start.
// (JWT_ADMIN_SECRET pernah hilang; test ini mencegah regresi.)
func TestEnvExampleSatisfiesLoad(t *testing.T) {
	f, err := os.Open("../../.env.example")
	if err != nil {
		t.Fatalf("buka .env.example: %v", err)
	}
	defer f.Close()

	// set tiap KEY=VAL dari .env.example ke environment proses test
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		k, v, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		k = strings.TrimSpace(k)
		v = strings.TrimSpace(v)
		if i := strings.Index(v, " #"); i >= 0 { // buang komentar inline
			v = strings.TrimSpace(v[:i])
		}
		t.Setenv(k, v) // auto-restore setelah test
	}
	if err := sc.Err(); err != nil {
		t.Fatal(err)
	}

	if _, err := Load(); err != nil {
		t.Fatalf(".env.example tak cukup untuk Load(): %v\n"+
			"-> ada env wajib yang belum didokumentasikan di .env.example", err)
	}
}
