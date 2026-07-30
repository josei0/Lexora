package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

const (
	AuditLoginOK        = "login.ok"
	AuditLoginFail      = "login.fail"
	AuditAdminLoginOK   = "admin.login.ok"
	AuditAdminLoginFail = "admin.login.fail"
	AuditAdminMFAEnroll = "admin.mfa_enroll"
	AuditLogout         = "logout"
	AuditPasswordChange = "password.change"
	AuditOrgCreate      = "org.create"
	AuditMemberAdd      = "member.add"
	AuditMemberUpdate   = "member.update"
	AuditDocUpload      = "document.upload"
	AuditKBWebIngest    = "kb.web_ingest"
	AuditSubAssign      = "subscription.assign"
	AuditInvoicePaid    = "billing.invoice_paid"
	AuditManualPaid     = "billing.manual_paid"
	AuditPromptUpdate   = "prompt.update"
	AuditRegister       = "auth.register"   // self-serve register (update6)
	AuditGoogleAuth     = "auth.google"     // login/register via Google (update6)
	AuditWebhookPaid    = "webhook.paid"    // pelunasan via webhook Xendit (update6)
)

// satu baris audit. Tanpa kolom bebas isi — hindari kebocoran data sensitif.
type AuditLog struct {
	ID         uuid.UUID  `json:"id"`
	OrgID      *uuid.UUID `json:"org_id,omitempty"`
	UserID     *uuid.UUID `json:"user_id,omitempty"`
	Action     string     `json:"action"`
	ResourceID *uuid.UUID `json:"resource_id,omitempty"`
	IP         string     `json:"ip"`
	CreatedAt  time.Time  `json:"created_at"`
}

type AuditRepository interface {
	Insert(ctx context.Context, e AuditLog) error
	Recent(ctx context.Context, limit int) ([]AuditLog, error)
}
