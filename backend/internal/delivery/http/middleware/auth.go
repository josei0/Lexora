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
)

const RefreshCookieName = "lexora_refresh"

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

// verify bearer -> inject identity. invalid token -> 401. absent -> continue
func Auth(signer *jwt.Signer) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			h := r.Header.Get("Authorization")
			if strings.HasPrefix(h, "Bearer ") {
				id, err := signer.Verify(strings.TrimPrefix(h, "Bearer "))
				if err != nil {
					writeError(w, http.StatusUnauthorized, "unauthorized", "sesi tidak valid")
					return
				}
				r = r.WithContext(context.WithValue(r.Context(), identityKey, id))
			}
			// inject refresh cookie for /auth/refresh + /auth/logout
			if c, err := r.Cookie(RefreshCookieName); err == nil {
				r = r.WithContext(context.WithValue(r.Context(), refreshKey, c.Value))
			}
			next.ServeHTTP(w, r)
		})
	}
}
