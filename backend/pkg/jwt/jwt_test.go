package jwt

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/lexora/backend/internal/domain"
)

func TestSignVerifyRoundtrip(t *testing.T) {
	s := New("test-secret-please-change", 15*time.Minute, AudienceApp)
	want := domain.Identity{
		UserID:     uuid.New(),
		OrgID:      uuid.New(),
		SystemRole: domain.SystemRoleNone,
		OrgRole:    domain.OrgRoleAdmin,
	}
	tok, err := s.Sign(want)
	if err != nil {
		t.Fatal(err)
	}
	got, err := s.Verify(tok)
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("roundtrip mismatch: %+v != %+v", got, want)
	}
}

func TestVerifyRejectsWrongSecret(t *testing.T) {
	tok, _ := New("secret-a", time.Minute, AudienceApp).Sign(domain.Identity{UserID: uuid.New()})
	if _, err := New("secret-b", time.Minute, AudienceApp).Verify(tok); err != domain.ErrInvalidToken {
		t.Fatalf("expected ErrInvalidToken, got %v", err)
	}
}

func TestVerifyRejectsExpired(t *testing.T) {
	tok, _ := New("secret", -time.Minute, AudienceApp).Sign(domain.Identity{UserID: uuid.New()})
	if _, err := New("secret", time.Minute, AudienceApp).Verify(tok); err != domain.ErrInvalidToken {
		t.Fatalf("expected ErrInvalidToken for expired, got %v", err)
	}
}

// app token gagal di verifier admin
func TestVerifyRejectsWrongAudience(t *testing.T) {
	tok, _ := New("shared", time.Minute, AudienceApp).Sign(domain.Identity{UserID: uuid.New()})
	if _, err := New("shared", time.Minute, AudienceAdmin).Verify(tok); err != domain.ErrInvalidToken {
		t.Fatalf("app token must fail admin verify, got %v", err)
	}
}
