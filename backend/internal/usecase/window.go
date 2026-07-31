package usecase

import (
	"time"

	"github.com/lexora/backend/internal/domain"
)

// WindowKind: jenis window limit gaya Claude (update8).
type WindowKind string

const (
	WindowSession WindowKind = "session"
	WindowWeekly  WindowKind = "weekly"
	WindowMonthly WindowKind = "monthly"
)

// WindowUsage: pemakaian satu window. Limit 0 = window nonaktif (tak membatasi).
type WindowUsage struct {
	Kind    WindowKind
	Limit   int64     // 0 = nonaktif untuk plan ini
	Used    int64     // token terpakai dalam window
	ResetAt time.Time // kapan window ini reset (untuk pesan "reset dalam X")
}

// Active: window ikut menggating (limit > 0).
func (w WindowUsage) Active() bool { return w.Limit > 0 }

// Hard: window mentok (>= limit). Nonaktif tak pernah hard.
func (w WindowUsage) Hard() bool { return w.Active() && w.Used >= w.Limit }

// Soft: >= 80% (peringatan). Nonaktif tak pernah soft.
func (w WindowUsage) Soft() bool {
	return w.Active() && w.Used >= int64(float64(w.Limit)*softThreshold)
}

// Remaining: sisa token; -1 kalau nonaktif.
func (w WindowUsage) Remaining() int64 {
	if !w.Active() {
		return -1
	}
	if w.Used >= w.Limit {
		return 0
	}
	return w.Limit - w.Used
}

// windowBounds: [from, resetAt) untuk tiap window pada saat `now`, plus session start.
// session: dari sessionStartedAt (nil/idle>5j -> mulai now). weekly/monthly: kalender WIB.
func windowBounds(kind WindowKind, now time.Time, sessionStartedAt *time.Time) (from, resetAt time.Time) {
	switch kind {
	case WindowSession:
		start := sessionStart(now, sessionStartedAt)
		return start, start.Add(domain.SessionWindow)
	case WindowWeekly:
		return weekWindow(now)
	default: // monthly
		return monthWindow(now)
	}
}

// sessionStart: kapan window session sekarang mulai (update8 §2.2).
// nil atau idle > 5 jam -> session baru mulai `now`. Selain itu lanjut yang lama.
func sessionStart(now time.Time, startedAt *time.Time) time.Time {
	if startedAt == nil || now.Sub(*startedAt) > domain.SessionWindow {
		return now
	}
	return *startedAt
}

// sessionExpired: true kalau session lama sudah lewat (perlu di-reset ke now saat pesan berikut).
func sessionExpired(now time.Time, startedAt *time.Time) bool {
	return startedAt == nil || now.Sub(*startedAt) > domain.SessionWindow
}
