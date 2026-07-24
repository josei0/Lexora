package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/lexora/backend/internal/delivery/http/dto"
	"github.com/lexora/backend/internal/delivery/http/middleware"
	"github.com/lexora/backend/internal/usecase"
)

// implements dto.StrictServerInterface
type API struct {
	auth         *usecase.Auth
	org          *usecase.Organization
	docs         *usecase.Document
	audit        *usecase.Audit
	refreshTTL   time.Duration
	secure       bool            // secure cookie
	adminOrigins map[string]bool // origin admin sah
}

func New(auth *usecase.Auth, org *usecase.Organization, docs *usecase.Document, audit *usecase.Audit, refreshTTL time.Duration, secure bool, adminOrigins []string) *API {
	set := make(map[string]bool, len(adminOrigins))
	for _, o := range adminOrigins {
		set[o] = true
	}
	return &API{auth: auth, org: org, docs: docs, audit: audit, refreshTTL: refreshTTL, secure: secure, adminOrigins: set}
}

// liveness
func (a *API) GetHealth(ctx context.Context, _ dto.GetHealthRequestObject) (dto.GetHealthResponseObject, error) {
	return dto.GetHealth200JSONResponse{Status: "ok"}, nil
}

// build cookie refresh
func (a *API) refreshCookie(name, value string, maxAge time.Duration) string {
	return (&http.Cookie{
		Name:     name,
		Value:    value,
		Path:     "/",
		HttpOnly: true,
		Secure:   a.secure,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   int(maxAge.Seconds()),
	}).String()
}

func (a *API) cookie(value string, maxAge time.Duration) string {
	return a.refreshCookie(middleware.RefreshCookieName(a.secure), value, maxAge)
}

func (a *API) clearCookie() string {
	return a.refreshCookie(middleware.RefreshCookieName(a.secure), "", -1)
}

func (a *API) adminCookie(value string, maxAge time.Duration) string {
	return a.refreshCookie(middleware.AdminRefreshCookieName(a.secure), value, maxAge)
}

func (a *API) clearAdminCookie() string {
	return a.refreshCookie(middleware.AdminRefreshCookieName(a.secure), "", -1)
}
