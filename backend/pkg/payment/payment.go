// Package payment: gateway tagihan (Xendit). Satu interface, satu impl (pola pkg/llm).
// Ganti gateway = ganti file ini, bukan arsitektur (update6 §1.1).
package payment

import "context"

// Gateway: kirim 1 tagihan, balik URL checkout. externalID = UUID invoice kita
// (idempotency + korelasi webhook).
type Gateway interface {
	CreateInvoice(ctx context.Context, in Invoice) (*Created, error)
}

type Invoice struct {
	ExternalID  string // invoice.ID kita
	AmountIDR   int64
	PayerEmail  string
	Description string // "Top-up 500 ribu token" / "Perpanjangan Pro 3 seat"
	SuccessURL  string // redirect setelah bayar
}

type Created struct {
	ProviderID  string // xendit invoice id
	CheckoutURL string // invoice_url
}
