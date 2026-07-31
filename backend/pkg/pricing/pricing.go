// Package pricing: harga per-token per model untuk estimasi biaya Maia (update6 §4.1).
package pricing

// Harga EFEKTIF Maia Router (USD per 1 JUTA token), diverifikasi dari Model Catalog
// MAIA Router 2026-07-30. Chat (sonnet/haiku/opus) = angka nyata katalog.
// Embedding & web-search BELUM ada di katalog -> masih placeholder list-price (ditandai).
type price struct{ inPerM, outPerM float64 } // USD per 1_000_000 token

var modelPrice = map[string]price{
	// ── Anthropic chat (verified: MAIA Model Catalog 2026-07-30) ──
	"anthropic/claude-sonnet-4-5": {inPerM: 3.00, outPerM: 15.00},
	"maia/claude-sonnet-4-5":      {inPerM: 3.00, outPerM: 15.00},
	"anthropic/claude-sonnet-4-6": {inPerM: 3.00, outPerM: 15.00},
	"anthropic/claude-sonnet-5":   {inPerM: 2.00, outPerM: 10.00},
	"anthropic/claude-haiku-4-5":  {inPerM: 1.00, outPerM: 5.00}, // dikoreksi dari 0.80/4.00
	"maia/claude-haiku-4-5":       {inPerM: 1.00, outPerM: 5.00},
	"anthropic/claude-opus-4-1":   {inPerM: 15.00, outPerM: 75.00},
	"anthropic/claude-opus-4-5":   {inPerM: 5.00, outPerM: 25.00},
	"anthropic/claude-opus-4-6":   {inPerM: 5.00, outPerM: 25.00},
	"anthropic/claude-opus-4-7":   {inPerM: 5.00, outPerM: 25.00},
	"anthropic/claude-opus-4-8":   {inPerM: 5.00, outPerM: 25.00},

	// ── BELUM diverifikasi di katalog Maia (placeholder list-price OpenAI) ──
	// TODO(harga): ganti saat embedding/web-search muncul di Model Catalog Maia.
	"openai/text-embedding-3-large":     {inPerM: 0.13, outPerM: 0.00}, // embedding: output 0
	"openai/gpt-4o-mini-search-preview": {inPerM: 0.15, outPerM: 0.60},
}

// fallback: model tak dikenal -> harga termahal yang kita tahu (opus-4-1),
// konservatif: jangan under-estimate biaya. Alert lebih baik kepagian dari telat.
var fallback = price{inPerM: 15.00, outPerM: 75.00}

// CostUSD: biaya satu penggunaan (input+output token pada 1 model).
func CostUSD(model string, inputTokens, outputTokens int64) float64 {
	p, ok := modelPrice[model]
	if !ok {
		p = fallback
	}
	return float64(inputTokens)/1e6*p.inPerM + float64(outputTokens)/1e6*p.outPerM
}

// Known: true kalau model punya harga eksplisit (bukan fallback). Untuk logging.
func Known(model string) bool {
	_, ok := modelPrice[model]
	return ok
}
