package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/lexora/backend/internal/domain"
	"github.com/lexora/backend/pkg/jwt"
)

type ctxKey int

const (
	identityKey ctxKey = iota
	refreshKey
	clientIPKey
)

// __Host- butuh Secure, dev pakai nama polos
func RefreshCookieName(secure bool) string {
	if secure {
		return "__Host-lexora_rt"
	}
	return "lexora_rt"
}

func AdminRefreshCookieName(secure bool) string {
	if secure {
		return "__Host-lexora_admin_rt"
	}
	return "lexora_admin_rt"
}

// coba dua nama
func readUserRefresh(r *http.Request) string {
	for _, name := range []string{RefreshCookieName(true), RefreshCookieName(false)} {
		if c, err := r.Cookie(name); err == nil {
			return c.Value
		}
	}
	return ""
}

// pull identity from ctx
func IdentityFrom(ctx context.Context) (domain.Identity, bool) {
	id, ok := ctx.Value(identityKey).(domain.Identity)
	return id, ok
}

// pull raw refresh cookie from ctx
func RefreshFrom(ctx context.Context) string {
	v, _ := ctx.Value(refreshKey).(string)
	return v
}

// pull client ip from ctx (for audit log)
func ClientIPFrom(ctx context.Context) string {
	v, _ := ctx.Value(clientIPKey).(string)
	return v
}

// inject client ip into ctx so handlers/usecase can audit it (clientIP defined in middleware.go)
func ClientIP(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := context.WithValue(r.Context(), clientIPKey, clientIP(r))
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// verify bearer, terima aud app+admin
func Auth(user, admin *jwt.Signer) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			h := r.Header.Get("Authorization")
			if raw, ok := strings.CutPrefix(h, "Bearer "); ok {
				id, err := user.Verify(raw)
				if err != nil {
					if id, err = admin.Verify(raw); err != nil {
						writeError(w, http.StatusUnauthorized, "unauthorized", "sesi tidak valid")
						return
					}
				}
				r = r.WithContext(context.WithValue(r.Context(), identityKey, id))
			}
			// inject refresh cookie for /auth/refresh + /auth/logout
			if raw := readUserRefresh(r); raw != "" {
				r = r.WithContext(context.WithValue(r.Context(), refreshKey, raw))
			}
			next.ServeHTTP(w, r)
		})
	}
}
