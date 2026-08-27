// Package billing wires Stripe Checkout + webhooks to org subscription state,
// and enforces free-plan usage limits for links and monthly clicks.
package billing

import (
	"context"
	"fmt"

	"github.com/stripe/stripe-go/v79"
	"github.com/stripe/stripe-go/v79/checkout/session"
	"github.com/stripe/stripe-go/v79/customer"

	"saas-project/backend/internal/org"
)

// UsageSource reports current-period usage for plan-limit checks.
type UsageSource interface {
	MonthlyClickCount(ctx context.Context, orgID string) (int64, error)
}

type Config struct {
	SecretKey             string
	WebhookSecret         string
	ProPriceID            string
	FreePlanLinkLimit     int
	FreePlanMonthlyClicks int
	FrontendURL           string
}

type Service struct {
	cfg   Config
	orgs  *org.Repository
	usage UsageSource
}

func NewService(cfg Config, orgs *org.Repository, usage UsageSource) *Service {
	stripe.Key = cfg.SecretKey
	return &Service{cfg: cfg, orgs: orgs, usage: usage}
}

// CanCreateLink implements link.PlanLimiter.
func (s *Service) CanCreateLink(ctx context.Context, orgID string, currentCount int) (bool, error) {
	o, err := s.orgs.FindByID(ctx, orgID)
	if err != nil {
		return false, err
	}
	if o.Plan != "free" {
		return true, nil
	}
	return currentCount < s.cfg.FreePlanLinkLimit, nil
}

// CanRecordClick reports whether the org's free-plan monthly click quota has room.
// Not enforced on the hot redirect path (clicks are recorded fire-and-forget),
// but exposed for the dashboard to show usage / upgrade prompts.
func (s *Service) UsageStatus(ctx context.Context, orgID string) (map[string]any, error) {
	o, err := s.orgs.FindByID(ctx, orgID)
	if err != nil {
		return nil, err
	}
	clicks, err := s.usage.MonthlyClickCount(ctx, orgID)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"plan":                    o.Plan,
		"monthly_clicks":          clicks,
		"monthly_click_limit":     s.cfg.FreePlanMonthlyClicks,
		"link_limit":              s.cfg.FreePlanLinkLimit,
	}, nil
}

// CreateCheckoutSession creates (or reuses) a Stripe customer for the org and
// returns a hosted Checkout URL for upgrading to the Pro plan.
func (s *Service) CreateCheckoutSession(ctx context.Context, orgID, orgName string) (string, error) {
	o, err := s.orgs.FindByID(ctx, orgID)
	if err != nil {
		return "", err
	}

	customerID := ""
	if o.StripeCustomerID != nil {
		customerID = *o.StripeCustomerID
	}
	if customerID == "" {
		cust, err := customer.New(&stripe.CustomerParams{
			Name: stripe.String(orgName),
		})
		if err != nil {
			return "", fmt.Errorf("create stripe customer: %w", err)
		}
		customerID = cust.ID
		if err := s.orgs.SetStripeCustomerID(ctx, orgID, customerID); err != nil {
			return "", fmt.Errorf("save stripe customer id: %w", err)
		}
	}

	params := &stripe.CheckoutSessionParams{
		Customer:   stripe.String(customerID),
		Mode:       stripe.String(string(stripe.CheckoutSessionModeSubscription)),
		SuccessURL: stripe.String(s.cfg.FrontendURL + "/billing?checkout=success"),
		CancelURL:  stripe.String(s.cfg.FrontendURL + "/billing?checkout=cancelled"),
		LineItems: []*stripe.CheckoutSessionLineItemParams{
			{
				Price:    stripe.String(s.cfg.ProPriceID),
				Quantity: stripe.Int64(1),
			},
		},
	}
	params.AddMetadata("org_id", orgID)

	sess, err := session.New(params)
	if err != nil {
		return "", fmt.Errorf("create checkout session: %w", err)
	}
	return sess.URL, nil
}
