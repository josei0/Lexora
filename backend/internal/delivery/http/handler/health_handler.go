package handler

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/lexora/backend/internal/delivery/http/dto"
	"github.com/lexora/backend/internal/delivery/http/middleware"
	"github.com/lexora/backend/internal/usecase"
)

// implements dto.StrictServerInterface
type API struct {
	auth       *usecase.Auth
	org        *usecase.Organization
	docs       *usecase.Document
	audit      *usecase.Audit
	refreshTTL time.Duration
	secure     bool // Secure flag on cookie (prod)
}

func New(auth *usecase.Auth, org *usecase.Organization, docs *usecase.Document, audit *usecase.Audit, refreshTTL time.Duration, secure bool) *API {
	return &API{auth: auth, org: org, docs: docs, audit: audit, refreshTTL: refreshTTL, secure: secure}
}

// liveness
func (a *API) GetHealth(ctx context.Context, _ dto.GetHealthRequestObject) (dto.GetHealthResponseObject, error) {
	return dto.GetHealth200JSONResponse{Status: "ok"}, nil
}

// build refresh cookie string
func (a *API) cookie(value string, maxAge time.Duration) string {
	c := &http.Cookie{
		Name:     middleware.RefreshCookieName,
		Value:    value,
		Path:     "/auth",
		HttpOnly: true,
		Secure:   a.secure,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   int(maxAge.Seconds()),
	}
	return c.String()
}

func (a *API) clearCookie() string {
	return fmt.Sprint((&http.Cookie{
		Name: middleware.RefreshCookieName, Value: "", Path: "/auth",
		HttpOnly: true, Secure: a.secure, SameSite: http.SameSiteStrictMode, MaxAge: -1,
	}).String())
}
