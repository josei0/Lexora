package usecase

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/lexora/backend/internal/domain"
)

type fakeSubs struct {
	sub          *domain.SubscriptionView
	sessionSetTo *time.Time // rekam SetSessionStarted (update8)
}

func (f *fakeSubs) ByOrg(context.Context, uuid.UUID) (*domain.SubscriptionView, error) {
	if f.sub == nil {
		return nil, domain.ErrNotFound
	}
	return f.sub, nil
}
func (f *fakeSubs) Upsert(context.Context, *domain.Subscription) error { return nil }
func (f *fakeSubs) SetSessionStarted(_ context.Context, _ uuid.UUID, at time.Time) error {
	f.sessionSetTo = &at
	if f.sub != nil {
		f.sub.SessionStartedAt = &at // cerminkan ke state (spt DB)
	}
	return nil
}

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

// subscription dgn masa aktif berakhir `daysAgo` hari lalu
func expiredSub(daysAgo int) *domain.SubscriptionView {
	end := time.Now().AddDate(0, 0, -daysAgo)
	v := subView(1000, 1)
	v.CurrentPeriodEnd = &end
	return v
}

// U4: lewat periode tapi masih grace -> tetap bisa dipakai, ditandai past_due
func TestSubPastDueStillWorks(t *testing.T) {
	b := NewBilling(&fakeSubs{sub: expiredSub(3)}, &fakeUsage{tokens: 10})
	q, err := b.Check(context.Background(), uuid.New(), time.Now())
	if err != nil {
		t.Fatalf("dalam grace tidak boleh diblok: %v", err)
	}
	if q.Status != domain.SubPastDue {
		t.Fatalf("mau past_due, dapat %q", q.Status)
	}
}

// U5: lewat grace -> expired, fitur menulis ditolak
func TestSubExpiredBlocksAfterGrace(t *testing.T) {
	b := NewBilling(&fakeSubs{sub: expiredSub(domain.GraceDays + 1)}, &fakeUsage{tokens: 10})
	if _, err := b.Check(context.Background(), uuid.New(), time.Now()); err != domain.ErrSubExpired {
		t.Fatalf("lewat grace harus ErrSubExpired, dapat %v", err)
	}
	if err := b.GateAccess(context.Background(), uuid.New(), time.Now()); err != domain.ErrSubExpired {
		t.Fatalf("GateAccess harus tolak expired, dapat %v", err)
	}
}

// U10: subscription lama (period NULL) tidak boleh mendadak terkunci saat migrasi
func TestNilPeriodNeverExpires(t *testing.T) {
	b := NewBilling(&fakeSubs{sub: subView(1000, 1)}, &fakeUsage{tokens: 10})
	q, err := b.Check(context.Background(), uuid.New(), time.Now())
	if err != nil {
		t.Fatalf("period NULL harus active: %v", err)
	}
	if q.Status != domain.SubActive {
		t.Fatalf("mau active, dapat %q", q.Status)
	}
}

// masa aktif habis lebih diprioritaskan daripada kuota: pesannya beda
func TestExpiredBeatsOverflow(t *testing.T) {
	v := expiredSub(domain.GraceDays + 1)
	b := NewBilling(&fakeSubs{sub: v}, &fakeUsage{tokens: 999999})
	if _, err := b.Check(context.Background(), uuid.New(), time.Now()); err != domain.ErrSubExpired {
		t.Fatalf("expired harus menang atas overflow, dapat %v", err)
	}
}

// Check tidak lagi error di hard: kebijakan degrade/blok milik RAG.gate.
// Yang diblok Billing cuma overflow (plafon absolut).
func TestBillingHardFlaggedNotBlocked(t *testing.T) {
	// limit 1000/seat * 2 seats = 2000; sudah pakai 2000 -> hard, belum overflow
	b := NewBilling(&fakeSubs{sub: subView(1000, 2)}, &fakeUsage{tokens: 2000})
	q, err := b.Check(context.Background(), uuid.New(), time.Now())
	if err != nil {
		t.Fatalf("hard tidak boleh diblok di Billing (degrade urusan RAG): %v", err)
	}
	if !q.Hard || q.Overflow {
		t.Fatalf("mau hard tanpa overflow, dapat hard=%v overflow=%v", q.Hard, q.Overflow)
	}
}

func TestBillingBlocksAtOverflow(t *testing.T) {
	// 2x limit -> plafon degrade tembus
	b := NewBilling(&fakeSubs{sub: subView(1000, 2)}, &fakeUsage{tokens: 4000})
	if _, err := b.Check(context.Background(), uuid.New(), time.Now()); err != domain.ErrQuotaExceeded {
		t.Fatalf("mau ErrQuotaExceeded di overflow, dapat %v", err)
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
