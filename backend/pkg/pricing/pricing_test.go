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
	// haiku: 500rb in @1.0 + 0 out = 0.5 (harga katalog Maia)
	if got := CostUSD("anthropic/claude-haiku-4-5", 500_000, 0); math.Abs(got-0.5) > 1e-9 {
		t.Errorf("haiku cost = %v, mau 0.5", got)
	}
	// nol token = nol cost
	if got := CostUSD("maia/claude-sonnet-4-5", 0, 0); got != 0 {
		t.Errorf("nol token harus 0, got %v", got)
	}
}

// model tak dikenal -> fallback (konservatif, jangan under-estimate)
func TestCostFallback(t *testing.T) {
	// fallback = model termahal yg diketahui (opus-4-1: $15/1M in)
	unknown := CostUSD("model/ngawur", 1_000_000, 0)
	opus := CostUSD("anthropic/claude-opus-4-1", 1_000_000, 0)
	if unknown != opus {
		t.Errorf("model tak dikenal harus pakai fallback termahal (opus), got %v vs %v", unknown, opus)
	}
	// fallback tak boleh lebih murah dari model termahal manapun (jangan under-estimate)
	if unknown < CostUSD("anthropic/claude-sonnet-4-5", 1_000_000, 0) {
		t.Error("fallback harus >= model chat termahal (konservatif)")
	}
	if Known("model/ngawur") {
		t.Error("model ngawur tak boleh Known")
	}
}
