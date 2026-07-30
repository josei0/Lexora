package usecase

import (
	"context"
	"math"
	"testing"

	"github.com/lexora/backend/internal/domain"
)

type fakeUsageSrc struct{ rows []domain.ModelUsage }

func (f *fakeUsageSrc) ModelTokens(context.Context) ([]domain.ModelUsage, error) {
	return f.rows, nil
}

// U2: estimasi = topup - Σ cost, turun sesuai token dipakai
func TestEstimateSubtractsCost(t *testing.T) {
	src := &fakeUsageSrc{rows: []domain.ModelUsage{
		{Model: "maia/claude-sonnet-4-5", InputTokens: 1_000_000, OutputTokens: 1_000_000}, // 18.0
		{Model: "anthropic/claude-haiku-4-5", InputTokens: 500_000, OutputTokens: 0},        // 0.4
	}}
	est := NewBalanceEstimator(src, 100.0) // topup $100

	bal, err := est.Estimate(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	// 100 - (18.0 + 0.4) = 81.6
	if math.Abs(bal-81.6) > 1e-9 {
		t.Errorf("estimasi = %v, mau 81.6", bal)
	}
}

// tanpa pemakaian -> saldo = topup penuh
func TestEstimateNoUsage(t *testing.T) {
	est := NewBalanceEstimator(&fakeUsageSrc{}, 50.0)
	bal, _ := est.Estimate(context.Background())
	if bal != 50.0 {
		t.Errorf("tanpa pemakaian saldo harus penuh 50, got %v", bal)
	}
}

// U3: pemakaian lewat topup -> saldo di bawah threshold (memicu alert)
func TestEstimateBelowThreshold(t *testing.T) {
	src := &fakeUsageSrc{rows: []domain.ModelUsage{
		{Model: "maia/claude-sonnet-4-5", InputTokens: 5_000_000, OutputTokens: 5_000_000}, // 90.0
	}}
	est := NewBalanceEstimator(src, 100.0)
	bal, _ := est.Estimate(context.Background())
	const threshold = 20.0
	// 100 - 90 = 10 < 20 -> alert
	if !(bal < threshold) {
		t.Errorf("saldo %v harus di bawah threshold %v (memicu alert)", bal, threshold)
	}
}
