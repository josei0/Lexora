package usecase

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"strings"
	"unicode"
)

// langka: 6 percobaan slug semua bentrok (praktis mustahil)
var ErrSlugExhausted = errors.New("gagal membuat slug unik")

// slugify: nama firma -> slug URL-aman. lowercase, non-alfanumerik jadi '-',
// gabung '-' beruntun, trim. Contoh: "Budi & Rekan, S.H." -> "budi-rekan-s-h".
func slugify(name string) string {
	var b strings.Builder
	prevDash := true // true di awal -> cegah leading '-'
	for _, r := range strings.ToLower(name) {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			b.WriteRune(r)
			prevDash = false
		case !prevDash:
			b.WriteByte('-')
			prevDash = true
		}
	}
	return strings.Trim(b.String(), "-")
}

// randomSuffix: 4 byte -> 8 hex char, untuk anti-bentrok slug.
func randomSuffix() (string, error) {
	b := make([]byte, 4)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// uniqueSlug: slugify(name) + suffix acak kalau slug dasar kosong atau bentrok.
// exists: cek keberadaan slug (biasanya OrgRepo.BySlug != ErrNotFound).
// ponytail: coba 5x; praktis mustahil bentrok 5x dgn 32-bit suffix.
func uniqueSlug(name string, exists func(string) bool) (string, error) {
	base := slugify(name)
	if base == "" {
		base = "org" // nama full-simbol/kosong -> fallback
	}
	if !exists(base) {
		return base, nil
	}
	for range 5 {
		suf, err := randomSuffix()
		if err != nil {
			return "", err
		}
		cand := base + "-" + suf
		if !exists(cand) {
			return cand, nil
		}
	}
	return "", ErrSlugExhausted
}
