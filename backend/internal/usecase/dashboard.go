package usecase

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/lexora/backend/internal/domain"
)

// Dashboard: snapshot statistik org (hari ini WIB, bulan berjalan)
type Dashboard struct {
	usage domain.UsageRepository
	subs  domain.SubscriptionRepository
}

func NewDashboard(usage domain.UsageRepository, subs domain.SubscriptionRepository) *Dashboard {
	return &Dashboard{usage: usage, subs: subs}
}

func (d *Dashboard) Stats(ctx context.Context, orgID uuid.UUID, now time.Time) (*domain.DashboardStats, error) {
	// awal hari ini WIB
	n := now.In(wib)
	dayStart := time.Date(n.Year(), n.Month(), n.Day(), 0, 0, 0, 0, wib)
	from, to := monthWindow(now)

	var st domain.DashboardStats
	var err error

	if st.ChatsToday, err = d.usage.ChatsSince(ctx, orgID, dayStart); err != nil {
		return nil, err
	}
	if st.TokensMonth, err = d.usage.OrgTokens(ctx, orgID, from, to); err != nil {
		return nil, err
	}
	if st.DocsIndexed, st.DocsTotal, err = d.usage.DocCounts(ctx, orgID); err != nil {
		return nil, err
	}
	if st.Members, err = d.usage.CountMembers(ctx, orgID); err != nil {
		return nil, err
	}

	// limit + seats dari subscription (kalau ada)
	if sub, err := d.subs.ByOrg(ctx, orgID); err == nil {
		st.Seats = sub.Seats
		st.TokenLimit = sub.Plan.MonthlyTokenLimit * int64(sub.Seats)
	}
	return &st, nil
}
