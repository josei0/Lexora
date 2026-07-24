package http

import (
	"net/http"

	"github.com/lexora/backend/internal/delivery/http/dto"
	"github.com/lexora/backend/internal/delivery/http/handler"
	"github.com/lexora/backend/internal/delivery/http/middleware"
	"github.com/lexora/backend/internal/usecase"
	"github.com/lexora/backend/pkg/jwt"
)

func NewRouter(api *handler.API, chats *handler.ChatAPI, billing *handler.BillingAPI, signer, adminSigner *jwt.Signer, corsApp, corsAdmin []string) http.Handler {
	strict := dto.NewStrictHandler(api, nil)
	generated := dto.HandlerWithOptions(strict, dto.StdHTTPServerOptions{
		BaseURL:     "",
		Middlewares: []dto.MiddlewareFunc{},
	})

	// chat + billing di luar openapi: mux manual
	mux := http.NewServeMux()
	chats.Routes(mux)
	billing.Routes(mux)
	api.AdminAuthRoutes(mux)
	mux.Handle("/", generated)

	var chain http.Handler = mux
	chain = middleware.Auth(signer, adminSigner)(chain)
	chain = middleware.BodyLimit(1<<20, usecase.MaxUploadBytes+(1<<20))(chain)
	chain = middleware.RateLimitLogin(chain)
	chain = middleware.CORS(corsApp, corsAdmin)(chain)
	chain = middleware.SecureHeaders(chain)
	chain = middleware.Recover(chain)
	return chain
}
