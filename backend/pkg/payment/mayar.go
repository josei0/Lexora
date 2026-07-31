package payment

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// Mayar Invoice API v2. Bearer auth. POST {base}/hl/v2/invoices/create.
// Base: https://api.mayar.id (prod) / https://api.mayar.club (sandbox).
// Endpoint & path terverifikasi dari CLI resmi Mayar (mayar-cli/src/commands/invoice.js).
// ponytail: tanpa retry. Gagal -> caller biarkan invoice pending tanpa URL
// (user klik bayar lagi). Idempotency di sisi kita via external_id/provider_id.
type Mayar struct {
	apiKey     string
	baseURL    string // tanpa trailing slash, mis. https://api.mayar.id/hl/v1
	successURL string
	http       *http.Client
}

// ErrPayerRequired: Mayar wajib name+email (bikin customer). Kosong = pasti gagal,
// jadi kita tolak lebih awal daripada kirim request yang pasti 404.
var ErrPayerRequired = errors.New("mayar: nama dan email pembayar wajib diisi")

func NewMayar(apiKey, baseURL, successURL string) *Mayar {
	base := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if base == "" {
		base = "https://api.mayar.id" // prod; sandbox = https://api.mayar.club
	}
	return &Mayar{
		apiKey:     apiKey,
		baseURL:    base,
		successURL: successURL,
		http:       &http.Client{Timeout: 15 * time.Second},
	}
}

type mayarItem struct {
	Quantity    int    `json:"quantity"`
	Rate        int64  `json:"rate"` // harga per item (IDR). total = rate*quantity
	Description string `json:"description"`
}

type mayarReq struct {
	Name        string      `json:"name"`
	Email       string      `json:"email"`
	Description string      `json:"description,omitempty"`
	RedirectURL string      `json:"redirectUrl,omitempty"`
	Items       []mayarItem `json:"items"`
}

// respons Mayar: {statusCode, messages, data:{id, transactionId, link, ...}}
type mayarResp struct {
	StatusCode int    `json:"statusCode"`
	Messages   string `json:"messages"`
	Data       struct {
		ID            string `json:"id"`
		TransactionID string `json:"transactionId"`
		Link          string `json:"link"`
	} `json:"data"`
}

func (m *Mayar) CreateInvoice(ctx context.Context, in Invoice) (*Created, error) {
	// Mayar bikin customer dari name+email; kosong -> 404. Tolak lebih awal.
	if strings.TrimSpace(in.PayerEmail) == "" || strings.TrimSpace(in.PayerName) == "" {
		return nil, ErrPayerRequired
	}
	success := in.SuccessURL
	if success == "" {
		success = m.successURL
	}
	body, err := json.Marshal(mayarReq{
		Name:        in.PayerName,
		Email:       in.PayerEmail,
		Description: in.Description,
		RedirectURL: success,
		Items: []mayarItem{{
			Quantity:    1,
			Rate:        in.AmountIDR, // total = rate*1 = AmountIDR (server-computed)
			Description: in.Description,
		}},
	})
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, m.baseURL+"/hl/v2/invoices/create", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+m.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := m.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var r mayarResp
	// baca body sekali; decode + sertakan di error kalau gagal
	var buf bytes.Buffer
	buf.ReadFrom(resp.Body)
	if resp.StatusCode/100 != 2 {
		return nil, fmt.Errorf("mayar %d: %s", resp.StatusCode, buf.String())
	}
	if err := json.Unmarshal(buf.Bytes(), &r); err != nil {
		return nil, fmt.Errorf("mayar: decode respons: %w", err)
	}
	// Mayar bisa balas HTTP 200 tapi statusCode != 200 di body (validation error dst)
	if r.StatusCode != 0 && r.StatusCode/100 != 2 {
		return nil, fmt.Errorf("mayar status %d: %s", r.StatusCode, r.Messages)
	}
	// korelasi webhook: simpan transactionId (yang muncul di webhook data.transactionId).
	// fallback ke id kalau transactionId kosong.
	providerID := r.Data.TransactionID
	if providerID == "" {
		providerID = r.Data.ID
	}
	if providerID == "" || r.Data.Link == "" {
		return nil, fmt.Errorf("mayar: respons tanpa id/link (messages=%q)", r.Messages)
	}
	return &Created{ProviderID: providerID, CheckoutURL: r.Data.Link}, nil
}

// FetchedInvoice: status invoice hasil re-fetch ke API Mayar (sumber kebenaran,
// bukan payload webhook yang tak bisa diverifikasi).
type FetchedInvoice struct {
	ID     string
	Status string // "SUCCESS"/"paid"/dst (apa adanya dari Mayar)
	Amount int64
	Paid   bool // true kalau status menandakan lunas
}

// GetInvoice: re-fetch status invoice by id. GET /hl/v2/invoices/{id}.
// Dipakai webhook untuk verifikasi (webhook Mayar tanpa signature -> jangan
// percaya payload, cek balik ke API pakai API key kita).
func (m *Mayar) GetInvoice(ctx context.Context, id string) (*FetchedInvoice, error) {
	if strings.TrimSpace(id) == "" {
		return nil, fmt.Errorf("mayar: id kosong")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, m.baseURL+"/hl/v2/invoices/"+id, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+m.apiKey)
	req.Header.Set("Accept", "application/json")

	resp, err := m.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var buf bytes.Buffer
	buf.ReadFrom(resp.Body)
	if resp.StatusCode/100 != 2 {
		return nil, fmt.Errorf("mayar get %d: %s", resp.StatusCode, buf.String())
	}
	var r struct {
		StatusCode int    `json:"statusCode"`
		Messages   string `json:"messages"`
		Data       struct {
			ID     string `json:"id"`
			Status string `json:"status"`
			Amount int64  `json:"amount"`
		} `json:"data"`
	}
	if err := json.Unmarshal(buf.Bytes(), &r); err != nil {
		return nil, fmt.Errorf("mayar get: decode: %w", err)
	}
	if r.StatusCode != 0 && r.StatusCode/100 != 2 {
		return nil, fmt.Errorf("mayar get status %d: %s", r.StatusCode, r.Messages)
	}
	// ⚠️ Nilai status "lunas" Mayar belum dikonfirmasi dari sandbox — CLI resmi
	// tak hardcode-nya. Webhook payload pakai "SUCCESS"; filter transaksi pakai
	// "paid". Normalisasi longgar (terima keduanya + varian umum). Handler webhook
	// TIDAK boleh gantung HANYA ke Paid ini — kombinasikan dgn: (1) event =
	// payment.received (hanya fire saat sudah bayar), (2) amount cocok. Paid = lapis ketiga.
	// TODO(sandbox): konfirmasi nilai status persis, ketatkan kalau perlu.
	s := strings.ToUpper(strings.TrimSpace(r.Data.Status))
	paid := s == "SUCCESS" || s == "PAID" || s == "SETTLED" || s == "CLOSED" || s == "ACTIVE"
	return &FetchedInvoice{ID: r.Data.ID, Status: r.Data.Status, Amount: r.Data.Amount, Paid: paid}, nil
}
