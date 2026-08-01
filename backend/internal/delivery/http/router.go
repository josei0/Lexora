package http

import (
	"net/http"

	"github.com/lexora/backend/internal/delivery/http/dto"
	"github.com/lexora/backend/internal/delivery/http/handler"
	"github.com/lexora/backend/internal/delivery/http/middleware"
	"github.com/lexora/backend/internal/usecase"
	"github.com/lexora/backend/pkg/jwt"
)

// cspNonce: reserved (CSP+nonce di-handle FE middleware.ts, backend API CSP statis
// di SecureHeaders). adminHost: isolasi rute admin (update9-S), kosong = nonaktif.
func NewRouter(api *handler.API, chats *handler.ChatAPI, billing *handler.BillingAPI,
	web *handler.WebAPI, invoices *handler.InvoiceAPI, wsdomains *handler.WebDomainAPI,
	signer, adminSigner *jwt.Signer, corsApp, corsAdmin []string, cspNonce bool, adminHost string) http.Handler {
	strict := dto.NewStrictHandler(api, nil)
	generated := dto.HandlerWithOptions(strict, dto.StdHTTPServerOptions{
		BaseURL:     "",
		Middlewares: []dto.MiddlewareFunc{},
	})

	// chat + billing di luar openapi: mux manual
	mux := http.NewServeMux()
	chats.Routes(mux)
	billing.Routes(mux)
	web.Routes(mux)
	invoices.Routes(mux)
	wsdomains.Routes(mux) // update9-B
	api.AdminAuthRoutes(mux)
	api.PublicAuthRoutes(mux)
	api.AdminOrgRoutes(mux)
	mux.Handle("/", generated)

	var chain http.Handler = mux
	chain = middleware.Auth(signer, adminSigner)(chain)
	chain = middleware.AdminHostOnly(adminHost)(chain) // update9-S: sebelum auth-strip, tolak Host asing
	chain = middleware.BodyLimit(1<<20, usecase.MaxUploadBytes+(1<<20))(chain)
	chain = middleware.RateLimitLogin(chain)
	chain = middleware.CORS(corsApp, corsAdmin)(chain)
	chain = middleware.SecureHeaders(chain)
	chain = middleware.Recover(chain)
	return chain
}
