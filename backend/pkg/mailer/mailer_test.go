package mailer

import (
	"os"
	"strings"
	"testing"
)

func TestBuildMessage(t *testing.T) {
	msg := string(buildMessage("me@x.com", "you@y.com", "Halo", "isi pesan"))

	for _, want := range []string{
		"From: me@x.com\r\n",
		"To: you@y.com\r\n",
		"Subject: Halo\r\n",
		"\r\n\r\n", // pemisah header/body (header terakhir + blank line)
		"isi pesan",
	} {
		if !strings.Contains(msg, want) {
			t.Errorf("pesan tak memuat %q\n---\n%s", want, msg)
		}
	}
	// body harus setelah blank line, bukan di header
	if i := strings.Index(msg, "\r\n\r\n"); i < 0 || !strings.HasSuffix(msg, "isi pesan") {
		t.Fatal("body tidak di posisi benar")
	}
}

func TestNoopWhenUnconfigured(t *testing.T) {
	m := New("", "")
	if m.Enabled() {
		t.Fatal("kredensial kosong harus no-op")
	}
	if err := m.Send("x@y.com", "s", "b"); err != nil {
		t.Fatalf("no-op Send tak boleh error: %v", err)
	}
	if New("u@gmail.com", "app-pw").Enabled() != true {
		t.Fatal("kredensial lengkap harus Enabled")
	}
}

// U1: kirim email nyata sekali. Auto-skip kalau SMTP_* belum di-set.
// Jalankan setelah App Password ada:
//   SMTP_USER=mindlaw.env@gmail.com SMTP_APP_PASSWORD=xxx MAILER_TEST_TO=mindlaw.env@gmail.com go test ./pkg/mailer -run TestSendReal -v
func TestSendReal(t *testing.T) {
	user, pw, to := os.Getenv("SMTP_USER"), os.Getenv("SMTP_APP_PASSWORD"), os.Getenv("MAILER_TEST_TO")
	if user == "" || pw == "" || to == "" {
		t.Skip("set SMTP_USER, SMTP_APP_PASSWORD, MAILER_TEST_TO untuk uji kirim nyata")
	}
	if err := New(user, pw).Send(to, "MindLaw mailer test", "Fase 13 — mailer jalan dari kode."); err != nil {
		t.Fatalf("kirim gagal: %v", err)
	}
}
