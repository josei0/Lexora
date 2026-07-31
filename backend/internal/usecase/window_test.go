package usecase

import (
	"testing"
	"time"

	"github.com/lexora/backend/internal/domain"
)

// weekWindow: reset Senin 00:00 WIB. Uji tiap hari dalam minggu -> from sama (Senin).
func TestWeekWindowMonday(t *testing.T) {
	// 2026-07-27 = Senin. Cek Senin..Minggu semua jatuh ke Senin yang sama.
	monday := time.Date(2026, 7, 27, 0, 0, 0, 0, wib) // Senin 00:00 WIB
	for offset := 0; offset < 7; offset++ {
		day := monday.AddDate(0, 0, offset).Add(13 * time.Hour) // siang tiap hari
		from, to := weekWindow(day)
		if !from.Equal(monday) {
			t.Errorf("hari +%d: from = %v, mau Senin %v", offset, from, monday)
		}
		if !to.Equal(monday.AddDate(0, 0, 7)) {
			t.Errorf("hari +%d: to harus Senin depan, dapat %v", offset, to)
		}
	}
}

// edge: Minggu 23:59 masih minggu ini; Senin 00:00 sudah minggu baru.
func TestWeekWindowSundayEdge(t *testing.T) {
	monday := time.Date(2026, 7, 27, 0, 0, 0, 0, wib)
	sunday2359 := monday.AddDate(0, 0, 6).Add(23*time.Hour + 59*time.Minute)
	if from, _ := weekWindow(sunday2359); !from.Equal(monday) {
		t.Errorf("Minggu 23:59 harus masih minggu ini (Senin %v), dapat %v", monday, from)
	}
	nextMon := monday.AddDate(0, 0, 7)
	if from, _ := weekWindow(nextMon); !from.Equal(nextMon) {
		t.Errorf("Senin depan 00:00 harus minggu baru %v, dapat %v", nextMon, from)
	}
}

// sessionStart: nil -> mulai now; idle >5j -> mulai now; dalam 5j -> lanjut.
func TestSessionStart(t *testing.T) {
	now := time.Date(2026, 7, 30, 12, 0, 0, 0, wib)

	// nil -> session baru = now
	if got := sessionStart(now, nil); !got.Equal(now) {
		t.Errorf("nil start -> now, dapat %v", got)
	}
	// mulai 2 jam lalu (dalam 5j) -> lanjut yang lama
	old := now.Add(-2 * time.Hour)
	if got := sessionStart(now, &old); !got.Equal(old) {
		t.Errorf("dalam 5j harus lanjut %v, dapat %v", old, got)
	}
	// mulai 6 jam lalu (idle >5j) -> session baru = now
	stale := now.Add(-6 * time.Hour)
	if got := sessionStart(now, &stale); !got.Equal(now) {
		t.Errorf("idle >5j harus reset ke now, dapat %v", got)
	}
	// tepat 5 jam (batas) -> masih lanjut (bukan > 5j)
	exactly := now.Add(-domain.SessionWindow)
	if got := sessionStart(now, &exactly); !got.Equal(exactly) {
		t.Errorf("tepat 5j harus lanjut (bukan idle), dapat %v", got)
	}
}

func TestSessionExpired(t *testing.T) {
	now := time.Date(2026, 7, 30, 12, 0, 0, 0, wib)
	within := now.Add(-2 * time.Hour)
	stale := now.Add(-6 * time.Hour)
	if sessionExpired(now, nil) != true {
		t.Error("nil harus expired")
	}
	if sessionExpired(now, &within) != false {
		t.Error("dalam 5j tak boleh expired")
	}
	if sessionExpired(now, &stale) != true {
		t.Error("idle >5j harus expired")
	}
}

// windowBounds session: pakai sessionStart + durasi 5j
func TestWindowBoundsSession(t *testing.T) {
	now := time.Date(2026, 7, 30, 12, 0, 0, 0, wib)
	start := now.Add(-1 * time.Hour)
	from, reset := windowBounds(WindowSession, now, &start)
	if !from.Equal(start) {
		t.Errorf("session from = start %v, dapat %v", start, from)
	}
	if !reset.Equal(start.Add(domain.SessionWindow)) {
		t.Errorf("session reset = start+5j, dapat %v", reset)
	}
}

// WindowUsage: active/hard/soft/remaining
func TestWindowUsageStatus(t *testing.T) {
	// nonaktif (limit 0)
	off := WindowUsage{Limit: 0, Used: 999}
	if off.Active() || off.Hard() || off.Soft() || off.Remaining() != -1 {
		t.Error("limit 0 harus nonaktif: tak active/hard/soft, remaining -1")
	}
	// 80% -> soft, belum hard
	soft := WindowUsage{Limit: 1000, Used: 800}
	if !soft.Soft() || soft.Hard() {
		t.Errorf("800/1000 harus soft belum hard: soft=%v hard=%v", soft.Soft(), soft.Hard())
	}
	// 100% -> hard, remaining 0
	hard := WindowUsage{Limit: 1000, Used: 1000}
	if !hard.Hard() || hard.Remaining() != 0 {
		t.Errorf("1000/1000 harus hard, remaining 0: hard=%v rem=%d", hard.Hard(), hard.Remaining())
	}
	// 50% -> tidak soft/hard, remaining 500
	mid := WindowUsage{Limit: 1000, Used: 500}
	if mid.Soft() || mid.Hard() || mid.Remaining() != 500 {
		t.Errorf("500/1000: rem=%d soft=%v hard=%v", mid.Remaining(), mid.Soft(), mid.Hard())
	}
}
