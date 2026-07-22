package http

import (
	"net/http"

	"github.com/lexora/backend/internal/delivery/http/dto"
	"github.com/lexora/backend/internal/delivery/http/handler"
)

// wire routes
func NewRouter(api *handler.API) http.Handler {
	strict := dto.NewStrictHandler(api, nil)
	return dto.HandlerWithOptions(strict, dto.StdHTTPServerOptions{
		BaseURL:    "",
		Middlewares: []dto.MiddlewareFunc{},
	})
}
