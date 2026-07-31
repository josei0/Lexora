package payment

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// M1: request dimap benar (Bearer, name/email, amount di items.rate) + respons diparse.
func TestMayarCreateInvoiceMapsAndParses(t *testing.T) {
	var gotAuth, gotPath string
	var gotBody mayarReq
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotPath = r.URL.Path
		b, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(b, &gotBody)
		w.WriteHeader(200)
		w.Write([]byte(`{"statusCode":200,"messages":"success","data":{"id":"inv-1","transactionId":"txn-99","link":"https://pay.mayar/x"}}`))
	}))
	defer srv.Close()

	m := NewMayar("KEY123", srv.URL, "https://app/success")
	c, err := m.CreateInvoice(context.Background(), Invoice{
		ExternalID: "uuid-1", AmountIDR: 79000, PayerEmail: "a@b.id", PayerName: "Andi", Description: "Top-up 500rb",
	})
	if err != nil {
		t.Fatal(err)
	}
	// auth + path
	if gotAuth != "Bearer KEY123" {
		t.Errorf("auth = %q, mau Bearer KEY123", gotAuth)
	}
	if gotPath != "/hl/v2/invoices/create" {
		t.Errorf("path = %q, mau /hl/v2/invoices/create", gotPath)
	}
	// body: name/email wajib, amount di items.rate (bukan field lain)
	if gotBody.Name != "Andi" || gotBody.Email != "a@b.id" {
		t.Errorf("name/email salah map: %+v", gotBody)
	}
	if len(gotBody.Items) != 1 || gotBody.Items[0].Rate != 79000 || gotBody.Items[0].Quantity != 1 {
		t.Errorf("amount harus di items[0].rate=79000 qty=1, dapat %+v", gotBody.Items)
	}
	if gotBody.RedirectURL != "https://app/success" {
		t.Errorf("redirectUrl salah: %q", gotBody.RedirectURL)
	}
	// respons: provider_id = transactionId (yg muncul di webhook), link = checkout
	if c.ProviderID != "txn-99" {
		t.Errorf("ProviderID harus transactionId (txn-99), dapat %q", c.ProviderID)
	}
	if c.CheckoutURL != "https://pay.mayar/x" {
		t.Errorf("CheckoutURL salah: %q", c.CheckoutURL)
	}
}

// provider_id fallback ke id kalau transactionId kosong
func TestMayarProviderIDFallback(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"statusCode":200,"data":{"id":"inv-only","transactionId":"","link":"https://pay/x"}}`))
	}))
	defer srv.Close()
	c, err := NewMayar("K", srv.URL, "").CreateInvoice(context.Background(), Invoice{
		AmountIDR: 1000, PayerEmail: "a@b.id", PayerName: "A",
	})
	if err != nil || c.ProviderID != "inv-only" {
		t.Fatalf("fallback ke id gagal: id=%q err=%v", c.ProviderID, err)
	}
}

// M-email: name/email kosong -> tolak SEBELUM hit HTTP (Mayar pasti 404)
func TestMayarRejectsEmptyPayer(t *testing.T) {
	hit := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { hit = true }))
	defer srv.Close()
	m := NewMayar("K", srv.URL, "")

	for _, in := range []Invoice{
		{AmountIDR: 1000, PayerEmail: "", PayerName: "A"},
		{AmountIDR: 1000, PayerEmail: "a@b.id", PayerName: ""},
		{AmountIDR: 1000, PayerEmail: "  ", PayerName: "A"},
	} {
		if _, err := m.CreateInvoice(context.Background(), in); !errors.Is(err, ErrPayerRequired) {
			t.Errorf("payer kosong harus ErrPayerRequired, dapat %v (in=%+v)", err, in)
		}
	}
	if hit {
		t.Fatal("tak boleh hit HTTP saat payer kosong (buang request pasti-gagal)")
	}
}

// M2: HTTP non-2xx -> error (caller biarkan invoice pending tanpa URL)
func TestMayarHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
		w.Write([]byte(`{"messages":"server error"}`))
	}))
	defer srv.Close()
	_, err := NewMayar("K", srv.URL, "").CreateInvoice(context.Background(), Invoice{
		AmountIDR: 1000, PayerEmail: "a@b.id", PayerName: "A",
	})
	if err == nil || !strings.Contains(err.Error(), "500") {
		t.Fatalf("HTTP 500 harus error memuat status, dapat %v", err)
	}
}

// HTTP 200 tapi statusCode body != 2xx (Mayar validation error) -> error
func TestMayarBodyStatusError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte(`{"statusCode":400,"messages":"Validation Error"}`))
	}))
	defer srv.Close()
	_, err := NewMayar("K", srv.URL, "").CreateInvoice(context.Background(), Invoice{
		AmountIDR: 1000, PayerEmail: "a@b.id", PayerName: "A",
	})
	if err == nil || !strings.Contains(err.Error(), "Validation Error") {
		t.Fatalf("body statusCode 400 harus error, dapat %v", err)
	}
}

// respons sukses tapi link kosong -> error (jangan balik checkout kosong)
func TestMayarMissingLink(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"statusCode":200,"data":{"id":"x","transactionId":"y","link":""}}`))
	}))
	defer srv.Close()
	_, err := NewMayar("K", srv.URL, "").CreateInvoice(context.Background(), Invoice{
		AmountIDR: 1000, PayerEmail: "a@b.id", PayerName: "A",
	})
	if err == nil {
		t.Fatal("link kosong harus error, jangan balik CheckoutURL kosong")
	}
}
