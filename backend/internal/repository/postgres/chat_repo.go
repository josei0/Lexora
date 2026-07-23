package postgres

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lexora/backend/internal/domain"
)

type ChatRepo struct{ db *pgxpool.Pool }

func NewChatRepo(db *pgxpool.Pool) *ChatRepo { return &ChatRepo{db} }

func (r *ChatRepo) Create(ctx context.Context, c *domain.Chat) error {
	err := r.db.QueryRow(ctx, `
		insert into chats (organization_id, user_id, title)
		values ($1, $2, $3) returning id, is_pinned, created_at, updated_at`,
		c.OrganizationID, c.UserID, c.Title,
	).Scan(&c.ID, &c.IsPinned, &c.CreatedAt, &c.UpdatedAt)
	return mapErr(err)
}

func (r *ChatRepo) ByID(ctx context.Context, id, orgID, userID uuid.UUID) (*domain.Chat, error) {
	return scanChat(r.db.QueryRow(ctx, `
		select id, organization_id, user_id, title, is_pinned, created_at, updated_at
		from chats
		where id = $1 and organization_id = $2 and user_id = $3 and deleted_at is null`,
		id, orgID, userID))
}

func (r *ChatRepo) ListByUser(ctx context.Context, orgID, userID uuid.UUID, search string, limit, offset int) ([]domain.Chat, error) {
	rows, err := r.db.Query(ctx, `
		select id, organization_id, user_id, title, is_pinned, created_at, updated_at
		from chats
		where organization_id = $1 and user_id = $2 and deleted_at is null
		  and ($3 = '' or title ilike '%' || $3 || '%')
		order by is_pinned desc, updated_at desc
		limit $4 offset $5`, orgID, userID, search, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []domain.Chat
	for rows.Next() {
		c, err := scanChat(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *c)
	}
	return out, rows.Err()
}

func (r *ChatRepo) Rename(ctx context.Context, id, orgID, userID uuid.UUID, title string) error {
	return r.affect(ctx, `
		update chats set title = $4, updated_at = now()
		where id = $1 and organization_id = $2 and user_id = $3 and deleted_at is null`,
		id, orgID, userID, title)
}

func (r *ChatRepo) SetPinned(ctx context.Context, id, orgID, userID uuid.UUID, pinned bool) error {
	return r.affect(ctx, `
		update chats set is_pinned = $4, updated_at = now()
		where id = $1 and organization_id = $2 and user_id = $3 and deleted_at is null`,
		id, orgID, userID, pinned)
}

func (r *ChatRepo) SoftDelete(ctx context.Context, id, orgID, userID uuid.UUID) error {
	return r.affect(ctx, `
		update chats set deleted_at = now()
		where id = $1 and organization_id = $2 and user_id = $3 and deleted_at is null`,
		id, orgID, userID)
}

func (r *ChatRepo) Touch(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, `update chats set updated_at = now() where id = $1`, id)
	return err
}

func (r *ChatRepo) AddMessage(ctx context.Context, m *domain.Message) error {
	err := r.db.QueryRow(ctx, `
		insert into messages (chat_id, role, content, model)
		values ($1, $2, $3, $4) returning id, created_at`,
		m.ChatID, m.Role, m.Content, m.Model,
	).Scan(&m.ID, &m.CreatedAt)
	return mapErr(err)
}

func (r *ChatRepo) Messages(ctx context.Context, chatID uuid.UUID) ([]domain.Message, error) {
	rows, err := r.db.Query(ctx, `
		select id, chat_id, role, content, model, created_at
		from messages where chat_id = $1 order by created_at`, chatID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var msgs []domain.Message
	byID := map[uuid.UUID]int{}
	for rows.Next() {
		var m domain.Message
		if err := rows.Scan(&m.ID, &m.ChatID, &m.Role, &m.Content, &m.Model, &m.CreatedAt); err != nil {
			return nil, err
		}
		byID[m.ID] = len(msgs)
		msgs = append(msgs, m)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(msgs) == 0 {
		return msgs, nil
	}

	crows, err := r.db.Query(ctx, `
		select c.id, c.message_id, c.document_chunk_id, c.document_id, c.reference_label, c.marker, c.page_no, c.score
		from citations c join messages m on m.id = c.message_id
		where m.chat_id = $1 order by c.marker`, chatID)
	if err != nil {
		return nil, err
	}
	defer crows.Close()
	for crows.Next() {
		var c domain.Citation
		if err := crows.Scan(&c.ID, &c.MessageID, &c.DocumentChunkID, &c.DocumentID, &c.ReferenceLabel, &c.Marker, &c.PageNo, &c.Score); err != nil {
			return nil, err
		}
		if i, ok := byID[c.MessageID]; ok {
			msgs[i].Citations = append(msgs[i].Citations, c)
		}
	}
	return msgs, crows.Err()
}

func (r *ChatRepo) AddCitations(ctx context.Context, cs []domain.Citation) error {
	if len(cs) == 0 {
		return nil
	}
	batch := &pgx.Batch{}
	for _, c := range cs {
		batch.Queue(`
			insert into citations (message_id, document_chunk_id, document_id, reference_label, marker, page_no, score)
			values ($1, $2, $3, $4, $5, $6, $7)`,
			c.MessageID, c.DocumentChunkID, c.DocumentID, c.ReferenceLabel, c.Marker, c.PageNo, c.Score)
	}
	br := r.db.SendBatch(ctx, batch)
	defer br.Close()
	for range cs {
		if _, err := br.Exec(); err != nil {
			return err
		}
	}
	return nil
}

func (r *ChatRepo) AddUsage(ctx context.Context, u domain.TokenUsage) error {
	_, err := r.db.Exec(ctx, `
		insert into token_usage (organization_id, user_id, message_id, model, input_tokens, output_tokens)
		values ($1, $2, $3, $4, $5, $6)`,
		u.OrganizationID, u.UserID, u.MessageID, u.Model, u.InputTokens, u.OutputTokens)
	return err
}

func (r *ChatRepo) affect(ctx context.Context, sql string, args ...any) error {
	tag, err := r.db.Exec(ctx, sql, args...)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func scanChat(row pgx.Row) (*domain.Chat, error) {
	var c domain.Chat
	err := row.Scan(&c.ID, &c.OrganizationID, &c.UserID, &c.Title, &c.IsPinned, &c.CreatedAt, &c.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, mapErr(err)
	}
	return &c, nil
}
