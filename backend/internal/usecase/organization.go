package usecase

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"time"

	"github.com/google/uuid"
	"github.com/lexora/backend/internal/domain"
	"github.com/lexora/backend/pkg/hash"
)

// Mailer: interface kecil (dependency inversion) - impl di pkg/mailer.
type Mailer interface {
	Send(to, subject, body string) error
	Enabled() bool
}

type Organization struct {
	orgs       domain.OrganizationRepository
	users      domain.UserRepository
	members    domain.MembershipRepository
	seats      *Subscription // ponytail: nil = tanpa batas seat
	mailer     Mailer        // nil = register langsung aktif (dev tanpa SMTP)
	appBaseURL string        // untuk link verifikasi email
}

func NewOrganization(o domain.OrganizationRepository, u domain.UserRepository, m domain.MembershipRepository) *Organization {
	return &Organization{orgs: o, users: u, members: m}
}

func (o *Organization) SetSeatGuard(s *Subscription) { o.seats = s }

// SetMailer: aktifkan verifikasi email register. Kosong = akun langsung aktif.
func (o *Organization) SetMailer(m Mailer, appBaseURL string) {
	o.mailer, o.appBaseURL = m, appBaseURL
}

const verifyTTL = 24 * time.Hour

// opsi addUser: Create/AddMember pakai default (temp password, aktif, must-change);
// Register pakai eksplisit (password user, nonaktif sampai verif).
type addUserOpts struct {
	password    *string // nil = generate temp password
	isActive    bool
	mustChange  bool
	verifyHash  *string
	verifyExp   *time.Time
	googleSub   *string
	verifiedNow bool // true (jalur Google) -> email_verified_at = now
}

type NewMember struct {
	UserID       uuid.UUID
	Email        string
	TempPassword string
}

// create org + first org admin (super admin only)
func (o *Organization) Create(ctx context.Context, name, slug, adminEmail, adminName string) (*domain.Organization, *NewMember, error) {
	org := &domain.Organization{Name: name, Slug: slug}
	if err := o.orgs.Create(ctx, org); err != nil {
		return nil, nil, err
	}
	// auto-assign Demo: tutup celah "tanpa subscription = unlimited".
	// Demo tanpa current_period_end -> tidak pernah expired.
	if o.seats != nil {
		if _, err := o.seats.Assign(ctx, org.ID, domain.PlanDemo, 1); err != nil {
			return nil, nil, err
		}
	}
	admin, err := o.addUser(ctx, org.ID, adminEmail, adminName, domain.OrgRoleAdmin, addUserOpts{isActive: true, mustChange: true})
	if err != nil {
		return nil, nil, err
	}
	return org, admin, nil
}

func (o *Organization) AddMember(ctx context.Context, orgID uuid.UUID, email, fullName, role string) (*NewMember, error) {
	if role != domain.OrgRoleAdmin && role != domain.OrgRoleMember {
		return nil, domain.ErrForbidden
	}
	if o.seats != nil {
		if err := o.seats.GuardSeat(ctx, orgID); err != nil {
			return nil, err
		}
	}
	return o.addUser(ctx, orgID, email, fullName, role, addUserOpts{isActive: true, mustChange: true})
}

// Register: self-serve. Bikin org+Demo+org_admin dgn password user, akun nonaktif
// sampai verifikasi email. mailer nil / disabled -> langsung aktif (dev).
// skipVerify=true (jalur Google): aktif langsung tanpa email. googleSub opsional.
func (o *Organization) Register(ctx context.Context, firmaName, adminName, email, password string, skipVerify bool, googleSub *string) (*domain.User, error) {
	slug, err := uniqueSlug(firmaName, func(s string) bool {
		_, err := o.orgs.BySlug(ctx, s)
		return err == nil
	})
	if err != nil {
		return nil, err
	}
	org := &domain.Organization{Name: firmaName, Slug: slug}
	if err := o.orgs.Create(ctx, org); err != nil {
		return nil, err
	}
	if o.seats != nil {
		if _, err := o.seats.Assign(ctx, org.ID, domain.PlanDemo, 1); err != nil {
			return nil, err
		}
	}

	// verifikasi email aktif hanya kalau mailer siap DAN bukan jalur skip (Google)
	active := skipVerify || o.mailer == nil || !o.mailer.Enabled()
	// Google (skipVerify) sudah memverifikasi email -> tandai verified sekarang
	opts := addUserOpts{password: &password, isActive: active, mustChange: false, googleSub: googleSub, verifiedNow: skipVerify}

	var rawToken string
	if !active {
		rawToken, err = randToken()
		if err != nil {
			return nil, err
		}
		h := hashToken(rawToken)
		exp := time.Now().Add(verifyTTL)
		opts.verifyHash, opts.verifyExp = &h, &exp
	}

	nm, err := o.addUser(ctx, org.ID, email, adminName, domain.OrgRoleAdmin, opts)
	if err != nil {
		return nil, err
	}
	if !active {
		o.sendVerifyEmail(email, rawToken) // best-effort; akun tetap dibuat
	}
	u, err := o.users.ByID(ctx, nm.UserID)
	if err != nil {
		return nil, err
	}
	return u, nil
}

// VerifyEmail: token cocok + belum kadaluarsa -> aktifkan akun.
func (o *Organization) VerifyEmail(ctx context.Context, rawToken string) error {
	u, err := o.users.ByVerifyToken(ctx, hashToken(rawToken))
	if err != nil {
		return domain.ErrInvalidToken
	}
	if u.VerifyExpiresAt == nil || time.Now().After(*u.VerifyExpiresAt) {
		return domain.ErrInvalidToken
	}
	return o.users.VerifyEmail(ctx, u.ID)
}

func (o *Organization) sendVerifyEmail(email, rawToken string) {
	if o.mailer == nil {
		return
	}
	link := o.appBaseURL + "/verify-email?token=" + rawToken
	_ = o.mailer.Send(email, "Verifikasi email MindLaw",
		"Klik untuk mengaktifkan akun MindLaw Anda:\n\n"+link+"\n\nTautan berlaku 24 jam.")
}

// addUser: bikin user + membership. opts.password nil = generate temp (dipakai
// Create/AddMember). NewMember.TempPassword kosong kalau password dari user (Register).
func (o *Organization) addUser(ctx context.Context, orgID uuid.UUID, email, fullName, role string, opts addUserOpts) (*NewMember, error) {
	var temp, plain string
	if opts.password != nil {
		plain = *opts.password
	} else {
		t, err := tempPassword()
		if err != nil {
			return nil, err
		}
		temp, plain = t, t
	}
	h, err := hash.Password(plain)
	if err != nil {
		return nil, err
	}
	u := &domain.User{
		Email:              email,
		PasswordHash:       h,
		FullName:           fullName,
		SystemRole:         domain.SystemRoleNone,
		IsActive:           opts.isActive,
		MustChangePassword: opts.mustChange,
		VerifyTokenHash:    opts.verifyHash,
		VerifyExpiresAt:    opts.verifyExp,
		GoogleSub:          opts.googleSub,
	}
	if opts.verifiedNow {
		now := time.Now()
		u.EmailVerifiedAt = &now
	}
	if err := o.users.Create(ctx, u); err != nil {
		return nil, err
	}
	if err := o.members.Create(ctx, &domain.Membership{UserID: u.ID, OrganizationID: orgID, Role: role}); err != nil {
		return nil, err
	}
	return &NewMember{UserID: u.ID, Email: email, TempPassword: temp}, nil
}

func (o *Organization) List(ctx context.Context) ([]domain.Organization, error) {
	return o.orgs.List(ctx)
}

func (o *Organization) ListMembers(ctx context.Context, orgID uuid.UUID) ([]domain.Member, error) {
	return o.members.ListByOrg(ctx, orgID)
}

// set member role - scoped to caller's org
func (o *Organization) UpdateMember(ctx context.Context, orgID, userID uuid.UUID, role *string, active *bool) (*domain.Member, error) {
	// ownership check - user must belong to caller's org
	if _, err := o.members.ByUserOrg(ctx, userID, orgID); err != nil {
		return nil, err
	}
	if role != nil {
		if *role != domain.OrgRoleAdmin && *role != domain.OrgRoleMember {
			return nil, domain.ErrForbidden
		}
		if err := o.members.SetRole(ctx, userID, orgID, *role); err != nil {
			return nil, err
		}
	}
	if active != nil {
		if err := o.users.SetActive(ctx, userID, *active); err != nil {
			return nil, err
		}
	}
	members, err := o.members.ListByOrg(ctx, orgID)
	if err != nil {
		return nil, err
	}
	for i := range members {
		if members[i].UserID == userID {
			return &members[i], nil
		}
	}
	return nil, domain.ErrNotFound
}

func tempPassword() (string, error) {
	b := make([]byte, 12)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
