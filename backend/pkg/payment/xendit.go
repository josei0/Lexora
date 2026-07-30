package payment

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// Xendit Invoice API. Basic auth: secret key = username, password kosong.
// ponytail: tanpa retry/circuit-breaker. Gagal -> caller biarkan invoice pending
// tanpa URL (user klik bayar lagi, idempoten via external_id di sisi Xendit).
type Xendit struct {
	secretKey  string
	successURL string
	http       *http.Client
}

func NewXendit(secretKey, successURL string) *Xendit {
	return &Xendit{
		secretKey:  secretKey,
		successURL: successURL,
		http:       &http.Client{Timeout: 15 * time.Second},
	}
}

type xenditReq struct {
	ExternalID         string `json:"external_id"`
	Amount             int64  `json:"amount"`
	PayerEmail         string `json:"payer_email,omitempty"`
	Description        string `json:"description,omitempty"`
	SuccessRedirectURL string `json:"success_redirect_url,omitempty"`
}

type xenditResp struct {
	ID         string `json:"id"`
	InvoiceURL string `json:"invoice_url"`
}

func (x *Xendit) CreateInvoice(ctx context.Context, in Invoice) (*Created, error) {
	success := in.SuccessURL
	if success == "" {
		success = x.successURL
	}
	body, err := json.Marshal(xenditReq{
		ExternalID:         in.ExternalID,
		Amount:             in.AmountIDR,
		PayerEmail:         in.PayerEmail,
		Description:        in.Description,
		SuccessRedirectURL: success,
	})
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.xendit.co/v2/invoices", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(x.secretKey, "") // password kosong
	req.Header.Set("Content-Type", "application/json")

	resp, err := x.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		var buf bytes.Buffer
		buf.ReadFrom(resp.Body)
		return nil, fmt.Errorf("xendit %d: %s", resp.StatusCode, buf.String())
	}
	var r xenditResp
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		return nil, err
	}
	if r.ID == "" || r.InvoiceURL == "" {
		return nil, fmt.Errorf("xendit: respons tanpa id/invoice_url")
	}
	return &Created{ProviderID: r.ID, CheckoutURL: r.InvoiceURL}, nil
}
