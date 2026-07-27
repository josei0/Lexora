package postgres

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lexora/backend/internal/domain"
)

type WebSearchRepo struct{ db *pgxpool.Pool }

func NewWebSearchRepo(db *pgxpool.Pool) *WebSearchRepo { return &WebSearchRepo{db} }

func (r *WebSearchRepo) Log(ctx context.Context, s domain.WebSearch) error {
	urls, err := json.Marshal(s.TopURLs)
	if err != nil {
		return err
	}
	_, err = r.db.Exec(ctx, `
		insert into web_searches (organization_id, user_id, query, provider, results_count, top_urls)
		values ($1, $2, $3, $4, $5, $6)`,
		s.OrganizationID, s.UserID, s.Query, s.Provider, s.ResultsCount, urls)
	return err
}

func (r *WebSearchRepo) CountToday(ctx context.Context, orgID, userID uuid.UUID, from time.Time) (int, error) {
	var n int
	err := r.db.QueryRow(ctx, `
		select count(*) from web_searches
		where organization_id = $1 and user_id = $2 and created_at >= $3`,
		orgID, userID, from).Scan(&n)
	return n, err
}

// retensi: query = pertanyaan hukum user (perkara klien), jangan disimpan selamanya
func (r *WebSearchRepo) DeleteOlderThan(ctx context.Context, t time.Time) (int64, error) {
	tag, err := r.db.Exec(ctx, `delete from web_searches where created_at < $1`, t)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}

func (r *WebSearchRepo) Candidates(ctx context.Context, orgID uuid.UUID, minHits int, since time.Time) ([]domain.WebCandidate, error) {
	rows, err := r.db.Query(ctx, `
		select url, count(*) as hits, max(created_at) as last_at
		from web_searches ws, jsonb_array_elements_text(ws.top_urls) as url
		where ws.organization_id = $1
		  and ws.created_at > $2
		  and url not in (
		    select source_url from documents
		    where organization_id = $1 and source_url is not null
		  )
		group by url having count(*) >= $3
		order by hits desc`,
		orgID, since, minHits)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.WebCandidate
	for rows.Next() {
		var c domain.WebCandidate
		if err := rows.Scan(&c.URL, &c.Hits, &c.LastAt); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}
