package usecase

import (
	"context"
	"log"

	"github.com/google/uuid"
	"github.com/lexora/backend/internal/domain"
)

type Audit struct{ repo domain.AuditRepository }

func NewAudit(repo domain.AuditRepository) *Audit { return &Audit{repo} }

// Record logs an audit entry; never fails the caller.
func (a *Audit) Record(ctx context.Context, action string, orgID, userID, resourceID *uuid.UUID, ip string) {
	e := domain.AuditLog{Action: action, IP: ip, OrgID: orgID, UserID: userID, ResourceID: resourceID}
	if err := a.repo.Insert(ctx, e); err != nil {
		log.Printf("audit insert: %v", err)
	}
}

func (a *Audit) Recent(ctx context.Context, limit int) ([]domain.AuditLog, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	return a.repo.Recent(ctx, limit)
}
