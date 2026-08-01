package usecase

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/lexora/backend/internal/domain"
)

// fake repo in-mem: unique host
type fakeDomainRepo struct{ hosts []string }

func (r *fakeDomainRepo) List(context.Context) ([]domain.WebSearchDomain, error) {
	out := make([]domain.WebSearchDomain, len(r.hosts))
	for i, h := range r.hosts {
		out[i] = domain.WebSearchDomain{ID: uuid.New(), Host: h}
	}
	return out, nil
}
func (r *fakeDomainRepo) Add(_ context.Context, host string) (*domain.WebSearchDomain, error) {
	for _, h := range r.hosts {
		if h == host {
			return nil, domain.ErrConflict
		}
	}
	r.hosts = append(r.hosts, host)
	return &domain.WebSearchDomain{ID: uuid.New(), Host: host}, nil
}
func (r *fakeDomainRepo) Remove(_ context.Context, host string) error {
	for i, h := range r.hosts {
		if h == host {
			r.hosts = append(r.hosts[:i], r.hosts[i+1:]...)
			return nil
		}
	}
	return domain.ErrNotFound
}

// spy setter: rekam host terakhir yang disuntik
type spySetter struct{ got []string }

func (s *spySetter) SetDomains(hosts []string) { s.got = hosts }

// AL1: add/list/remove + unique host tolak dobel; refresh sampai ke setter.
func TestWebDomainCRUDRefreshesSetters(t *testing.T) {
	repo := &fakeDomainRepo{}
	spy := &spySetter{}
	uc := NewWebDomain(repo, spy)
	ctx := context.Background()

	if _, err := uc.Add(ctx, "jdihn.go.id"); err != nil {
		t.Fatal(err)
	}
	// refresh menyuntik host ke setter tanpa restart
	if len(spy.got) != 1 || spy.got[0] != "jdihn.go.id" {
		t.Fatalf("setter tidak ter-refresh: %v", spy.got)
	}
	// dobel ditolak (unique)
	if _, err := uc.Add(ctx, "jdihn.go.id"); err != domain.ErrConflict {
		t.Fatalf("host dobel harus ErrConflict, dapat %v", err)
	}
	// host kosong ditolak sebelum sentuh repo
	if _, err := uc.Add(ctx, "  "); err != domain.ErrInvalidUpload {
		t.Fatalf("host kosong harus ErrInvalidUpload, dapat %v", err)
	}

	list, _ := uc.List(ctx)
	if len(list) != 1 {
		t.Fatalf("list = %d, mau 1", len(list))
	}

	// remove -> setter kosong lagi
	if err := uc.Remove(ctx, "jdihn.go.id"); err != nil {
		t.Fatal(err)
	}
	if len(spy.got) != 0 {
		t.Fatalf("setelah remove setter harus kosong: %v", spy.got)
	}
	if err := uc.Remove(ctx, "tidak-ada.go.id"); err != domain.ErrNotFound {
		t.Fatalf("remove host tak ada harus ErrNotFound, dapat %v", err)
	}
}
