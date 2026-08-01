package usecase

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/lexora/backend/internal/domain"
)

// fake reminder repo: simpan kandidat + rekam MarkReminded ke state (cermin DB)
// sehingga panggilan SendDue kedua di hari sama melihat last_reminder_at terisi.
type fakeReminderRepo struct {
	cands  []domain.ReminderCandidate
	marked map[uuid.UUID]time.Time
}

func (f *fakeReminderRepo) DueReminders(_ context.Context, _ time.Time) ([]domain.ReminderCandidate, error) {
	out := make([]domain.ReminderCandidate, len(f.cands))
	copy(out, f.cands)
	for i := range out {
		if t, ok := f.marked[out[i].OrganizationID]; ok {
			tt := t
			out[i].LastReminderAt = &tt // cerminkan mark
		}
	}
	return out, nil
}

func (f *fakeReminderRepo) MarkReminded(_ context.Context, orgID uuid.UUID, at time.Time) error {
	if f.marked == nil {
		f.marked = map[uuid.UUID]time.Time{}
	}
	f.marked[orgID] = at
	return nil
}

// mailer yang bisa dimatikan (D5): Enabled ikut flag.
type toggleMailer struct {
	fakeMailer
	on bool
}

func (m *toggleMailer) Enabled() bool { return m.on }

// kandidat di titik `offsetDays` relatif now, di batas hari (daysUntil eksak).
func candAt(now time.Time, offsetDays int) domain.ReminderCandidate {
	end := truncDay(now).AddDate(0, 0, offsetDays)
	return domain.ReminderCandidate{
		OrganizationID: uuid.New(), PeriodEnd: end,
		AdminEmail: "admin@firma.id", AdminName: "Admin", OrgName: "Firma X", PlanName: "Pro",
	}
}

// D1: kandidat per titik tanggal (H-7/H-1/H+3) -> subject cocok; non-due -> skip.
func TestD1DueKindPerDatePoint(t *testing.T) {
	now := time.Date(2026, 8, 1, 9, 0, 0, 0, time.UTC)
	repo := &fakeReminderRepo{cands: []domain.ReminderCandidate{
		candAt(now, 7), candAt(now, 1), candAt(now, -3), candAt(now, -1), // -1 = bukan titik due
	}}
	mail := &toggleMailer{on: true}
	uc := NewDunning(repo, mail, "https://app.test")

	n, err := uc.SendDue(context.Background(), now)
	if err != nil {
		t.Fatal(err)
	}
	if n != 3 {
		t.Fatalf("mau 3 email (H-7/H-1/H+3), non-due di-skip; dapat %d", n)
	}
	joined := strings.Join(subjectsOf(mail), " | ")
	for _, want := range []string{"7 hari", "besok", "sudah berakhir"} {
		if !strings.Contains(joined, want) {
			t.Errorf("subject %q hilang; sent=%q", want, joined)
		}
	}
}

// D2: H-7 kirim tepat 1 email ke org_admin, body punya link Tagihan + tanggal.
func TestD2SendsAtH7(t *testing.T) {
	now := time.Date(2026, 8, 1, 9, 0, 0, 0, time.UTC)
	repo := &fakeReminderRepo{cands: []domain.ReminderCandidate{candAt(now, 7)}}
	mail := &toggleMailer{on: true}
	uc := NewDunning(repo, mail, "https://app.test")

	n, err := uc.SendDue(context.Background(), now)
	if err != nil || n != 1 {
		t.Fatalf("H-7 harus kirim 1: n=%d err=%v", n, err)
	}
	if mail.sent[0].to != "admin@firma.id" {
		t.Fatalf("tujuan salah: %s", mail.sent[0].to)
	}
	if !strings.Contains(mail.sent[0].body, "https://app.test/app/billing") {
		t.Errorf("body harus punya link Tagihan: %q", mail.sent[0].body)
	}
	if !strings.Contains(mail.sent[0].body, "8 Agustus 2026") {
		t.Errorf("body harus punya tanggal jatuh tempo: %q", mail.sent[0].body)
	}
}

// D3: idempoten - jalankan 2x di hari yg sama -> total tetap 1 email.
func TestD3IdempotentSameDay(t *testing.T) {
	now := time.Date(2026, 8, 1, 9, 0, 0, 0, time.UTC)
	repo := &fakeReminderRepo{cands: []domain.ReminderCandidate{candAt(now, 7)}}
	mail := &toggleMailer{on: true}
	uc := NewDunning(repo, mail, "https://app.test")

	if n, _ := uc.SendDue(context.Background(), now); n != 1 {
		t.Fatalf("run pertama harus 1, dapat %d", n)
	}
	later := now.Add(2 * time.Hour) // masih hari sama
	if n, _ := uc.SendDue(context.Background(), later); n != 0 {
		t.Fatalf("run kedua hari sama harus 0 (idempoten), dapat %d", n)
	}
	if len(mail.sent) != 1 {
		t.Fatalf("total email harus 1, dapat %d", len(mail.sent))
	}
}

// D4: past_due H+3 kirim reminder (period_end 3 hari lalu).
func TestD4PastDueH3(t *testing.T) {
	now := time.Date(2026, 8, 1, 9, 0, 0, 0, time.UTC)
	repo := &fakeReminderRepo{cands: []domain.ReminderCandidate{candAt(now, -3)}}
	mail := &toggleMailer{on: true}
	uc := NewDunning(repo, mail, "https://app.test")

	n, err := uc.SendDue(context.Background(), now)
	if err != nil || n != 1 {
		t.Fatalf("H+3 past_due harus kirim 1: n=%d err=%v", n, err)
	}
	if !strings.Contains(mail.sent[0].body, "masa tenggang") {
		t.Errorf("body past_due harus sebut masa tenggang: %q", mail.sent[0].body)
	}
}

// D5: SMTP kosong (Enabled false) -> no-op, nol email, tanpa error.
func TestD5NoSMTPNoOp(t *testing.T) {
	now := time.Date(2026, 8, 1, 9, 0, 0, 0, time.UTC)
	repo := &fakeReminderRepo{cands: []domain.ReminderCandidate{candAt(now, 7)}}
	mail := &toggleMailer{on: false} // SMTP kosong
	uc := NewDunning(repo, mail, "https://app.test")

	n, err := uc.SendDue(context.Background(), now)
	if err != nil {
		t.Fatalf("SMTP kosong tak boleh error: %v", err)
	}
	if n != 0 || len(mail.sent) != 0 {
		t.Fatalf("SMTP kosong harus no-op: n=%d sent=%d", n, len(mail.sent))
	}
}

func subjectsOf(m *toggleMailer) []string {
	var out []string
	for _, s := range m.sent {
		out = append(out, s.subject)
	}
	return out
}
