package usecase

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/lexora/backend/internal/domain"
)

// in-memory fakes

type fakeUsers struct{ m map[uuid.UUID]*domain.User }

func (f *fakeUsers) ByEmail(_ context.Context, email string) (*domain.User, error) {
	for _, u := range f.m {
		if u.Email == email {
			return u, nil
		}
	}
	return nil, domain.ErrNotFound
}
func (f *fakeUsers) ByID(_ context.Context, id uuid.UUID) (*domain.User, error) {
	if u, ok := f.m[id]; ok {
		return u, nil
	}
	return nil, domain.ErrNotFound
}
func (f *fakeUsers) Create(_ context.Context, u *domain.User) error {
	u.ID = uuid.New()
	f.m[u.ID] = u
	return nil
}
func (f *fakeUsers) UpdatePassword(_ context.Context, id uuid.UUID, h string) error {
	f.m[id].PasswordHash = h
	return nil
}
func (f *fakeUsers) SetActive(_ context.Context, id uuid.UUID, a bool) error {
	f.m[id].IsActive = a
	return nil
}
func (f *fakeUsers) SetTOTPSecret(_ context.Context, id uuid.UUID, s string) error {
	f.m[id].TOTPSecret = &s
	return nil
}
func (f *fakeUsers) ConfirmTOTP(_ context.Context, id uuid.UUID) error {
	now := time.Now()
	f.m[id].TOTPConfirmedAt = &now
	return nil
}
func (f *fakeUsers) SetTOTPLastStep(_ context.Context, id uuid.UUID, step int64) error {
	f.m[id].TOTPLastStep = step
	return nil
}
func (f *fakeUsers) ByVerifyToken(_ context.Context, h string) (*domain.User, error) {
	for _, u := range f.m {
		if u.VerifyTokenHash != nil && *u.VerifyTokenHash == h {
			return u, nil
		}
	}
	return nil, domain.ErrNotFound
}
func (f *fakeUsers) VerifyEmail(_ context.Context, id uuid.UUID) error {
	now := time.Now()
	u := f.m[id]
	u.IsActive, u.EmailVerifiedAt, u.VerifyTokenHash, u.VerifyExpiresAt = true, &now, nil, nil
	return nil
}
func (f *fakeUsers) ByGoogleSub(_ context.Context, sub string) (*domain.User, error) {
	for _, u := range f.m {
		if u.GoogleSub != nil && *u.GoogleSub == sub {
			return u, nil
		}
	}
	return nil, domain.ErrNotFound
}
func (f *fakeUsers) LinkGoogle(_ context.Context, id uuid.UUID, sub string) error {
	f.m[id].GoogleSub = &sub
	return nil
}

type fakeMembers struct{ list []domain.Membership }

func (f *fakeMembers) Create(_ context.Context, m *domain.Membership) error {
	f.list = append(f.list, *m)
	return nil
}

// scoped by org - the anti-IDOR guard under test
func (f *fakeMembers) ByUserOrg(_ context.Context, userID, orgID uuid.UUID) (*domain.Membership, error) {
	for i := range f.list {
		if f.list[i].UserID == userID && f.list[i].OrganizationID == orgID {
			return &f.list[i], nil
		}
	}
	return nil, domain.ErrNotFound
}
func (f *fakeMembers) ListByOrg(_ context.Context, orgID uuid.UUID) ([]domain.Member, error) {
	var out []domain.Member
	for _, m := range f.list {
		if m.OrganizationID == orgID {
			out = append(out, domain.Member{UserID: m.UserID, Role: m.Role, IsActive: true})
		}
	}
	return out, nil
}
func (f *fakeMembers) SetRole(_ context.Context, userID, orgID uuid.UUID, role string) error {
	for i := range f.list {
		if f.list[i].UserID == userID && f.list[i].OrganizationID == orgID {
			f.list[i].Role = role
			return nil
		}
	}
	return domain.ErrNotFound
}
func (f *fakeMembers) Primary(_ context.Context, userID uuid.UUID) (*domain.Membership, error) {
	for i := range f.list {
		if f.list[i].UserID == userID {
			return &f.list[i], nil
		}
	}
	return nil, domain.ErrNotFound
}

type fakeOrgs struct{ list []domain.Organization }

func (f *fakeOrgs) Create(_ context.Context, o *domain.Organization) error {
	o.ID = uuid.New()
	f.list = append(f.list, *o)
	return nil
}
func (f *fakeOrgs) List(_ context.Context) ([]domain.Organization, error) { return f.list, nil }
func (f *fakeOrgs) BySlug(_ context.Context, s string) (*domain.Organization, error) {
	return nil, domain.ErrNotFound
}

// tenant isolation: org A admin cannot touch a user in org B
func TestUpdateMemberCrossOrgBlocked(t *testing.T) {
	orgA, orgB := uuid.New(), uuid.New()
	userB := uuid.New()
	members := &fakeMembers{list: []domain.Membership{
		{UserID: userB, OrganizationID: orgB, Role: domain.OrgRoleMember},
	}}
	uc := NewOrganization(&fakeOrgs{}, &fakeUsers{m: map[uuid.UUID]*domain.User{}}, members)

	admin := domain.OrgRoleAdmin
	// orgA admin tries to promote userB (who lives in orgB)
	_, err := uc.UpdateMember(context.Background(), orgA, userB, &admin, nil)
	if err != domain.ErrNotFound {
		t.Fatalf("cross-org update must be blocked, got: %v", err)
	}
	// userB role must be unchanged
	if members.list[0].Role != domain.OrgRoleMember {
		t.Fatalf("cross-org update leaked: role changed to %s", members.list[0].Role)
	}
}

// T3: nonaktif anggota -> cabut semua sesi refresh-nya (update9-S)
func TestDeactivateRevokesRefresh(t *testing.T) {
	org := uuid.New()
	uid := uuid.New()
	members := &fakeMembers{list: []domain.Membership{
		{UserID: uid, OrganizationID: org, Role: domain.OrgRoleMember},
	}}
	users := &fakeUsers{m: map[uuid.UUID]*domain.User{uid: {ID: uid, IsActive: true}}}
	rt := &fakeRefresh{m: map[string]*domain.RefreshToken{
		"h1": {ID: uuid.New(), UserID: uid, FamilyID: uuid.New(), TokenHash: "h1"},
	}}
	uc := NewOrganization(&fakeOrgs{}, users, members)
	uc.SetRefreshRevoker(rt)

	off := false
	if _, err := uc.UpdateMember(context.Background(), org, uid, nil, &off); err != nil {
		t.Fatal(err)
	}
	if rt.m["h1"].RevokedAt == nil {
		t.Fatal("nonaktif anggota harus cabut refresh token-nya")
	}
}

// list scoping: caller only sees own org members
func TestListMembersScopedToOrg(t *testing.T) {
	orgA, orgB := uuid.New(), uuid.New()
	members := &fakeMembers{list: []domain.Membership{
		{UserID: uuid.New(), OrganizationID: orgA, Role: domain.OrgRoleMember},
		{UserID: uuid.New(), OrganizationID: orgA, Role: domain.OrgRoleAdmin},
		{UserID: uuid.New(), OrganizationID: orgB, Role: domain.OrgRoleMember},
	}}
	uc := NewOrganization(&fakeOrgs{}, &fakeUsers{m: map[uuid.UUID]*domain.User{}}, members)

	got, err := uc.ListMembers(context.Background(), orgA)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 org-A members, got %d (leak?)", len(got))
	}
}
