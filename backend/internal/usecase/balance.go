package usecase

import (
	"context"

	"github.com/lexora/backend/internal/domain"
	"github.com/lexora/backend/pkg/pricing"
)

// sumber token per model (dipenuhi UsageRepo). Interface kecil = testable.
type modelUsageSource interface {
	ModelTokens(ctx context.Context) ([]domain.ModelUsage, error)
}

// BalanceEstimator: estimasi saldo Maia = topup manual - Σ cost token (update6 §4.1).
// Estimasi, bukan saldo real; cukup untuk alert "menipis".
type BalanceEstimator struct {
	usage      modelUsageSource
	topupTotal float64 // MAIA_TOPUP_TOTAL_USD: total isi manual di dashboard Maia
}

func NewBalanceEstimator(usage modelUsageSource, topupTotalUSD float64) *BalanceEstimator {
	return &BalanceEstimator{usage: usage, topupTotal: topupTotalUSD}
}

// Estimate: saldo tersisa (USD). Bisa negatif kalau pemakaian > topup tercatat.
func (e *BalanceEstimator) Estimate(ctx context.Context) (float64, error) {
	rows, err := e.usage.ModelTokens(ctx)
	if err != nil {
		return 0, err
	}
	spent := 0.0
	for _, r := range rows {
		spent += pricing.CostUSD(r.Model, r.InputTokens, r.OutputTokens)
	}
	return e.topupTotal - spent, nil
}
