package handler

import (
	"errors"
	"testing"
	"time"

	"github.com/lexora/backend/internal/domain"
	"github.com/lexora/backend/internal/usecase"
)

// update8 F3: event error SSE saat kuota memblokir harus kaya info PAYG
func TestSSEErrorEventQuotaRich(t *testing.T) {
	reset := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	meta := usecase.AskMeta{Blocked: &usecase.WindowUsage{
		Kind: usecase.WindowMonthly, Limit: 1000, Used: 1000, ResetAt: reset,
	}}
	ev := sseErrorEvent(domain.ErrQuotaExceeded, meta)
	if ev["code"] != "quota_exceeded" || ev["window"] != "monthly" {
		t.Fatalf("event salah: %+v", ev)
	}
	if ev["reset_at"] != reset.Format(time.RFC3339) {
		t.Fatalf("reset_at salah: %v", ev["reset_at"])
	}
	pkgs, ok := ev["packages"].([]map[string]any)
	if !ok || len(pkgs) != 2 {
		t.Fatalf("packages harus 2 paket, dapat %+v", ev["packages"])
	}
	if pkgs[0]["code"] != "small" || pkgs[1]["code"] != "large" {
		t.Fatalf("urutan paket harus stabil small->large: %+v", pkgs)
	}
}

// error non-quota = event sederhana {error: pesan}; quota tanpa Blocked juga sederhana
func TestSSEErrorEventPlain(t *testing.T) {
	ev := sseErrorEvent(errors.New("boom"), usecase.AskMeta{})
	if ev["error"] != "gagal memproses pertanyaan" || len(ev) != 1 {
		t.Fatalf("event non-quota harus sederhana: %+v", ev)
	}
	ev = sseErrorEvent(domain.ErrQuotaExceeded, usecase.AskMeta{})
	if _, rich := ev["code"]; rich || len(ev) != 1 {
		t.Fatalf("quota tanpa Blocked harus sederhana: %+v", ev)
	}
}
