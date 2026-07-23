package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// action codes untuk audit log
const (
	AuditLoginOK        = "login.ok"
	AuditLoginFail      = "login.fail"
	AuditLogout         = "logout"
	AuditPasswordChange = "password.change"
	AuditOrgCreate      = "org.create"
	AuditMemberAdd      = "member.add"
	AuditMemberUpdate   = "member.update"
	AuditDocUpload      = "document.upload"
	AuditSubAssign      = "subscription.assign"
	AuditPromptUpdate   = "prompt.update"
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
	// terbaru dulu, dibatasi limit
	Recent(ctx context.Context, limit int) ([]AuditLog, error)
}
