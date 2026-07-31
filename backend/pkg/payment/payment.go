// Package payment: gateway tagihan. Satu interface, banyak impl (pola pkg/llm).
// Ganti gateway = ganti impl, bukan arsitektur (update6 §1.1). Aktif: Mayar.
package payment

import "context"

// Gateway: kirim 1 tagihan, balik URL checkout. externalID = UUID invoice kita
// (idempotency + korelasi webhook via provider_id).
type Gateway interface {
	CreateInvoice(ctx context.Context, in Invoice) (*Created, error)
}

type Invoice struct {
	ExternalID  string // invoice.ID kita
	AmountIDR   int64
	PayerEmail  string // WAJIB non-kosong untuk Mayar (customer dibuat dari ini)
	PayerName   string // WAJIB non-kosong untuk Mayar
	Description string // "Top-up 500 ribu token" / "Perpanjangan Pro 3 seat"
	SuccessURL  string // redirect setelah bayar
}

type Created struct {
	ProviderID  string // id transaksi gateway (disimpan ke invoices.provider_id)
	CheckoutURL string // URL bayar
}
