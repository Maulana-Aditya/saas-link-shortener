package link

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"saas-project/backend/internal/db"
)

var ErrNotFound = errors.New("link not found")
var ErrSlugTaken = errors.New("slug already in use")

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) Create(ctx context.Context, orgID, slug, longURL, createdBy string) (*db.Link, error) {
	var l db.Link
	err := r.pool.QueryRow(ctx,
		`INSERT INTO links (org_id, slug, long_url, created_by) VALUES ($1, $2, $3, $4)
		 RETURNING id, org_id, slug, long_url, created_by, created_at, updated_at`,
		orgID, slug, longURL, createdBy,
	).Scan(&l.ID, &l.OrgID, &l.Slug, &l.LongURL, &l.CreatedBy, &l.CreatedAt, &l.UpdatedAt)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, ErrSlugTaken
		}
		return nil, err
	}
	return &l, nil
}

func (r *Repository) FindBySlug(ctx context.Context, slug string) (*db.Link, error) {
	var l db.Link
	err := r.pool.QueryRow(ctx,
		`SELECT id, org_id, slug, long_url, created_by, created_at, updated_at FROM links WHERE slug = $1`,
		slug,
	).Scan(&l.ID, &l.OrgID, &l.Slug, &l.LongURL, &l.CreatedBy, &l.CreatedAt, &l.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &l, nil
}

func (r *Repository) ListByOrg(ctx context.Context, orgID string) ([]db.Link, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, org_id, slug, long_url, created_by, created_at, updated_at
		 FROM links WHERE org_id = $1 ORDER BY created_at DESC`,
		orgID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var links []db.Link
	for rows.Next() {
		var l db.Link
		if err := rows.Scan(&l.ID, &l.OrgID, &l.Slug, &l.LongURL, &l.CreatedBy, &l.CreatedAt, &l.UpdatedAt); err != nil {
			return nil, err
		}
		links = append(links, l)
	}
	return links, rows.Err()
}

func (r *Repository) CountByOrg(ctx context.Context, orgID string) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM links WHERE org_id = $1`, orgID).Scan(&count)
	return count, err
}

func (r *Repository) DeleteByIDAndOrg(ctx context.Context, id, orgID string) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM links WHERE id = $1 AND org_id = $2`, id, orgID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
