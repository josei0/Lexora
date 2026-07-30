// Package pricing: harga per-token per model untuk estimasi biaya Maia (update6 §4.1).
package pricing

// TODO(harga): VERIFIKASI harga EFEKTIF Maia sebelum produksi.
// Angka di bawah = list-price publik Anthropic/OpenAI (USD per 1 JUTA token).
// Maia Router bisa markup di atas list-price — ganti dengan harga tagihan Maia
// yang sebenarnya. Estimasi saldo hanya seakurat angka ini.
type price struct{ inPerM, outPerM float64 } // USD per 1_000_000 token

var modelPrice = map[string]price{
	"maia/claude-sonnet-4-5":            {inPerM: 3.00, outPerM: 15.00},
	"anthropic/claude-sonnet-4-5":       {inPerM: 3.00, outPerM: 15.00},
	"anthropic/claude-haiku-4-5":        {inPerM: 0.80, outPerM: 4.00},
	"maia/claude-haiku-4-5":             {inPerM: 0.80, outPerM: 4.00},
	"openai/text-embedding-3-large":     {inPerM: 0.13, outPerM: 0.00}, // embedding: output 0
	"openai/gpt-4o-mini-search-preview": {inPerM: 0.15, outPerM: 0.60},
}

// fallback: model tak dikenal -> pakai harga model termahal (konservatif, jangan
// under-estimate biaya). Alert lebih baik terlalu cepat daripada telat.
var fallback = price{inPerM: 3.00, outPerM: 15.00}

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
