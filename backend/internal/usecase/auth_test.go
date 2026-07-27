package usecase

import (
	"context"
	"net/url"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/lexora/backend/internal/domain"
	"github.com/lexora/backend/pkg/hash"
	"github.com/lexora/backend/pkg/jwt"
	"github.com/pquerna/otp/totp"
)

type fakeRefresh struct{ m map[string]*domain.RefreshToken }

func (f *fakeRefresh) Create(_ context.Context, t *domain.RefreshToken) error {
	t.ID = uuid.New()
	f.m[t.TokenHash] = t
	return nil
}
func (f *fakeRefresh) ByHash(_ context.Context, h string) (*domain.RefreshToken, error) {
	if t, ok := f.m[h]; ok {
		return t, nil
	}
	return nil, domain.ErrNotFound
}
func (f *fakeRefresh) Revoke(_ context.Context, id uuid.UUID) error {
	for _, t := range f.m {
		if t.ID == id {
			now := time.Now()
			t.RevokedAt = &now
		}
	}
	return nil
}
func (f *fakeRefresh) RevokeByHash(_ context.Context, h string) error {
	if t, ok := f.m[h]; ok {
		now := time.Now()
		t.RevokedAt = &now
	}
	return nil
}
func (f *fakeRefresh) RevokeFamily(_ context.Context, familyID uuid.UUID) error {
	for _, t := range f.m {
		if t.FamilyID == familyID {
			now := time.Now()
			t.RevokedAt = &now
		}
	}
	return nil
}

type fakeRecovery struct{ codes []domain.RecoveryCode }

func (f *fakeRecovery) Replace(_ context.Context, userID uuid.UUID, hashes []string) error {
	f.codes = nil
	for _, h := range hashes {
		f.codes = append(f.codes, domain.RecoveryCode{ID: uuid.New(), UserID: userID, CodeHash: h})
	}
	return nil
}
func (f *fakeRecovery) Unused(_ context.Context, userID uuid.UUID) ([]domain.RecoveryCode, error) {
	var out []domain.RecoveryCode
	for _, c := range f.codes {
		if c.UserID == userID && c.UsedAt == nil {
			out = append(out, c)
		}
	}
	return out, nil
}
func (f *fakeRecovery) MarkUsed(_ context.Context, id uuid.UUID) error {
	for i := range f.codes {
		if f.codes[i].ID == id {
			now := time.Now()
			f.codes[i].UsedAt = &now
		}
	}
	return nil
}

func newAuthUC(users *fakeUsers) (*Auth, *jwt.Signer, *jwt.Signer) {
	appS := jwt.New("user-secret", time.Minute, jwt.AudienceApp)
	adminS := jwt.New("admin-secret", time.Minute, jwt.AudienceAdmin)
	uc := NewAuth(users, &fakeMembers{}, &fakeRefresh{m: map[string]*domain.RefreshToken{}}, &fakeRecovery{}, appS, adminS, time.Hour)
	return uc, appS, adminS
}

func mkUser(email, pw, role string) *domain.User {
	h, _ := hash.Password(pw)
	return &domain.User{ID: uuid.New(), Email: email, PasswordHash: h, SystemRole: role, IsActive: true}
}

// jalankan enrollment penuh, balikin token + recovery codes
func enrollAdmin(t *testing.T, uc *Auth, email, pw string) *AdminStep {
	t.Helper()
	ctx := context.Background()
	step, err := uc.AdminLogin(ctx, email, pw)
	if err != nil {
		t.Fatal(err)
	}
	if !step.EnrollRequired || step.Tokens != nil {
		t.Fatalf("login pertama harus paksa enroll tanpa token, got %+v", step)
	}
	code := totpCode(t, step.OTPAuthURL, time.Now())
	done, err := uc.AdminEnroll(ctx, email, pw, code)
	if err != nil {
		t.Fatal(err)
	}
	return done
}

func totpCode(t *testing.T, otpauth string, at time.Time) string {
	t.Helper()
	u, err := url.Parse(otpauth)
	if err != nil {
		t.Fatal(err)
	}
	code, err := totp.GenerateCode(u.Query().Get("secret"), at)
	if err != nil {
		t.Fatal(err)
	}
	return code
}

// B11: reject super_admin di app
func TestAppLoginRejectsSuperAdmin(t *testing.T) {
	su := mkUser("super@x", "pw12345678", domain.SystemRoleSuperAdmin)
	uc, _, _ := newAuthUC(&fakeUsers{m: map[uuid.UUID]*domain.User{su.ID: su}})
	if _, err := uc.Login(context.Background(), "super@x", "pw12345678"); err != domain.ErrInvalidCreds {
		t.Fatalf("super_admin harus ditolak di app login, got %v", err)
	}
}

// admin only
func TestAdminLoginRejectsNonSuperAdmin(t *testing.T) {
	u := mkUser("member@x", "pw12345678", domain.SystemRoleNone)
	uc, _, _ := newAuthUC(&fakeUsers{m: map[uuid.UUID]*domain.User{u.ID: u}})
	if _, err := uc.AdminLogin(context.Background(), "member@x", "pw12345678"); err != domain.ErrInvalidCreds {
		t.Fatalf("non-super_admin harus ditolak di admin login, got %v", err)
	}
}

// U18: tanpa enrollment tak ada token; setelah enroll, login = 2 langkah
func TestAdminEnrollmentForced(t *testing.T) {
	su := mkUser("super@x", "pw12345678", domain.SystemRoleSuperAdmin)
	uc, _, _ := newAuthUC(&fakeUsers{m: map[uuid.UUID]*domain.User{su.ID: su}})
	ctx := context.Background()

	// verify sebelum enroll -> tolak
	if _, err := uc.AdminVerify(ctx, "super@x", "pw12345678", "000000"); err != domain.ErrInvalidCreds {
		t.Fatalf("verify sebelum enroll harus ditolak, got %v", err)
	}

	done := enrollAdmin(t, uc, "super@x", "pw12345678")
	if done.Tokens == nil || done.Tokens.Access == "" {
		t.Fatal("enroll sukses harus terbitkan token")
	}
	if len(done.RecoveryCodes) != 8 {
		t.Fatalf("harus 8 recovery codes, got %d", len(done.RecoveryCodes))
	}

	// login berikutnya: bukan enroll lagi, tapi minta MFA, tetap tanpa token
	step, err := uc.AdminLogin(ctx, "super@x", "pw12345678")
	if err != nil {
		t.Fatal(err)
	}
	if !step.MFARequired || step.EnrollRequired || step.Tokens != nil {
		t.Fatalf("setelah enroll, login harus MFARequired tanpa token, got %+v", step)
	}
}

// U19 (T19): kode TOTP sekali pakai dalam window-nya
func TestTOTPReplayRejected(t *testing.T) {
	su := mkUser("super@x", "pw12345678", domain.SystemRoleSuperAdmin)
	users := &fakeUsers{m: map[uuid.UUID]*domain.User{su.ID: su}}
	uc, _, _ := newAuthUC(users)
	ctx := context.Background()

	step, err := uc.AdminLogin(ctx, "super@x", "pw12345678")
	if err != nil {
		t.Fatal(err)
	}
	code := totpCode(t, step.OTPAuthURL, time.Now())
	if _, err := uc.AdminEnroll(ctx, "super@x", "pw12345678", code); err != nil {
		t.Fatal(err)
	}

	// replay kode yang sama (step sudah terpakai saat enroll) -> tolak
	if _, err := uc.AdminVerify(ctx, "super@x", "pw12345678", code); err != domain.ErrInvalidCreds {
		t.Fatalf("replay kode dalam window harus ditolak, got %v", err)
	}

	// kode step berikutnya (toleransi skew +1) -> sah
	next := totpCode(t, step.OTPAuthURL, time.Now().Add(30*time.Second))
	if _, err := uc.AdminVerify(ctx, "super@x", "pw12345678", next); err != nil {
		t.Fatalf("kode step berikutnya harus sah, got %v", err)
	}
}

// recovery code: jalur alternatif, sekali pakai
func TestRecoveryCodeSingleUse(t *testing.T) {
	su := mkUser("super@x", "pw12345678", domain.SystemRoleSuperAdmin)
	uc, _, _ := newAuthUC(&fakeUsers{m: map[uuid.UUID]*domain.User{su.ID: su}})
	ctx := context.Background()

	done := enrollAdmin(t, uc, "super@x", "pw12345678")
	rc := done.RecoveryCodes[0]

	if _, err := uc.AdminVerify(ctx, "super@x", "pw12345678", rc); err != nil {
		t.Fatalf("recovery code harus sah, got %v", err)
	}
	if _, err := uc.AdminVerify(ctx, "super@x", "pw12345678", rc); err != domain.ErrInvalidCreds {
		t.Fatalf("recovery code bekas harus ditolak, got %v", err)
	}
}

// U20: member dinonaktifkan tidak bisa login
func TestInactiveUserCannotLogin(t *testing.T) {
	u := mkUser("m@x", "pw12345678", domain.SystemRoleNone)
	users := &fakeUsers{m: map[uuid.UUID]*domain.User{u.ID: u}}
	uc, _, _ := newAuthUC(users)
	ctx := context.Background()

	if err := users.SetActive(ctx, u.ID, false); err != nil {
		t.Fatal(err)
	}
	if _, err := uc.Login(ctx, "m@x", "pw12345678"); err != domain.ErrInactive {
		t.Fatalf("user nonaktif harus ditolak login, got %v", err)
	}
}

// token admin: aud terpisah
func TestAdminLoginIssuesAdminAudience(t *testing.T) {
	su := mkUser("super@x", "pw12345678", domain.SystemRoleSuperAdmin)
	uc, appS, adminS := newAuthUC(&fakeUsers{m: map[uuid.UUID]*domain.User{su.ID: su}})
	tok := enrollAdmin(t, uc, "super@x", "pw12345678").Tokens
	if _, err := adminS.Verify(tok.Access); err != nil {
		t.Fatalf("token admin harus lolos verifier admin: %v", err)
	}
	if _, err := appS.Verify(tok.Access); err == nil {
		t.Fatal("token admin tak boleh lolos verifier app (aud + kunci beda)")
	}
}

// reuse → cabut family
func TestRefreshReuseRevokesFamily(t *testing.T) {
	su := mkUser("s@x", "pw12345678", domain.SystemRoleSuperAdmin)
	uc, _, _ := newAuthUC(&fakeUsers{m: map[uuid.UUID]*domain.User{su.ID: su}})
	ctx := context.Background()

	tok1 := enrollAdmin(t, uc, "s@x", "pw12345678").Tokens
	tok2, err := uc.AdminRefresh(ctx, tok1.Refresh) // rotasi: tok1 revoked, tok2 hidup
	if err != nil {
		t.Fatal(err)
	}
	// pakai ulang tok1 (sudah direvoke) = theft
	if _, err := uc.AdminRefresh(ctx, tok1.Refresh); err != domain.ErrInvalidToken {
		t.Fatalf("reuse token lama harus ditolak, got %v", err)
	}
	// efeknya: tok2 yang sah pun ikut mati (family dicabut)
	if _, err := uc.AdminRefresh(ctx, tok2.Refresh); err != domain.ErrInvalidToken {
		t.Fatalf("token sah harus ikut mati setelah reuse terdeteksi, got %v", err)
	}
}
