package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/lexora/backend/internal/domain"
	"github.com/lexora/backend/pkg/jwt"
)

// fake verifier: balik klaim tetap kalau token == "good", selain itu error (aud salah dll)
type fakeGoogle struct {
	claims  *GoogleClaims
	enabled bool
}

func (f *fakeGoogle) Enabled() bool { return f.enabled }
func (f *fakeGoogle) Verify(_ context.Context, idToken string) (*GoogleClaims, error) {
	if idToken != "good" || f.claims == nil {
		return nil, errors.New("aud/token tidak valid")
	}
	return f.claims, nil
}

// rakit Auth + Organization (registrar) berbagi repo yang sama
func newGoogleUC(users *fakeUsers, members *fakeMembers, claims *GoogleClaims) *Auth {
	appS := jwt.New("user-secret", time.Minute, jwt.AudienceApp)
	adminS := jwt.New("admin-secret", time.Minute, jwt.AudienceAdmin)
	auth := NewAuth(users, members, &fakeRefresh{m: map[string]*domain.RefreshToken{}}, &fakeRecovery{}, appS, adminS, time.Hour)
	org := NewOrganization(&fakeOrgs{}, users, members)
	auth.SetGoogle(&fakeGoogle{claims: claims, enabled: true}, org)
	return auth
}

// U14: login Google user baru -> auto-register aktif langsung + org bernama = nama orang
func TestLoginGoogleAutoRegister(t *testing.T) {
	users := &fakeUsers{m: map[uuid.UUID]*domain.User{}}
	members := &fakeMembers{}
	auth := newGoogleUC(users, members, &GoogleClaims{Sub: "g-123", Email: "budi@gmail.com", Name: "Budi Santoso"})

	tok, err := auth.LoginGoogle(context.Background(), "good")
	if err != nil {
		t.Fatalf("auto-register harus sukses: %v", err)
	}
	if tok.Access == "" {
		t.Fatal("harus balik token")
	}
	var u *domain.User
	for _, x := range users.m {
		u = x
	}
	if u == nil || !u.IsActive || u.EmailVerifiedAt == nil {
		t.Fatal("akun Google harus aktif langsung + verified")
	}
	if u.GoogleSub == nil || *u.GoogleSub != "g-123" {
		t.Fatal("google_sub harus tersimpan")
	}
}

// U15: email existing tanpa google_sub -> 409 (ErrGoogleUnlinked)
func TestLoginGoogleExistingUnlinked(t *testing.T) {
	uid := uuid.New()
	existing := &domain.User{ID: uid, Email: "ada@gmail.com", IsActive: true} // GoogleSub nil
	users := &fakeUsers{m: map[uuid.UUID]*domain.User{uid: existing}}
	auth := newGoogleUC(users, &fakeMembers{}, &GoogleClaims{Sub: "g-9", Email: "ada@gmail.com", Name: "Ada"})

	_, err := auth.LoginGoogle(context.Background(), "good")
	if err != domain.ErrGoogleUnlinked {
		t.Fatalf("email existing tanpa link Google harus ErrGoogleUnlinked, got %v", err)
	}
}

// login Google user existing yang SUDAH link -> login sukses (bukan register ulang)
func TestLoginGoogleExistingLinked(t *testing.T) {
	uid := uuid.New()
	sub := "g-5"
	existing := &domain.User{ID: uid, Email: "linked@gmail.com", IsActive: true, GoogleSub: &sub}
	users := &fakeUsers{m: map[uuid.UUID]*domain.User{uid: existing}}
	members := &fakeMembers{list: []domain.Membership{{UserID: uid, OrganizationID: uuid.New(), Role: domain.OrgRoleAdmin}}}
	auth := newGoogleUC(users, members, &GoogleClaims{Sub: sub, Email: "linked@gmail.com", Name: "L"})

	tok, err := auth.LoginGoogle(context.Background(), "good")
	if err != nil || tok.Access == "" {
		t.Fatalf("login user ter-link harus sukses: %v", err)
	}
	if len(users.m) != 1 {
		t.Fatal("tidak boleh bikin user baru untuk email existing")
	}
}

// U16: aud/token salah -> tolak
func TestLoginGoogleBadToken(t *testing.T) {
	users := &fakeUsers{m: map[uuid.UUID]*domain.User{}}
	auth := newGoogleUC(users, &fakeMembers{}, &GoogleClaims{Sub: "x", Email: "x@x.com"})

	if _, err := auth.LoginGoogle(context.Background(), "bad-token"); err != domain.ErrInvalidToken {
		t.Fatalf("token salah harus ErrInvalidToken, got %v", err)
	}
}

// U17: name kosong -> nama firma fallback ke bagian lokal email
func TestFirmaFromGoogleFallback(t *testing.T) {
	if got := firmaFromGoogle("", "budi.s@law.id"); got != "budi.s" {
		t.Errorf("name kosong harus fallback local-part, got %q", got)
	}
	if got := firmaFromGoogle("  Budi  ", "x@y.com"); got != "Budi" {
		t.Errorf("name terisi harus dipakai (trimmed), got %q", got)
	}
}
