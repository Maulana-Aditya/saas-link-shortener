package org

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"saas-project/backend/internal/db"
)

var ErrNotFound = errors.New("org not found")

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) Create(ctx context.Context, name, slug string) (*db.Org, error) {
	var o db.Org
	err := r.pool.QueryRow(ctx,
		`INSERT INTO orgs (name, slug) VALUES ($1, $2)
		 RETURNING id, name, slug, plan, stripe_customer_id, stripe_subscription_id, created_at, updated_at`,
		name, slug,
	).Scan(&o.ID, &o.Name, &o.Slug, &o.Plan, &o.StripeCustomerID, &o.StripeSubscriptionID, &o.CreatedAt, &o.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &o, nil
}

func (r *Repository) AddMember(ctx context.Context, orgID, userID, role string) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO org_members (org_id, user_id, role) VALUES ($1, $2, $3)`,
		orgID, userID, role,
	)
	return err
}

func (r *Repository) FindByID(ctx context.Context, id string) (*db.Org, error) {
	var o db.Org
	err := r.pool.QueryRow(ctx,
		`SELECT id, name, slug, plan, stripe_customer_id, stripe_subscription_id, created_at, updated_at FROM orgs WHERE id = $1`,
		id,
	).Scan(&o.ID, &o.Name, &o.Slug, &o.Plan, &o.StripeCustomerID, &o.StripeSubscriptionID, &o.CreatedAt, &o.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &o, nil
}

// FirstOrgForUser returns the first org (by join order) a user belongs to.
// Used to resolve the "active org" right after login until multi-org switching UI exists.
func (r *Repository) FirstOrgForUser(ctx context.Context, userID string) (*db.Org, error) {
	var o db.Org
	err := r.pool.QueryRow(ctx,
		`SELECT o.id, o.name, o.slug, o.plan, o.stripe_customer_id, o.stripe_subscription_id, o.created_at, o.updated_at
		 FROM orgs o
		 JOIN org_members m ON m.org_id = o.id
		 WHERE m.user_id = $1
		 ORDER BY m.created_at ASC
		 LIMIT 1`,
		userID,
	).Scan(&o.ID, &o.Name, &o.Slug, &o.Plan, &o.StripeCustomerID, &o.StripeSubscriptionID, &o.CreatedAt, &o.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &o, nil
}

type Member struct {
	UserID string
	Email  string
	Name   string
	Role   string
}

func (r *Repository) ListMembers(ctx context.Context, orgID string) ([]Member, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT u.id, u.email, u.name, m.role
		 FROM org_members m
		 JOIN users u ON u.id = m.user_id
		 WHERE m.org_id = $1
		 ORDER BY m.created_at ASC`,
		orgID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var members []Member
	for rows.Next() {
		var m Member
		if err := rows.Scan(&m.UserID, &m.Email, &m.Name, &m.Role); err != nil {
			return nil, err
		}
		members = append(members, m)
	}
	return members, rows.Err()
}

func (r *Repository) SetStripeCustomerID(ctx context.Context, orgID, customerID string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE orgs SET stripe_customer_id = $2, updated_at = now() WHERE id = $1`,
		orgID, customerID,
	)
	return err
}

func (r *Repository) UpdateStripeInfo(ctx context.Context, orgID string, customerID, subscriptionID, plan string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE orgs SET stripe_customer_id = $2, stripe_subscription_id = $3, plan = $4, updated_at = now() WHERE id = $1`,
		orgID, customerID, subscriptionID, plan,
	)
	return err
}

// DowngradeByStripeCustomerID resets an org to the free plan when its Stripe subscription is cancelled.
func (r *Repository) DowngradeByStripeCustomerID(ctx context.Context, customerID string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE orgs SET plan = 'free', stripe_subscription_id = NULL, updated_at = now() WHERE stripe_customer_id = $1`,
		customerID,
	)
	return err
}
