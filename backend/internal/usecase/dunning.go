package usecase

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/google/uuid"
	"github.com/lexora/backend/internal/domain"
)

// ReminderRepo: irisan sempit yg dibutuhkan dunning (interface segregation) -
// impl konkret di postgres.InvoiceRepo. Fake test tak perlu seluruh InvoiceRepository.
type ReminderRepo interface {
	DueReminders(ctx context.Context, now time.Time) ([]domain.ReminderCandidate, error)
	MarkReminded(ctx context.Context, orgID uuid.UUID, at time.Time) error
}

// Dunning: kirim email reminder langganan di titik H-7/H-1 (mendatang) + H+3
// (past_due). Reuse Mailer (no-op kalau SMTP kosong). baseURL untuk link Tagihan.
type Dunning struct {
	repo    ReminderRepo
	mail    Mailer
	baseURL string
}

func NewDunning(repo ReminderRepo, mail Mailer, baseURL string) *Dunning {
	return &Dunning{repo: repo, mail: mail, baseURL: baseURL}
}

// SendDue: satu email per kandidat due. Skip kalau SMTP mati atau sudah
// diingatkan hari yang sama (anti-dobel). Balikan = jumlah email terkirim.
func (d *Dunning) SendDue(ctx context.Context, now time.Time) (int, error) {
	if d.mail == nil || !d.mail.Enabled() {
		return 0, nil // no-op tanpa error (dev tanpa SMTP)
	}
	cands, err := d.repo.DueReminders(ctx, now)
	if err != nil {
		return 0, err
	}
	sent := 0
	for _, c := range cands {
		// idempoten: sudah diingatkan hari ini -> lewati
		if c.LastReminderAt != nil && sameDay(*c.LastReminderAt, now) {
			continue
		}
		subject, body := d.render(c, now)
		if subject == "" {
			continue // bukan titik due (defensif)
		}
		if err := d.mail.Send(c.AdminEmail, subject, body); err != nil {
			return sent, err
		}
		if err := d.repo.MarkReminded(ctx, c.OrganizationID, now); err != nil {
			return sent, err
		}
		sent++
	}
	return sent, nil
}

// render: subject+body per titik due. Kosong = bukan H-7/H-1/H+3.
func (d *Dunning) render(c domain.ReminderCandidate, now time.Time) (subject, body string) {
	link := d.baseURL + "/app/billing"
	due := fmtDate(c.PeriodEnd)
	switch daysUntil(now, c.PeriodEnd) {
	case 7:
		subject = "Langganan MindLaw " + c.OrgName + " berakhir dalam 7 hari"
		body = fmt.Sprintf("Halo %s,\n\nLangganan %s (%s) berakhir pada %s. Perpanjang lebih awal agar layanan tak terputus.\n\nKelola di: %s\n\nTerima kasih.",
			c.AdminName, c.OrgName, c.PlanName, due, link)
	case 1:
		subject = "Langganan MindLaw " + c.OrgName + " berakhir besok"
		body = fmt.Sprintf("Halo %s,\n\nLangganan %s (%s) berakhir besok, %s. Segera perpanjang agar akses tetap aktif.\n\nKelola di: %s\n\nTerima kasih.",
			c.AdminName, c.OrgName, c.PlanName, due, link)
	case -3:
		subject = "Langganan MindLaw " + c.OrgName + " sudah berakhir"
		body = fmt.Sprintf("Halo %s,\n\nLangganan %s (%s) berakhir pada %s dan kini dalam masa tenggang. Perpanjang sekarang sebelum akses dinonaktifkan.\n\nKelola di: %s\n\nTerima kasih.",
			c.AdminName, c.OrgName, c.PlanName, due, link)
	}
	return subject, body
}

// daysUntil: selisih hari kalender end-now (bulatkan; tahan DST 23/25 jam).
func daysUntil(now, end time.Time) int {
	d1 := truncDay(now)
	d2 := truncDay(end)
	return int(math.Round(d2.Sub(d1).Hours() / 24))
}

func truncDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}

func sameDay(a, b time.Time) bool {
	return a.Year() == b.Year() && a.Month() == b.Month() && a.Day() == b.Day()
}

// bulan ID untuk tanggal email (locale-free, cukup untuk lokalisasi ID).
var bulanID = [...]string{"", "Januari", "Februari", "Maret", "April", "Mei", "Juni",
	"Juli", "Agustus", "September", "Oktober", "November", "Desember"}

func fmtDate(t time.Time) string {
	return fmt.Sprintf("%d %s %d", t.Day(), bulanID[t.Month()], t.Year())
}
