package usecase

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/lexora/backend/internal/domain"
	"github.com/lexora/backend/pkg/hash"
	"github.com/lexora/backend/pkg/jwt"
	"github.com/pquerna/otp/totp"
)

type Auth struct {
	users       domain.UserRepository
	members     domain.MembershipRepository
	refresh     domain.RefreshTokenRepository
	recovery    domain.RecoveryCodeRepository
	signer      *jwt.Signer // token app (aud=app)
	adminSigner *jwt.Signer // token admin (aud=admin, kunci terpisah)
	refreshTTL  time.Duration
	loginLimit  *acctLimiter
}

func NewAuth(u domain.UserRepository, m domain.MembershipRepository, r domain.RefreshTokenRepository, rec domain.RecoveryCodeRepository, s, admin *jwt.Signer, refreshTTL time.Duration) *Auth {
	return &Auth{u, m, r, rec, s, admin, refreshTTL, newAcctLimiter(10, time.Minute)}
}

type Tokens struct {
	UserID             uuid.UUID // for audit logging
	Access             string
	Refresh            string // raw token for cookie
	ExpiresIn          int    // access ttl seconds
	MustChangePassword bool
}

const accessTTLSeconds = 15 * 60

func (a *Auth) Login(ctx context.Context, email, password string) (*Tokens, error) {
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
	// super_admin lewat panel admin
	if u.SystemRole == domain.SystemRoleSuperAdmin {
		return nil, domain.ErrInvalidCreds
	}
	return a.issue(ctx, u, a.signer, uuid.Nil)
}

// hasil langkah login admin; Tokens terisi hanya saat 2FA lolos
type AdminStep struct {
	Tokens         *Tokens
	EnrollRequired bool
	MFARequired    bool
	OTPAuthURL     string   // saat enroll: bahan QR
	RecoveryCodes  []string // sekali tampil, setelah enroll sukses
}

const totpIssuer = "Lexora Admin"

// kredensial dasar admin, dipakai ketiga langkah (stateless: password ikut tiap call)
func (a *Auth) adminUser(ctx context.Context, email, password string) (*domain.User, error) {
	u, err := a.users.ByEmail(ctx, email)
	if err != nil {
		_ = hash.Verify(password, dummyHash)
		return nil, domain.ErrInvalidCreds
	}
	if err := hash.Verify(password, u.PasswordHash); err != nil {
		return nil, domain.ErrInvalidCreds
	}
	if !u.IsActive {
		return nil, domain.ErrInactive
	}
	// non-super_admin ditolak
	if u.SystemRole != domain.SystemRoleSuperAdmin {
		return nil, domain.ErrInvalidCreds
	}
	return u, nil
}

// langkah 1: password. Tidak pernah menerbitkan token (2FA wajib - update2 keputusan #5)
func (a *Auth) AdminLogin(ctx context.Context, email, password string) (*AdminStep, error) {
	u, err := a.adminUser(ctx, email, password)
	if err != nil {
		return nil, err
	}
	if u.TOTPConfirmedAt == nil {
		return a.beginEnroll(ctx, u)
	}
	return &AdminStep{MFARequired: true}, nil
}

// enrollment paksa: secret dipertahankan antar-percobaan biar QR stabil
func (a *Auth) beginEnroll(ctx context.Context, u *domain.User) (*AdminStep, error) {
	if u.TOTPSecret != nil {
		return &AdminStep{EnrollRequired: true, OTPAuthURL: otpauthURL(u.Email, *u.TOTPSecret)}, nil
	}
	key, err := totp.Generate(totp.GenerateOpts{Issuer: totpIssuer, AccountName: u.Email})
	if err != nil {
		return nil, err
	}
	if err := a.users.SetTOTPSecret(ctx, u.ID, key.Secret()); err != nil {
		return nil, err
	}
	return &AdminStep{EnrollRequired: true, OTPAuthURL: key.URL()}, nil
}

// konfirmasi enrollment: kode valid -> confirmed + recovery codes + token penuh
func (a *Auth) AdminEnroll(ctx context.Context, email, password, code string) (*AdminStep, error) {
	u, err := a.adminUser(ctx, email, password)
	if err != nil {
		return nil, err
	}
	if u.TOTPConfirmedAt != nil || u.TOTPSecret == nil {
		return nil, domain.ErrInvalidCreds
	}
	step, ok := validTOTP(*u.TOTPSecret, code, u.TOTPLastStep)
	if !ok {
		return nil, domain.ErrInvalidCreds
	}
	if err := a.users.SetTOTPLastStep(ctx, u.ID, step); err != nil {
		return nil, err
	}
	if err := a.users.ConfirmTOTP(ctx, u.ID); err != nil {
		return nil, err
	}
	codes, hashes, err := genRecoveryCodes()
	if err != nil {
		return nil, err
	}
	if err := a.recovery.Replace(ctx, u.ID, hashes); err != nil {
		return nil, err
	}
	tok, err := a.issue(ctx, u, a.adminSigner, uuid.Nil)
	if err != nil {
		return nil, err
	}
	return &AdminStep{Tokens: tok, RecoveryCodes: codes}, nil
}

// langkah 2: kode TOTP (6 digit) atau recovery code -> token penuh
func (a *Auth) AdminVerify(ctx context.Context, email, password, code string) (*Tokens, error) {
	u, err := a.adminUser(ctx, email, password)
	if err != nil {
		return nil, err
	}
	if u.TOTPConfirmedAt == nil || u.TOTPSecret == nil {
		return nil, domain.ErrInvalidCreds
	}
	code = strings.TrimSpace(code)
	if len(code) == 6 {
		step, ok := validTOTP(*u.TOTPSecret, code, u.TOTPLastStep)
		if !ok {
			return nil, domain.ErrInvalidCreds
		}
		if err := a.users.SetTOTPLastStep(ctx, u.ID, step); err != nil {
			return nil, err
		}
	} else if !a.useRecoveryCode(ctx, u.ID, code) {
		return nil, domain.ErrInvalidCreds
	}
	return a.issue(ctx, u, a.adminSigner, uuid.Nil)
}

func (a *Auth) useRecoveryCode(ctx context.Context, userID uuid.UUID, code string) bool {
	codes, err := a.recovery.Unused(ctx, userID)
	if err != nil {
		return false
	}
	for _, c := range codes {
		if hash.Verify(code, c.CodeHash) == nil {
			return a.recovery.MarkUsed(ctx, c.ID) == nil
		}
	}
	return false
}

// tiap step 30 detik hanya sah sekali (anti-replay T19); toleransi jam ±1 step; banding constant-time
func validTOTP(secret, code string, lastStep int64) (int64, bool) {
	now := time.Now().Unix() / 30
	for _, step := range []int64{now, now - 1, now + 1} {
		if step <= lastStep {
			continue
		}
		want, err := totp.GenerateCode(secret, time.Unix(step*30, 0))
		if err != nil {
			return 0, false
		}
		if subtle.ConstantTimeCompare([]byte(want), []byte(code)) == 1 {
			return step, true
		}
	}
	return 0, false
}

// 8 kode format XXXXX-XXXXX, hash argon2id (pkg/hash yang sama dengan password)
func genRecoveryCodes() (codes, hashes []string, err error) {
	for range 8 {
		b := make([]byte, 5)
		if _, err := rand.Read(b); err != nil {
			return nil, nil, err
		}
		c := strings.ToUpper(hex.EncodeToString(b))
		c = c[:5] + "-" + c[5:]
		h, err := hash.Password(c)
		if err != nil {
			return nil, nil, err
		}
		codes = append(codes, c)
		hashes = append(hashes, h)
	}
	return codes, hashes, nil
}

// rekonstruksi URL provisioning untuk secret yang sudah tersimpan
func otpauthURL(email, secret string) string {
	q := url.Values{}
	q.Set("secret", secret)
	q.Set("issuer", totpIssuer)
	q.Set("algorithm", "SHA1")
	q.Set("digits", "6")
	q.Set("period", "30")
	return "otpauth://totp/" + url.PathEscape(totpIssuer+":"+email) + "?" + q.Encode()
}

// nil = sesi baru
func (a *Auth) issue(ctx context.Context, u *domain.User, signer *jwt.Signer, familyID uuid.UUID) (*Tokens, error) {
	id := domain.Identity{UserID: u.ID, SystemRole: u.SystemRole}
	if u.SystemRole != domain.SystemRoleSuperAdmin {
		m, err := a.members.Primary(ctx, u.ID)
		if err != nil {
			return nil, err
		}
		id.OrgID = m.OrganizationID
		id.OrgRole = m.Role
	}

	access, err := signer.Sign(id)
	if err != nil {
		return nil, err
	}

	raw, err := randToken()
	if err != nil {
		return nil, err
	}
	if familyID == uuid.Nil {
		familyID = uuid.New()
	}
	rt := &domain.RefreshToken{UserID: u.ID, FamilyID: familyID, TokenHash: hashToken(raw), ExpiresAt: time.Now().Add(a.refreshTTL)}
	if err := a.refresh.Create(ctx, rt); err != nil {
		return nil, err
	}
	return &Tokens{UserID: u.ID, Access: access, Refresh: raw, ExpiresIn: accessTTLSeconds, MustChangePassword: u.MustChangePassword}, nil
}

func (a *Auth) Refresh(ctx context.Context, raw string) (*Tokens, error) {
	return a.rotate(ctx, raw, a.signer, false)
}

// rotasi sesi admin
func (a *Auth) AdminRefresh(ctx context.Context, raw string) (*Tokens, error) {
	return a.rotate(ctx, raw, a.adminSigner, true)
}

func (a *Auth) rotate(ctx context.Context, raw string, signer *jwt.Signer, superAdminOnly bool) (*Tokens, error) {
	rt, err := a.refresh.ByHash(ctx, hashToken(raw))
	if err != nil {
		return nil, domain.ErrInvalidToken
	}
	// reuse = theft, cabut family
	if rt.RevokedAt != nil {
		_ = a.refresh.RevokeFamily(ctx, rt.FamilyID)
		return nil, domain.ErrInvalidToken
	}
	if time.Now().After(rt.ExpiresAt) {
		return nil, domain.ErrInvalidToken
	}
	u, err := a.users.ByID(ctx, rt.UserID)
	if err != nil {
		return nil, domain.ErrInvalidToken
	}
	if !u.IsActive {
		return nil, domain.ErrInactive
	}
	if superAdminOnly && u.SystemRole != domain.SystemRoleSuperAdmin {
		return nil, domain.ErrInvalidToken
	}
	if err := a.refresh.Revoke(ctx, rt.ID); err != nil {
		return nil, err
	}
	return a.issue(ctx, u, signer, rt.FamilyID)
}

func (a *Auth) Logout(ctx context.Context, raw string) error {
	if raw == "" {
		return nil
	}
	return a.refresh.RevokeByHash(ctx, hashToken(raw))
}

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
