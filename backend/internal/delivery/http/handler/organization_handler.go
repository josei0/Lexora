package handler

import (
	"context"

	"github.com/google/uuid"
	"github.com/lexora/backend/internal/delivery/http/dto"
	"github.com/lexora/backend/internal/delivery/http/middleware"
	"github.com/lexora/backend/internal/domain"
)

// POST /organizations (super admin only)
func (a *API) CreateOrganization(ctx context.Context, req dto.CreateOrganizationRequestObject) (dto.CreateOrganizationResponseObject, error) {
	id, ok := middleware.IdentityFrom(ctx)
	if !ok || !id.IsSuperAdmin() {
		return dto.CreateOrganization403JSONResponse{ErrorResponseJSONResponse: dto.ErrorResponseJSONResponse(errBody("forbidden", "akses ditolak"))}, nil
	}
	if req.Body == nil {
		return dto.CreateOrganization403JSONResponse{ErrorResponseJSONResponse: dto.ErrorResponseJSONResponse(errBody("invalid_request", "data tidak lengkap"))}, nil
	}
	org, admin, err := a.org.Create(ctx, req.Body.Name, req.Body.Slug, string(req.Body.AdminEmail), req.Body.AdminFullName)
	if err != nil {
		if err == domain.ErrConflict {
			return dto.CreateOrganization409JSONResponse(errBody("conflict", "slug atau email sudah dipakai")), nil
		}
		return nil, err
	}
	uid := id.UserID
	orgID := org.ID
	a.audit.Record(ctx, domain.AuditOrgCreate, &orgID, &uid, &orgID, middleware.ClientIPFrom(ctx))
	return dto.CreateOrganization201JSONResponse{
		Organization: dto.Organization{Id: org.ID, Name: org.Name, Slug: org.Slug, CreatedAt: org.CreatedAt},
		Admin:        dto.AddMemberResponse{UserId: admin.UserID, Email: admin.Email, TempPassword: admin.TempPassword},
	}, nil
}

// GET /organizations (super admin only)
func (a *API) ListOrganizations(ctx context.Context, _ dto.ListOrganizationsRequestObject) (dto.ListOrganizationsResponseObject, error) {
	id, ok := middleware.IdentityFrom(ctx)
	if !ok || !id.IsSuperAdmin() {
		return dto.ListOrganizations403JSONResponse{ErrorResponseJSONResponse: dto.ErrorResponseJSONResponse(errBody("forbidden", "akses ditolak"))}, nil
	}
	orgs, err := a.org.List(ctx)
	if err != nil {
		return nil, err
	}
	out := make(dto.ListOrganizations200JSONResponse, len(orgs))
	for i, o := range orgs {
		out[i] = dto.Organization{Id: o.ID, Name: o.Name, Slug: o.Slug, CreatedAt: o.CreatedAt}
	}
	return out, nil
}

// POST /members (org admin only)
func (a *API) AddMember(ctx context.Context, req dto.AddMemberRequestObject) (dto.AddMemberResponseObject, error) {
	id, ok := middleware.IdentityFrom(ctx)
	if !ok || !id.IsOrgAdmin() {
		return dto.AddMember403JSONResponse{ErrorResponseJSONResponse: dto.ErrorResponseJSONResponse(errBody("forbidden", "akses ditolak"))}, nil
	}
	if req.Body == nil {
		return dto.AddMember403JSONResponse{ErrorResponseJSONResponse: dto.ErrorResponseJSONResponse(errBody("invalid_request", "data tidak lengkap"))}, nil
	}
	// org from jwt - anti-IDOR
	m, err := a.org.AddMember(ctx, id.OrgID, string(req.Body.Email), req.Body.FullName, string(req.Body.Role))
	if err != nil {
		if err == domain.ErrConflict {
			return dto.AddMember409JSONResponse(errBody("conflict", "email sudah terdaftar")), nil
		}
		if err == domain.ErrSeatsFull {
			return dto.AddMember409JSONResponse(errBody("seats_full", "jumlah seat langganan sudah penuh")), nil
		}
		if err == domain.ErrForbidden {
			return dto.AddMember403JSONResponse{ErrorResponseJSONResponse: dto.ErrorResponseJSONResponse(errBody("invalid_role", "role tidak valid"))}, nil
		}
		return nil, err
	}
	uid := id.UserID
	orgID := id.OrgID
	newUID := m.UserID
	a.audit.Record(ctx, domain.AuditMemberAdd, &orgID, &uid, &newUID, middleware.ClientIPFrom(ctx))
	return dto.AddMember201JSONResponse{UserId: m.UserID, Email: m.Email, TempPassword: m.TempPassword}, nil
}

// GET /members (org admin only)
func (a *API) ListMembers(ctx context.Context, _ dto.ListMembersRequestObject) (dto.ListMembersResponseObject, error) {
	id, ok := middleware.IdentityFrom(ctx)
	if !ok || !id.IsOrgAdmin() {
		return dto.ListMembers403JSONResponse{ErrorResponseJSONResponse: dto.ErrorResponseJSONResponse(errBody("forbidden", "akses ditolak"))}, nil
	}
	members, err := a.org.ListMembers(ctx, id.OrgID)
	if err != nil {
		return nil, err
	}
	out := make(dto.ListMembers200JSONResponse, len(members))
	for i, m := range members {
		out[i] = toMember(m)
	}
	return out, nil
}

// PATCH /members/{userId} (org admin only)
func (a *API) UpdateMember(ctx context.Context, req dto.UpdateMemberRequestObject) (dto.UpdateMemberResponseObject, error) {
	id, ok := middleware.IdentityFrom(ctx)
	if !ok || !id.IsOrgAdmin() {
		return dto.UpdateMember403JSONResponse{ErrorResponseJSONResponse: dto.ErrorResponseJSONResponse(errBody("forbidden", "akses ditolak"))}, nil
	}
	var role *string
	var active *bool
	if req.Body != nil {
		if req.Body.Role != nil {
			s := string(*req.Body.Role)
			role = &s
		}
		active = req.Body.IsActive
	}
	// scoped to caller org - anti-IDOR
	m, err := a.org.UpdateMember(ctx, id.OrgID, uuid.UUID(req.UserId), role, active)
	if err != nil {
		if err == domain.ErrNotFound {
			return dto.UpdateMember404JSONResponse(errBody("not_found", "anggota tidak ditemukan")), nil
		}
		if err == domain.ErrForbidden {
			return dto.UpdateMember403JSONResponse{ErrorResponseJSONResponse: dto.ErrorResponseJSONResponse(errBody("invalid_role", "role tidak valid"))}, nil
		}
		return nil, err
	}
	uid := id.UserID
	orgID := id.OrgID
	targetUID := m.UserID
	a.audit.Record(ctx, domain.AuditMemberUpdate, &orgID, &uid, &targetUID, middleware.ClientIPFrom(ctx))
	return dto.UpdateMember200JSONResponse(toMember(*m)), nil
}

func toMember(m domain.Member) dto.Member {
	return dto.Member{
		UserId:   m.UserID,
		Email:    m.Email,
		FullName: m.FullName,
		Role:     dto.MemberRole(m.Role),
		IsActive: m.IsActive,
	}
}
