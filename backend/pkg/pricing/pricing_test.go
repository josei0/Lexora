package pricing

import (
	"math"
	"testing"
)

func TestCostUSD(t *testing.T) {
	// sonnet: 1jt in @3.0 + 1jt out @15.0 = 18.0
	if got := CostUSD("maia/claude-sonnet-4-5", 1_000_000, 1_000_000); math.Abs(got-18.0) > 1e-9 {
		t.Errorf("sonnet cost = %v, mau 18.0", got)
	}
	// haiku: 500rb in @0.8 + 0 out = 0.4
	if got := CostUSD("anthropic/claude-haiku-4-5", 500_000, 0); math.Abs(got-0.4) > 1e-9 {
		t.Errorf("haiku cost = %v, mau 0.4", got)
	}
	// nol token = nol cost
	if got := CostUSD("maia/claude-sonnet-4-5", 0, 0); got != 0 {
		t.Errorf("nol token harus 0, got %v", got)
	}
}

// model tak dikenal -> fallback (konservatif, jangan under-estimate)
func TestCostFallback(t *testing.T) {
	unknown := CostUSD("model/ngawur", 1_000_000, 0)
	sonnet := CostUSD("maia/claude-sonnet-4-5", 1_000_000, 0)
	if unknown != sonnet {
		t.Errorf("model tak dikenal harus pakai fallback termahal, got %v vs %v", unknown, sonnet)
	}
	if Known("model/ngawur") {
		t.Error("model ngawur tak boleh Known")
	}
}
