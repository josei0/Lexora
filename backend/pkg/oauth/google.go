package oauth

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"time"
)

// verifikasi Google ID token via endpoint publik tokeninfo (stdlib, tanpa lib JWKS).
// ponytail: cukup untuk volume login user. Kalau besar, ganti ke verifikasi JWT
// lokal pakai JWKS Google (oauth2/v3/certs) - alur pemanggil tak berubah.
type Google struct {
	clientID string
	http     *http.Client
}

func NewGoogle(clientID string) *Google {
	return &Google{clientID: clientID, http: &http.Client{Timeout: 10 * time.Second}}
}

// Enabled: false kalau GOOGLE_CLIENT_ID kosong (fitur Google mati).
func (g *Google) Enabled() bool { return g.clientID != "" }

type Claims struct {
	Sub   string // subject Google, stabil per akun
	Email string
	Name  string
}

var (
	ErrDisabled     = errors.New("google login tidak aktif")
	ErrInvalidToken = errors.New("google id token tidak valid")
)

// tokeninfo response (field yang kita pakai). email_verified & aud bisa string.
type tokenInfo struct {
	Aud           string `json:"aud"`
	Sub           string `json:"sub"`
	Email         string `json:"email"`
	EmailVerified string `json:"email_verified"` // "true"/"false" (string)
	Name          string `json:"name"`
}

// Verify: validasi id token ke Google, cek aud == clientID + email terverifikasi.
func (g *Google) Verify(ctx context.Context, idToken string) (*Claims, error) {
	if !g.Enabled() {
		return nil, ErrDisabled
	}
	if idToken == "" {
		return nil, ErrInvalidToken
	}
	u := "https://oauth2.googleapis.com/tokeninfo?id_token=" + url.QueryEscape(idToken)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	resp, err := g.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, ErrInvalidToken // token kadaluarsa/palsu -> Google balas 4xx
	}
	var ti tokenInfo
	if err := json.NewDecoder(resp.Body).Decode(&ti); err != nil {
		return nil, ErrInvalidToken
	}
	// aud wajib cocok client kita (cegah token dari app lain dipakai di sini)
	if ti.Aud != g.clientID {
		return nil, ErrInvalidToken
	}
	if ti.EmailVerified != "true" || ti.Email == "" || ti.Sub == "" {
		return nil, ErrInvalidToken
	}
	return &Claims{Sub: ti.Sub, Email: ti.Email, Name: ti.Name}, nil
}
