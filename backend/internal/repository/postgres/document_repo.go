package postgres

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lexora/backend/internal/domain"
)

type DocumentRepo struct{ db *pgxpool.Pool }

func NewDocumentRepo(db *pgxpool.Pool) *DocumentRepo { return &DocumentRepo{db} }

func (r *DocumentRepo) Create(ctx context.Context, d *domain.Document) error {
	return r.db.QueryRow(ctx, `
		insert into documents (organization_id, uploaded_by, scope, file_name, mime_type, file_size, storage_path, status, source_url)
		values ($1,$2,$3,$4,$5,$6,$7,$8,$9) returning id, created_at`,
		d.OrganizationID, d.UploadedBy, d.Scope, d.FileName, d.MimeType, d.FileSize, d.StoragePath, d.Status, d.SourceURL,
	).Scan(&d.ID, &d.CreatedAt)
}

// anti-duplikat ingest web, scoped org
func (r *DocumentRepo) BySourceURL(ctx context.Context, orgID uuid.UUID, url string) (*domain.Document, error) {
	var d domain.Document
	err := r.db.QueryRow(ctx, `
		select id, organization_id, uploaded_by, scope, file_name, mime_type, file_size, storage_path, status, error, source_url, created_at
		from documents where organization_id = $1 and source_url = $2
		limit 1`, orgID, url).Scan(
		&d.ID, &d.OrganizationID, &d.UploadedBy, &d.Scope, &d.FileName, &d.MimeType,
		&d.FileSize, &d.StoragePath, &d.Status, &d.Error, &d.SourceURL, &d.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &d, nil
}

// scoped by org - anti-IDOR
func (r *DocumentRepo) ListByOrg(ctx context.Context, orgID uuid.UUID, limit, offset int) ([]domain.Document, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	rows, err := r.db.Query(ctx, `
		select id, organization_id, uploaded_by, scope, file_name, mime_type, file_size, storage_path, status, error, created_at
		from documents where organization_id = $1 order by created_at desc limit $2 offset $3`,
		orgID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanDocs(rows)
}

// scoped by org - anti-IDOR
func (r *DocumentRepo) ByID(ctx context.Context, id, orgID uuid.UUID) (*domain.Document, error) {
	return scanDoc(r.db.QueryRow(ctx, `
		select id, organization_id, uploaded_by, scope, file_name, mime_type, file_size, storage_path, status, error, created_at
		from documents where id = $1 and organization_id = $2`, id, orgID))
}

// worker-only: fetch by id without org scope (org from row itself)
func (r *DocumentRepo) AnyByID(ctx context.Context, id uuid.UUID) (*domain.Document, error) {
	return scanDoc(r.db.QueryRow(ctx, `
		select id, organization_id, uploaded_by, scope, file_name, mime_type, file_size, storage_path, status, error, created_at
		from documents where id = $1`, id))
}

func (r *DocumentRepo) SetStatus(ctx context.Context, id uuid.UUID, status string, errMsg *string) error {
	_, err := r.db.Exec(ctx, `update documents set status = $2, error = $3, updated_at = now() where id = $1`,
		id, status, errMsg)
	return err
}

// startup recovery: docs stuck uploaded/processing
func (r *DocumentRepo) PendingIDs(ctx context.Context) ([]uuid.UUID, error) {
	rows, err := r.db.Query(ctx, `select id from documents where status in ('uploaded','processing') order by created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func (r *DocumentRepo) InsertChunks(ctx context.Context, chunks []domain.DocumentChunk) error {
	if len(chunks) == 0 {
		return nil
	}
	batch := &pgx.Batch{}
	for _, c := range chunks {
		batch.Queue(`insert into document_chunks (document_id, chunk_index, page_no, qdrant_point_id)
			values ($1,$2,$3,$4)`, c.DocumentID, c.ChunkIndex, c.PageNo, c.QdrantPointID)
	}
	br := r.db.SendBatch(ctx, batch)
	defer br.Close()
	for range chunks {
		if _, err := br.Exec(); err != nil {
			return err
		}
	}
	return nil
}

func (r *DocumentRepo) DeleteChunks(ctx context.Context, documentID uuid.UUID) error {
	_, err := r.db.Exec(ctx, `delete from document_chunks where document_id = $1`, documentID)
	return err
}

func scanDoc(row pgx.Row) (*domain.Document, error) {
	var d domain.Document
	err := row.Scan(&d.ID, &d.OrganizationID, &d.UploadedBy, &d.Scope, &d.FileName, &d.MimeType,
		&d.FileSize, &d.StoragePath, &d.Status, &d.Error, &d.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &d, nil
}

func scanDocs(rows pgx.Rows) ([]domain.Document, error) {
	var out []domain.Document
	for rows.Next() {
		var d domain.Document
		if err := rows.Scan(&d.ID, &d.OrganizationID, &d.UploadedBy, &d.Scope, &d.FileName, &d.MimeType,
			&d.FileSize, &d.StoragePath, &d.Status, &d.Error, &d.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}
