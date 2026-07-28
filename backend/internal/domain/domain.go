package domain

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

var (
	ErrNotFound      = errors.New("not found")
	ErrInvalidCreds  = errors.New("invalid credentials")
	ErrConflict      = errors.New("conflict")
	ErrForbidden     = errors.New("forbidden")
	ErrInactive      = errors.New("account inactive")
	ErrInvalidToken  = errors.New("invalid token")
	ErrWrongPassword = errors.New("wrong password")
	ErrRateLimited   = errors.New("rate limited")
	ErrQuotaExceeded = errors.New("quota exceeded")
	ErrSubExpired      = errors.New("subscription expired")
	ErrDailyCapReached = errors.New("daily message cap reached")
	ErrSeatsFull     = errors.New("seats full")

	ErrInvalidUpload   = errors.New("invalid upload")
	ErrTooLarge        = errors.New("file too large")
	ErrUnsupportedType = errors.New("unsupported file type")
)

const (
	SystemRoleSuperAdmin = "super_admin"
	SystemRoleNone       = "none"
	OrgRoleAdmin         = "org_admin"
	OrgRoleMember        = "member"
)

type User struct {
	ID                 uuid.UUID
	Email              string
	PasswordHash       string
	FullName           string
	SystemRole         string
	IsActive           bool
	MustChangePassword bool
	CreatedAt          time.Time

	// TOTP super_admin: secret terisi + confirmed nil = enrollment menggantung
	TOTPSecret      *string
	TOTPConfirmedAt *time.Time
	TOTPLastStep    int64 // step terakhir terpakai - anti-replay (T19)
}

// kode recovery TOTP sekali pakai
type RecoveryCode struct {
	ID       uuid.UUID
	UserID   uuid.UUID
	CodeHash string
	UsedAt   *time.Time
}

type Organization struct {
	ID        uuid.UUID
	Name      string
	Slug      string
	CreatedAt time.Time
}

type Membership struct {
	UserID         uuid.UUID
	OrganizationID uuid.UUID
	Role           string
}

type Member struct {
	UserID   uuid.UUID
	Email    string
	FullName string
	Role     string
	IsActive bool
}

type RefreshToken struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	FamilyID  uuid.UUID // reuse detection
	TokenHash string
	ExpiresAt time.Time
	RevokedAt *time.Time
}

type Identity struct {
	UserID     uuid.UUID
	OrgID      uuid.UUID // zero for super admin
	SystemRole string
	OrgRole    string
	Name       string // display only (JWT claim), bukan untuk otorisasi
	Email      string // display only
}

func (i Identity) IsSuperAdmin() bool { return i.SystemRole == SystemRoleSuperAdmin }
func (i Identity) IsOrgAdmin() bool   { return i.OrgRole == OrgRoleAdmin }

type UserRepository interface {
	ByEmail(ctx context.Context, email string) (*User, error)
	ByID(ctx context.Context, id uuid.UUID) (*User, error)
	Create(ctx context.Context, u *User) error
	UpdatePassword(ctx context.Context, id uuid.UUID, hash string) error
	SetActive(ctx context.Context, id uuid.UUID, active bool) error
	SetTOTPSecret(ctx context.Context, id uuid.UUID, secret string) error
	ConfirmTOTP(ctx context.Context, id uuid.UUID) error
	SetTOTPLastStep(ctx context.Context, id uuid.UUID, step int64) error
}

type RecoveryCodeRepository interface {
	Replace(ctx context.Context, userID uuid.UUID, hashes []string) error
	Unused(ctx context.Context, userID uuid.UUID) ([]RecoveryCode, error)
	MarkUsed(ctx context.Context, id uuid.UUID) error
}

type OrganizationRepository interface {
	Create(ctx context.Context, o *Organization) error
	List(ctx context.Context) ([]Organization, error)
	BySlug(ctx context.Context, slug string) (*Organization, error)
}

type MembershipRepository interface {
	Create(ctx context.Context, m *Membership) error
	// scoped by org - anti-IDOR
	ByUserOrg(ctx context.Context, userID, orgID uuid.UUID) (*Membership, error)
	ListByOrg(ctx context.Context, orgID uuid.UUID) ([]Member, error)
	SetRole(ctx context.Context, userID, orgID uuid.UUID, role string) error
	Primary(ctx context.Context, userID uuid.UUID) (*Membership, error)
}

type RefreshTokenRepository interface {
	Create(ctx context.Context, t *RefreshToken) error
	ByHash(ctx context.Context, hash string) (*RefreshToken, error)
	Revoke(ctx context.Context, id uuid.UUID) error
	RevokeByHash(ctx context.Context, hash string) error
	RevokeFamily(ctx context.Context, familyID uuid.UUID) error // cabut se-family
}
