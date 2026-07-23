package usecase

import (
	"context"

	"github.com/google/uuid"
	"github.com/lexora/backend/internal/domain"
)

// Subscription: assign plan + seats per org (super admin), baca subscription org
type Subscription struct {
	subs  domain.SubscriptionRepository
	plans domain.PlanRepository
	usage domain.UsageRepository
}

func NewSubscription(subs domain.SubscriptionRepository, plans domain.PlanRepository, usage domain.UsageRepository) *Subscription {
	return &Subscription{subs: subs, plans: plans, usage: usage}
}

func (s *Subscription) Plans(ctx context.Context) ([]domain.Plan, error) {
	return s.plans.List(ctx)
}

func (s *Subscription) ByOrg(ctx context.Context, orgID uuid.UUID) (*domain.SubscriptionView, error) {
	return s.subs.ByOrg(ctx, orgID)
}

// super admin set plan + seats untuk org
func (s *Subscription) Assign(ctx context.Context, orgID uuid.UUID, planCode string, seats int) (*domain.SubscriptionView, error) {
	if seats < 1 {
		return nil, domain.ErrInvalidUpload
	}
	plan, err := s.plans.ByCode(ctx, planCode)
	if err != nil {
		return nil, err
	}
	// tolak kalau anggota sekarang sudah lebih dari seats yang diminta
	members, err := s.usage.CountMembers(ctx, orgID)
	if err != nil {
		return nil, err
	}
	if members > seats {
		return nil, domain.ErrSeatsFull
	}
	sub := &domain.Subscription{OrganizationID: orgID, PlanID: plan.ID, Seats: seats}
	if err := s.subs.Upsert(ctx, sub); err != nil {
		return nil, err
	}
	return s.subs.ByOrg(ctx, orgID)
}

// dipanggil sebelum tambah anggota; error kalau seats penuh.
// org tanpa subscription = tanpa batas seat (belum onboard billing).
func (s *Subscription) GuardSeat(ctx context.Context, orgID uuid.UUID) error {
	sub, err := s.subs.ByOrg(ctx, orgID)
	if err == domain.ErrNotFound {
		return nil
	}
	if err != nil {
		return err
	}
	members, err := s.usage.CountMembers(ctx, orgID)
	if err != nil {
		return err
	}
	if members >= sub.Seats {
		return domain.ErrSeatsFull
	}
	return nil
}
