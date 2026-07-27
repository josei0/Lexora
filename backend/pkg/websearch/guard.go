package websearch

import (
	"fmt"
	"net"
	"net/url"
	"strings"
)

// SSRF guard (10-SECURITY #9). Keamanan: tidak kena ponytail.
type Guard struct{ allowed []string }

func NewGuard(domains []string) *Guard {
	out := make([]string, 0, len(domains))
	for _, d := range domains {
		if d = strings.ToLower(strings.TrimSpace(d)); d != "" {
			out = append(out, d)
		}
	}
	return &Guard{allowed: out}
}

func (g *Guard) Domains() []string { return g.allowed }

// skema + allowlist domain + resolve DNS -> tolak IP internal
func (g *Guard) Check(raw string) error {
	u, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("url tidak valid")
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("skema %q tidak diizinkan", u.Scheme)
	}
	host := strings.ToLower(u.Hostname())
	if host == "" {
		return fmt.Errorf("host kosong")
	}
	if !g.allowedHost(host) {
		return fmt.Errorf("domain %q di luar allowlist", host)
	}
	// resolve dulu: allowlist saja tidak cukup kalau DNS-nya menunjuk ke dalam
	ips, err := net.LookupIP(host)
	if err != nil {
		return fmt.Errorf("resolve %q gagal", host)
	}
	for _, ip := range ips {
		if isInternal(ip) {
			return fmt.Errorf("host %q resolve ke IP internal", host)
		}
	}
	return nil
}

// cocok persis atau subdomain (bukan substring: "evil-bpk.go.id" harus tertolak)
func (g *Guard) allowedHost(host string) bool {
	for _, d := range g.allowed {
		if host == d || strings.HasSuffix(host, "."+d) {
			return true
		}
	}
	return false
}

func isInternal(ip net.IP) bool {
	if ip.IsLoopback() || ip.IsPrivate() || ip.IsUnspecified() ||
		ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsMulticast() {
		return true
	}
	// IPv4-mapped IPv6 (::ffff:127.0.0.1) - cek ulang bentuk v4-nya
	if v4 := ip.To4(); v4 != nil && !ip.Equal(v4) {
		return isInternal(v4)
	}
	// IPv6 unique local fc00::/7
	if len(ip) == net.IPv6len && ip.To4() == nil && ip[0]&0xfe == 0xfc {
		return true
	}
	return false
}
