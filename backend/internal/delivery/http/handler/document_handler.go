package handler

import (
	"context"
	"io"
	"mime/multipart"

	"github.com/google/uuid"

	"github.com/lexora/backend/internal/delivery/http/dto"
	"github.com/lexora/backend/internal/delivery/http/middleware"
	"github.com/lexora/backend/internal/domain"
	"github.com/lexora/backend/internal/usecase"
)

// POST /documents (org admin) - multipart upload, returns 202
func (a *API) UploadDocument(ctx context.Context, req dto.UploadDocumentRequestObject) (dto.UploadDocumentResponseObject, error) {
	id, ok := middleware.IdentityFrom(ctx)
	if !ok || !id.IsOrgAdmin() {
		return dto.UploadDocument403JSONResponse(errBody("forbidden", "akses ditolak")), nil
	}
	if req.Body == nil {
		return dto.UploadDocument400JSONResponse{ErrorResponseJSONResponse: dto.ErrorResponseJSONResponse(errBody("invalid_request", "file wajib"))}, nil
	}

	fileName, data, err := readFilePart(req.Body)
	if err != nil {
		return dto.UploadDocument400JSONResponse{ErrorResponseJSONResponse: dto.ErrorResponseJSONResponse(errBody("invalid_request", "gagal baca file"))}, nil
	}

	doc, err := a.docs.Upload(ctx, id.OrgID, id.UserID, fileName, data)
	if err != nil {
		switch err {
		case domain.ErrTooLarge:
			return dto.UploadDocument413JSONResponse(errBody("too_large", "file melebihi 20MB")), nil
		case domain.ErrUnsupportedType, domain.ErrInvalidUpload:
			return dto.UploadDocument400JSONResponse{ErrorResponseJSONResponse: dto.ErrorResponseJSONResponse(errBody("unsupported", "tipe file tidak didukung (PDF, DOCX, TXT)"))}, nil
		}
		return nil, err
	}
	return dto.UploadDocument202JSONResponse(toDocument(doc)), nil
}

// GET /documents (own org, paginated)
func (a *API) ListDocuments(ctx context.Context, req dto.ListDocumentsRequestObject) (dto.ListDocumentsResponseObject, error) {
	id, ok := middleware.IdentityFrom(ctx)
	if !ok || id.OrgID == uuid.Nil {
		// need org context (super admin has none for KB)
		return dto.ListDocuments403JSONResponse{ErrorResponseJSONResponse: dto.ErrorResponseJSONResponse(errBody("forbidden", "akses ditolak"))}, nil
	}
	limit, offset := 50, 0
	if req.Params.Limit != nil {
		limit = *req.Params.Limit
	}
	if req.Params.Offset != nil {
		offset = *req.Params.Offset
	}
	list, err := a.docs.List(ctx, id.OrgID, limit, offset)
	if err != nil {
		return nil, err
	}
	out := make(dto.ListDocuments200JSONResponse, len(list))
	for i, d := range list {
		out[i] = toDocument(&d)
	}
	return out, nil
}

// read first file part from multipart reader (bounded)
func readFilePart(mr *multipart.Reader) (string, []byte, error) {
	for {
		part, err := mr.NextPart()
		if err != nil {
			return "", nil, err
		}
		if part.FormName() == "file" {
			name := part.FileName()
			data, err := io.ReadAll(io.LimitReader(part, usecase.MaxUploadBytes+1))
			part.Close()
			return name, data, err
		}
		part.Close()
	}
}

func toDocument(d *domain.Document) dto.Document {
	scope := dto.DocumentScope(d.Scope)
	return dto.Document{
		Id:        d.ID,
		FileName:  d.FileName,
		MimeType:  d.MimeType,
		FileSize:  d.FileSize,
		Scope:     &scope,
		Status:    dto.DocumentStatus(d.Status),
		Error:     d.Error,
		CreatedAt: d.CreatedAt,
	}
}
