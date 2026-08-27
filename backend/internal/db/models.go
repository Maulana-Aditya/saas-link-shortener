package db

import "time"

type User struct {
	ID           string
	Email        string
	PasswordHash string
	Name         string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type Org struct {
	ID                    string
	Name                  string
	Slug                  string
	Plan                  string
	StripeCustomerID      *string
	StripeSubscriptionID  *string
	CreatedAt             time.Time
	UpdatedAt             time.Time
}

type OrgMember struct {
	OrgID     string
	UserID    string
	Role      string
	CreatedAt time.Time
}

type Link struct {
	ID        string
	OrgID     string
	Slug      string
	LongURL   string
	CreatedBy *string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type ClickEvent struct {
	ID        int64
	LinkID    string
	OrgID     string
	IPHash    *string
	Country   *string
	Referrer  *string
	UserAgent *string
	CreatedAt time.Time
}
