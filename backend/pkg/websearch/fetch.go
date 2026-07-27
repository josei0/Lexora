package websearch

import (
	"context"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

const (
	maxBody     = 5 << 20 // 5MB
	fetchTimeut = 10 * time.Second
	maxRedirect = 3
)

// ambil halaman jadi teks polos. Tiap hop redirect divalidasi ulang guard —
// redirect ke IP internal itu bypass SSRF klasik.
type Fetcher struct {
	guard *Guard
	http  *http.Client
}

func NewFetcher(g *Guard) *Fetcher {
	return &Fetcher{
		guard: g,
		http: &http.Client{
			Timeout: fetchTimeut,
			CheckRedirect: func(*http.Request, []*http.Request) error {
				return http.ErrUseLastResponse // redirect diikuti manual
			},
		},
	}
}

func (f *Fetcher) FetchText(ctx context.Context, rawURL string) (title, text string, err error) {
	for range maxRedirect + 1 {
		if err := f.guard.Check(rawURL); err != nil {
			return "", "", err
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
		if err != nil {
			return "", "", err
		}
		req.Header.Set("User-Agent", "LexoraBot/1.0")

		resp, err := f.http.Do(req)
		if err != nil {
			return "", "", err
		}
		if loc := resp.Header.Get("Location"); resp.StatusCode >= 300 && resp.StatusCode < 400 && loc != "" {
			resp.Body.Close()
			next, err := resolveRef(rawURL, loc)
			if err != nil {
				return "", "", err
			}
			rawURL = next
			continue
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return "", "", fmt.Errorf("sumber balas %d", resp.StatusCode)
		}
		body, err := io.ReadAll(io.LimitReader(resp.Body, maxBody))
		if err != nil {
			return "", "", err
		}
		raw := string(body)
		return htmlTitle(raw), htmlToText(raw), nil
	}
	return "", "", fmt.Errorf("terlalu banyak redirect")
}

// Location bisa relatif; resolve ke absolut sebelum divalidasi ulang
func resolveRef(base, ref string) (string, error) {
	b, err := url.Parse(base)
	if err != nil {
		return "", fmt.Errorf("redirect tidak valid")
	}
	r, err := url.Parse(ref)
	if err != nil {
		return "", fmt.Errorf("redirect tidak valid")
	}
	return b.ResolveReference(r).String(), nil
}

var (
	// RE2 tanpa backreference: tiap tag ditulis eksplisit
	scriptStyle = regexp.MustCompile(`(?is)<script\b[^>]*>.*?</script\s*>|<style\b[^>]*>.*?</style\s*>|<noscript\b[^>]*>.*?</noscript\s*>`)
	blockTag    = regexp.MustCompile(`(?i)</?(p|div|br|li|tr|h[1-6]|section|article)\b[^>]*>`)
	anyTag      = regexp.MustCompile(`(?s)<[^>]*>`)
	manyNewline = regexp.MustCompile(`\n{3,}`)
	manySpace   = regexp.MustCompile(`[ \t]{2,}`)
	titleTag    = regexp.MustCompile(`(?is)<title[^>]*>(.*?)</title>`)
)

// buang script/style lalu seluruh tag. HTML mentah tidak pernah dikirim ke model.
func htmlToText(raw string) string {
	s := scriptStyle.ReplaceAllString(raw, " ")
	s = blockTag.ReplaceAllString(s, "\n")
	s = anyTag.ReplaceAllString(s, "")
	s = html.UnescapeString(s)
	s = strings.ReplaceAll(s, "\r", "")
	s = manySpace.ReplaceAllString(s, " ")

	lines := strings.Split(s, "\n")
	for i, l := range lines {
		lines[i] = strings.TrimSpace(l)
	}
	s = strings.Join(lines, "\n")
	return strings.TrimSpace(manyNewline.ReplaceAllString(s, "\n\n"))
}

func htmlTitle(raw string) string {
	m := titleTag.FindStringSubmatch(raw)
	if len(m) < 2 {
		return ""
	}
	return strings.TrimSpace(html.UnescapeString(anyTag.ReplaceAllString(m[1], "")))
}
