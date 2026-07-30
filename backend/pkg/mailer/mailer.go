package mailer

import (
	"log/slog"
	"net/smtp"
	"strings"
	"sync"
)

// Gmail SMTP + App Password (net/smtp, nol dependency).
// Kredensial kosong = no-op: akun tetap jalan tanpa email (dev), warn sekali.
type Mailer struct {
	from     string
	password string
	host     string // smtp.gmail.com
	addr     string // smtp.gmail.com:587
	noop     bool
	warnOnce sync.Once
}

// New: user/appPassword kosong -> mode no-op (dev tanpa SMTP).
func New(user, appPassword string) *Mailer {
	const host = "smtp.gmail.com"
	return &Mailer{
		from:     user,
		password: appPassword,
		host:     host,
		addr:     host + ":587",
		noop:     user == "" || appPassword == "",
	}
}

// Enabled: true kalau kredensial lengkap. Pemanggil pakai ini untuk memutuskan
// apakah alur butuh-email (mis. verifikasi register) aktif atau di-skip.
func (m *Mailer) Enabled() bool { return !m.noop }

// Send: kirim 1 email teks. No-op (log warn sekali) kalau kredensial kosong.
func (m *Mailer) Send(to, subject, body string) error {
	if m.noop {
		m.warnOnce.Do(func() {
			slog.Warn("mailer no-op: SMTP_USER/SMTP_APP_PASSWORD kosong, email tidak dikirim")
		})
		return nil
	}
	auth := smtp.PlainAuth("", m.from, m.password, m.host)
	return smtp.SendMail(m.addr, auth, m.from, []string{to}, buildMessage(m.from, to, subject, body))
}

// buildMessage: rakit header RFC 5322 minimal + body. Dipisah agar testable
// tanpa koneksi SMTP. CRLF wajib antar-header.
func buildMessage(from, to, subject, body string) []byte {
	var b strings.Builder
	b.WriteString("From: " + from + "\r\n")
	b.WriteString("To: " + to + "\r\n")
	b.WriteString("Subject: " + subject + "\r\n")
	b.WriteString("MIME-Version: 1.0\r\n")
	b.WriteString("Content-Type: text/plain; charset=\"utf-8\"\r\n")
	b.WriteString("\r\n") // pemisah header/body
	b.WriteString(body)
	return []byte(b.String())
}

// helper opsional untuk pemanggil (link verifikasi dll)
func (m *Mailer) From() string { return m.from }
