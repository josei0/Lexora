package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/lexora/backend/internal/delivery/http/middleware"
	"github.com/lexora/backend/internal/domain"
	"github.com/lexora/backend/internal/usecase"
	"github.com/lexora/backend/pkg/jwt"
)

// fake repo in-mem, unique host
type memDomainRepo struct{ hosts []string }

func (r *memDomainRepo) List(context.Context) ([]domain.WebSearchDomain, error) {
	out := make([]domain.WebSearchDomain, len(r.hosts))
	for i, h := range r.hosts {
		out[i] = domain.WebSearchDomain{ID: uuid.New(), Host: h}
	}
	return out, nil
}
func (r *memDomainRepo) Add(_ context.Context, host string) (*domain.WebSearchDomain, error) {
	for _, h := range r.hosts {
		if h == host {
			return nil, domain.ErrConflict
		}
	}
	r.hosts = append(r.hosts, host)
	return &domain.WebSearchDomain{ID: uuid.New(), Host: host}, nil
}
func (r *memDomainRepo) Remove(_ context.Context, host string) error {
	for i, h := range r.hosts {
		if h == host {
			r.hosts = append(r.hosts[:i], r.hosts[i+1:]...)
			return nil
		}
	}
	return domain.ErrNotFound
}

// server dgn auth chain nyata (mint JWT -> lewat middleware.Auth)
func domainServer(repo *memDomainRepo) (http.Handler, *jwt.Signer) {
	uc := usecase.NewWebDomain(repo)
	api := NewWebDomainAPI(uc)
	mux := http.NewServeMux()
	api.Routes(mux)
	signer := jwt.New("test-secret", time.Hour, jwt.AudienceAdmin)
	return middleware.Auth(signer, signer)(mux), signer
}

func token(t *testing.T, s *jwt.Signer, role string) string {
	t.Helper()
	tok, err := s.Sign(domain.Identity{UserID: uuid.New(), SystemRole: role})
	if err != nil {
		t.Fatal(err)
	}
	return tok
}

// AL4: non-super_admin -> 403
func TestWebDomainForbidsNonSuperAdmin(t *testing.T) {
	srv, signer := domainServer(&memDomainRepo{})
	req := httptest.NewRequest(http.MethodGet, "/admin/web-domains", nil)
	req.Header.Set("Authorization", "Bearer "+token(t, signer, domain.SystemRoleNone))
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("non-admin harus 403, dapat %d", rec.Code)
	}
}

// AL5: POST tambah host -> muncul di List (super_admin)
func TestWebDomainAddThenList(t *testing.T) {
	repo := &memDomainRepo{}
	srv, signer := domainServer(repo)
	bearer := "Bearer " + token(t, signer, domain.SystemRoleSuperAdmin)

	// POST tambah
	req := httptest.NewRequest(http.MethodPost, "/admin/web-domains", strings.NewReader(`{"host":"jdihn.go.id"}`))
	req.Header.Set("Authorization", bearer)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("POST harus 201, dapat %d (%s)", rec.Code, rec.Body.String())
	}

	// GET list memuat host baru
	req = httptest.NewRequest(http.MethodGet, "/admin/web-domains", nil)
	req.Header.Set("Authorization", bearer)
	rec = httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "jdihn.go.id") {
		t.Fatalf("list harus memuat host baru: %d %s", rec.Code, rec.Body.String())
	}
}
