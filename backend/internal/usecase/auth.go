package usecase

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/lexora/backend/internal/domain"
	"github.com/lexora/backend/pkg/hash"
	"github.com/lexora/backend/pkg/jwt"
)

// auth usecase
type Auth struct {
	users      domain.UserRepository
	members    domain.MembershipRepository
	refresh    domain.RefreshTokenRepository
	signer     *jwt.Signer
	refreshTTL time.Duration
	loginLimit *acctLimiter
}

func NewAuth(u domain.UserRepository, m domain.MembershipRepository, r domain.RefreshTokenRepository, s *jwt.Signer, refreshTTL time.Duration) *Auth {
	return &Auth{u, m, r, s, refreshTTL, newAcctLimiter(10, time.Minute)}
}

type Tokens struct {
	Access             string
	Refresh            string // raw token for cookie
	ExpiresIn          int    // access ttl seconds
	MustChangePassword bool
}

const accessTTLSeconds = 15 * 60

// login: verify creds, issue tokens
func (a *Auth) Login(ctx context.Context, email, password string) (*Tokens, error) {
	// per-account rate limit
	if !a.loginLimit.allow(email) {
		return nil, domain.ErrRateLimited
	}
	u, err := a.users.ByEmail(ctx, email)
	if err != nil {
		// generic - no user enumeration
		_ = hash.Verify(password, dummyHash) // constant-time-ish
		return nil, domain.ErrInvalidCreds
	}
	if err := hash.Verify(password, u.PasswordHash); err != nil {
		return nil, domain.ErrInvalidCreds
	}
	if !u.IsActive {
		return nil, domain.ErrInactive
	}
	return a.issue(ctx, u)
}

// issue access + refresh
func (a *Auth) issue(ctx context.Context, u *domain.User) (*Tokens, error) {
	id := domain.Identity{UserID: u.ID, SystemRole: u.SystemRole}
	if u.SystemRole != domain.SystemRoleSuperAdmin {
		m, err := a.members.Primary(ctx, u.ID)
		if err != nil {
			return nil, err
		}
		id.OrgID = m.OrganizationID
		id.OrgRole = m.Role
	}

	access, err := a.signer.Sign(id)
	if err != nil {
		return nil, err
	}

	raw, err := randToken()
	if err != nil {
		return nil, err
	}
	rt := &domain.RefreshToken{UserID: u.ID, TokenHash: hashToken(raw), ExpiresAt: time.Now().Add(a.refreshTTL)}
	if err := a.refresh.Create(ctx, rt); err != nil {
		return nil, err
	}
	return &Tokens{Access: access, Refresh: raw, ExpiresIn: accessTTLSeconds, MustChangePassword: u.MustChangePassword}, nil
}

// refresh: rotate token
func (a *Auth) Refresh(ctx context.Context, raw string) (*Tokens, error) {
	rt, err := a.refresh.ByHash(ctx, hashToken(raw))
	if err != nil {
		return nil, domain.ErrInvalidToken
	}
	if rt.RevokedAt != nil || time.Now().After(rt.ExpiresAt) {
		return nil, domain.ErrInvalidToken
	}
	u, err := a.users.ByID(ctx, rt.UserID)
	if err != nil {
		return nil, domain.ErrInvalidToken
	}
	if !u.IsActive {
		return nil, domain.ErrInactive
	}
	// rotate: revoke old, issue new
	if err := a.refresh.Revoke(ctx, rt.ID); err != nil {
		return nil, err
	}
	return a.issue(ctx, u)
}

// logout: revoke refresh
func (a *Auth) Logout(ctx context.Context, raw string) error {
	if raw == "" {
		return nil
	}
	return a.refresh.RevokeByHash(ctx, hashToken(raw))
}

// change own password
func (a *Auth) ChangePassword(ctx context.Context, userID uuid.UUID, current, next string) error {
	u, err := a.users.ByID(ctx, userID)
	if err != nil {
		return err
	}
	if err := hash.Verify(current, u.PasswordHash); err != nil {
		return domain.ErrWrongPassword
	}
	h, err := hash.Password(next)
	if err != nil {
		return err
	}
	return a.users.UpdatePassword(ctx, userID, h)
}

func randToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func hashToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

// bogus hash for constant-time login on missing user
var dummyHash, _ = hash.Password("dummy-password-for-timing")

// per-account sliding-window limiter
// ponytail: in-memory, pindah Redis kalau multi-instance
type acctLimiter struct {
	mu     sync.Mutex
	hits   map[string][]time.Time
	limit  int
	window time.Duration
}

func newAcctLimiter(limit int, window time.Duration) *acctLimiter {
	return &acctLimiter{hits: map[string][]time.Time{}, limit: limit, window: window}
}

func (l *acctLimiter) allow(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	cutoff := time.Now().Add(-l.window)
	kept := l.hits[key][:0]
	for _, t := range l.hits[key] {
		if t.After(cutoff) {
			kept = append(kept, t)
		}
	}
	if len(kept) >= l.limit {
		l.hits[key] = kept
		return false
	}
	l.hits[key] = append(kept, time.Now())
	return true
}
