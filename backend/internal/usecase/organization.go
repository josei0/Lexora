package usecase

import (
	"context"
	"crypto/rand"
	"encoding/base64"

	"github.com/google/uuid"
	"github.com/lexora/backend/internal/domain"
	"github.com/lexora/backend/pkg/hash"
)

// organization + member usecase
type Organization struct {
	orgs    domain.OrganizationRepository
	users   domain.UserRepository
	members domain.MembershipRepository
	seats   *Subscription // ponytail: nil = tanpa batas seat
}

func NewOrganization(o domain.OrganizationRepository, u domain.UserRepository, m domain.MembershipRepository) *Organization {
	return &Organization{o, u, m, nil}
}

func (o *Organization) SetSeatGuard(s *Subscription) { o.seats = s }

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
	admin, err := o.addUser(ctx, org.ID, adminEmail, adminName, domain.OrgRoleAdmin)
	if err != nil {
		return nil, nil, err
	}
	return org, admin, nil
}

// add member to caller's org (org admin only)
func (o *Organization) AddMember(ctx context.Context, orgID uuid.UUID, email, fullName, role string) (*NewMember, error) {
	if role != domain.OrgRoleAdmin && role != domain.OrgRoleMember {
		return nil, domain.ErrForbidden
	}
	if o.seats != nil {
		if err := o.seats.GuardSeat(ctx, orgID); err != nil {
			return nil, err
		}
	}
	return o.addUser(ctx, orgID, email, fullName, role)
}

// create user + membership with temp password
func (o *Organization) addUser(ctx context.Context, orgID uuid.UUID, email, fullName, role string) (*NewMember, error) {
	temp, err := tempPassword()
	if err != nil {
		return nil, err
	}
	h, err := hash.Password(temp)
	if err != nil {
		return nil, err
	}
	u := &domain.User{
		Email:              email,
		PasswordHash:       h,
		FullName:           fullName,
		SystemRole:         domain.SystemRoleNone,
		IsActive:           true,
		MustChangePassword: true,
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
	// return fresh view
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

// random one-time password
func tempPassword() (string, error) {
	b := make([]byte, 12)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
