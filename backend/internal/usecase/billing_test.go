package usecase

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/lexora/backend/internal/domain"
)

type fakeSubs struct{ sub *domain.SubscriptionView }

func (f *fakeSubs) ByOrg(context.Context, uuid.UUID) (*domain.SubscriptionView, error) {
	if f.sub == nil {
		return nil, domain.ErrNotFound
	}
	return f.sub, nil
}
func (f *fakeSubs) Upsert(context.Context, *domain.Subscription) error { return nil }

type fakeUsage struct {
	tokens  int64
	members int
}

func (f *fakeUsage) OrgTokens(context.Context, uuid.UUID, time.Time, time.Time) (int64, error) {
	return f.tokens, nil
}
func (f *fakeUsage) CountMembers(context.Context, uuid.UUID) (int, error) { return f.members, nil }
func (f *fakeUsage) ChatsSince(context.Context, uuid.UUID, time.Time) (int, error) {
	return 0, nil
}
func (f *fakeUsage) DocCounts(context.Context, uuid.UUID) (int, int, error) { return 0, 0, nil }

func subView(limitPerSeat int64, seats int) *domain.SubscriptionView {
	return &domain.SubscriptionView{
		Subscription: domain.Subscription{Seats: seats},
		Plan:         domain.Plan{MonthlyTokenLimit: limitPerSeat},
	}
}

func TestBillingHardBlocksAtLimit(t *testing.T) {
	// limit 1000/seat * 2 seats = 2000; sudah pakai 2000 -> hard
	b := NewBilling(&fakeSubs{sub: subView(1000, 2)}, &fakeUsage{tokens: 2000})
	_, err := b.Check(context.Background(), uuid.New(), time.Now())
	if err != domain.ErrQuotaExceeded {
		t.Fatalf("mau ErrQuotaExceeded, dapat %v", err)
	}
}

func TestBillingSoftAt80(t *testing.T) {
	// limit 1000; pakai 800 -> soft, belum hard
	b := NewBilling(&fakeSubs{sub: subView(1000, 1)}, &fakeUsage{tokens: 800})
	q, err := b.Usage(context.Background(), uuid.New(), time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if !q.Soft || q.Hard {
		t.Fatalf("mau soft & bukan hard, dapat soft=%v hard=%v", q.Soft, q.Hard)
	}
}

func TestBillingNoSubscriptionUnlimited(t *testing.T) {
	// tanpa subscription -> tanpa limit, tidak pernah blok
	b := NewBilling(&fakeSubs{}, &fakeUsage{tokens: 1 << 40})
	q, err := b.Check(context.Background(), uuid.New(), time.Now())
	if err != nil {
		t.Fatalf("org tanpa subscription tidak boleh diblok: %v", err)
	}
	if q.Limit != 0 || q.Remaining() != -1 {
		t.Fatalf("mau unlimited, dapat limit=%d remaining=%d", q.Limit, q.Remaining())
	}
}

func TestSeatGuardRejectsWhenFull(t *testing.T) {
	// 2 seats, sudah 2 anggota -> tolak tambah
	s := NewSubscription(&fakeSubs{sub: subView(0, 2)}, nil, &fakeUsage{members: 2})
	if err := s.GuardSeat(context.Background(), uuid.New()); err != domain.ErrSeatsFull {
		t.Fatalf("mau ErrSeatsFull, dapat %v", err)
	}
}

func TestSeatGuardAllowsWhenRoom(t *testing.T) {
	s := NewSubscription(&fakeSubs{sub: subView(0, 3)}, nil, &fakeUsage{members: 2})
	if err := s.GuardSeat(context.Background(), uuid.New()); err != nil {
		t.Fatalf("masih ada slot, tidak boleh error: %v", err)
	}
}
