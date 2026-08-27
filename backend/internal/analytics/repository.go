package analytics

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) InsertClick(ctx context.Context, orgID, linkID string, ipHash, country, referrer, userAgent *string) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO click_events (link_id, org_id, ip_hash, country, referrer, user_agent) VALUES ($1, $2, $3, $4, $5, $6)`,
		linkID, orgID, ipHash, country, referrer, userAgent,
	)
	return err
}

type DailyCount struct {
	Day   time.Time `json:"day"`
	Count int64     `json:"count"`
}

func (r *Repository) ClicksByDay(ctx context.Context, orgID string, since time.Time) ([]DailyCount, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT date_trunc('day', created_at) AS day, COUNT(*)
		 FROM click_events WHERE org_id = $1 AND created_at >= $2
		 GROUP BY day ORDER BY day ASC`,
		orgID, since,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []DailyCount
	for rows.Next() {
		var d DailyCount
		if err := rows.Scan(&d.Day, &d.Count); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

type TopReferrer struct {
	Referrer string `json:"referrer"`
	Count    int64  `json:"count"`
}

func (r *Repository) TopReferrers(ctx context.Context, orgID string, since time.Time, limit int) ([]TopReferrer, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT COALESCE(NULLIF(referrer, ''), 'direct') AS referrer, COUNT(*)
		 FROM click_events WHERE org_id = $1 AND created_at >= $2
		 GROUP BY referrer ORDER BY COUNT(*) DESC LIMIT $3`,
		orgID, since, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []TopReferrer
	for rows.Next() {
		var t TopReferrer
		if err := rows.Scan(&t.Referrer, &t.Count); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func (r *Repository) MonthlyClickCount(ctx context.Context, orgID string, since time.Time) (int64, error) {
	var count int64
	err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM click_events WHERE org_id = $1 AND created_at >= $2`,
		orgID, since,
	).Scan(&count)
	return count, err
}
