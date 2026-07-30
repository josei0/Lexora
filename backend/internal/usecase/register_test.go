package usecase

import (
	"context"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/lexora/backend/internal/domain"
)

// fake mailer: rekam email terkirim + selalu Enabled (paksa jalur verifikasi)
type fakeMailer struct {
	sent []struct{ to, subject, body string }
}

func (f *fakeMailer) Enabled() bool { return true }
func (f *fakeMailer) Send(to, subject, body string) error {
	f.sent = append(f.sent, struct{ to, subject, body string }{to, subject, body})
	return nil
}

func newRegisterUC(mailer Mailer) (*Organization, *fakeUsers) {
	users := &fakeUsers{m: map[uuid.UUID]*domain.User{}}
	uc := NewOrganization(&fakeOrgs{}, users, &fakeMembers{})
	if mailer != nil {
		uc.SetMailer(mailer, "https://app.test")
	}
	return uc, users
}

// U4: register bikin org+admin nonaktif + kirim email verif
func TestRegisterCreatesInactiveAdmin(t *testing.T) {
	mailer := &fakeMailer{}
	uc, _ := newRegisterUC(mailer)

	u, err := uc.Register(context.Background(), "Budi & Rekan", "Budi", "budi@law.id", "password8", false, nil)
	if err != nil {
		t.Fatal(err)
	}
	if u.IsActive {
		t.Error("akun register harus nonaktif sampai verifikasi")
	}
	if u.MustChangePassword {
		t.Error("password dari user, mustChange harus false")
	}
	if u.VerifyTokenHash == nil {
		t.Error("token verif harus tersimpan")
	}
	if len(mailer.sent) != 1 || mailer.sent[0].to != "budi@law.id" {
		t.Fatalf("email verif harus terkirim ke pendaftar, sent=%+v", mailer.sent)
	}
	if !strings.Contains(mailer.sent[0].body, "https://app.test/verify-email?token=") {
		t.Errorf("email harus memuat link verif, body=%q", mailer.sent[0].body)
	}
}

// U5: verif token benar -> akun aktif
func TestVerifyEmailActivates(t *testing.T) {
	mailer := &fakeMailer{}
	uc, users := newRegisterUC(mailer)
	if _, err := uc.Register(context.Background(), "Firma", "A", "a@x.id", "password8", false, nil); err != nil {
		t.Fatal(err)
	}
	// ambil raw token dari link email (satu-satunya cara user dapat token)
	body := mailer.sent[0].body
	raw := body[strings.Index(body, "token=")+len("token=") : strings.Index(body, "\n\nTautan")]

	if err := uc.VerifyEmail(context.Background(), raw); err != nil {
		t.Fatalf("verif token benar harus sukses: %v", err)
	}
	for _, u := range users.m {
		if !u.IsActive || u.EmailVerifiedAt == nil {
			t.Error("akun harus aktif + email_verified_at terisi setelah verif")
		}
		if u.VerifyTokenHash != nil {
			t.Error("token harus hangus setelah dipakai")
		}
	}
}

// U6: token salah -> tolak
func TestVerifyEmailBadToken(t *testing.T) {
	uc, _ := newRegisterUC(&fakeMailer{})
	if _, err := uc.Register(context.Background(), "F", "A", "a@x.id", "password8", false, nil); err != nil {
		t.Fatal(err)
	}
	if err := uc.VerifyEmail(context.Background(), "token-ngawur"); err != domain.ErrInvalidToken {
		t.Fatalf("token salah harus ErrInvalidToken, got %v", err)
	}
}

// mailer nil (dev) -> akun langsung aktif, tanpa token
func TestRegisterNoMailerActivatesImmediately(t *testing.T) {
	uc, _ := newRegisterUC(nil)
	u, err := uc.Register(context.Background(), "F", "A", "a@x.id", "password8", false, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !u.IsActive || u.VerifyTokenHash != nil {
		t.Error("tanpa mailer, akun harus aktif langsung tanpa token")
	}
}
