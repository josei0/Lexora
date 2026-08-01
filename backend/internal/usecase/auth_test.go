package usecase

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/lexora/backend/internal/domain"
	"github.com/lexora/backend/pkg/hash"
	"github.com/lexora/backend/pkg/jwt"
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
func (f *fakeRefresh) RevokeAllForUser(_ context.Context, userID uuid.UUID) error {
	for _, t := range f.m {
		if t.UserID == userID {
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

// token admin: aud terpisah (TOTP dibuang - AdminLogin langsung terbitkan token)
func TestAdminLoginIssuesAdminAudience(t *testing.T) {
	su := mkUser("super@x", "pw12345678", domain.SystemRoleSuperAdmin)
	uc, appS, adminS := newAuthUC(&fakeUsers{m: map[uuid.UUID]*domain.User{su.ID: su}})
	step, err := uc.AdminLogin(context.Background(), "super@x", "pw12345678")
	if err != nil {
		t.Fatal(err)
	}
	tok := step.Tokens
	if tok == nil {
		t.Fatal("AdminLogin harus langsung terbitkan token (TOTP dibuang)")
	}
	if _, err := adminS.Verify(tok.Access); err != nil {
		t.Fatalf("token admin harus lolos verifier admin: %v", err)
	}
	if _, err := appS.Verify(tok.Access); err == nil {
		t.Fatal("token admin tak boleh lolos verifier app (aud + kunci beda)")
	}
}

// T2: ganti password -> refresh lama invalid (update9-S)
func TestChangePasswordRevokesRefresh(t *testing.T) {
	su := mkUser("s@x", "pw12345678", domain.SystemRoleSuperAdmin)
	uc, _, _ := newAuthUC(&fakeUsers{m: map[uuid.UUID]*domain.User{su.ID: su}})
	ctx := context.Background()

	step, err := uc.AdminLogin(ctx, "s@x", "pw12345678")
	if err != nil {
		t.Fatal(err)
	}
	if err := uc.ChangePassword(ctx, su.ID, "pw12345678", "newpw12345678"); err != nil {
		t.Fatal(err)
	}
	// refresh lama harus mati
	if _, err := uc.AdminRefresh(ctx, step.Tokens.Refresh); err != domain.ErrInvalidToken {
		t.Fatalf("refresh lama harus invalid setelah ganti password, got %v", err)
	}
}

// reuse → cabut family
func TestRefreshReuseRevokesFamily(t *testing.T) {
	su := mkUser("s@x", "pw12345678", domain.SystemRoleSuperAdmin)
	uc, _, _ := newAuthUC(&fakeUsers{m: map[uuid.UUID]*domain.User{su.ID: su}})
	ctx := context.Background()

	step, err := uc.AdminLogin(ctx, "s@x", "pw12345678")
	if err != nil {
		t.Fatal(err)
	}
	tok1 := step.Tokens
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
