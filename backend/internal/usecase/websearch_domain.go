package usecase

import (
	"context"
	"strings"

	"github.com/lexora/backend/internal/domain"
)

// setter allowlist dinamis (guard + search). Diimpl pkg/websearch (update9-B).
type DomainSetter interface {
	SetDomains([]string)
}

// WebDomain: super_admin kelola allowlist domain web-search. Tulis DB -> refresh
// kedua enforcement point (guard SSRF + prompt search) tanpa restart.
type WebDomain struct {
	repo    domain.WebDomainRepository
	setters []DomainSetter
}

func NewWebDomain(repo domain.WebDomainRepository, setters ...DomainSetter) *WebDomain {
	return &WebDomain{repo: repo, setters: setters}
}

func (u *WebDomain) List(ctx context.Context) ([]domain.WebSearchDomain, error) {
	return u.repo.List(ctx)
}

func (u *WebDomain) Add(ctx context.Context, host string) (*domain.WebSearchDomain, error) {
	host = strings.ToLower(strings.TrimSpace(host))
	if host == "" {
		return nil, domain.ErrInvalidUpload
	}
	d, err := u.repo.Add(ctx, host)
	if err != nil {
		return nil, err
	}
	if err := u.refresh(ctx); err != nil {
		return nil, err
	}
	return d, nil
}

func (u *WebDomain) Remove(ctx context.Context, host string) error {
	if err := u.repo.Remove(ctx, strings.ToLower(strings.TrimSpace(host))); err != nil {
		return err
	}
	return u.refresh(ctx)
}

// baca DB -> suntik ke semua setter. Dipanggil saat boot + tiap perubahan.
func (u *WebDomain) Refresh(ctx context.Context) error { return u.refresh(ctx) }

func (u *WebDomain) refresh(ctx context.Context) error {
	list, err := u.repo.List(ctx)
	if err != nil {
		return err
	}
	hosts := make([]string, len(list))
	for i, d := range list {
		hosts[i] = d.Host
	}
	for _, s := range u.setters {
		s.SetDomains(hosts)
	}
	return nil
}
