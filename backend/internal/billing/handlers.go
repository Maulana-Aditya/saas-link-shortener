package billing

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"

	"github.com/stripe/stripe-go/v79"
	"github.com/stripe/stripe-go/v79/webhook"

	"saas-project/backend/internal/httputil"
	"saas-project/backend/internal/middleware"
	"saas-project/backend/internal/org"
)

type Handler struct {
	service       *Service
	orgs          *org.Repository
	webhookSecret string
}

func NewHandler(service *Service, orgs *org.Repository, webhookSecret string) *Handler {
	return &Handler{service: service, orgs: orgs, webhookSecret: webhookSecret}
}

// CreateCheckoutSession starts a Stripe Checkout flow to upgrade the caller's org to Pro.
func (h *Handler) CreateCheckoutSession(w http.ResponseWriter, r *http.Request) {
	orgID := middleware.OrgID(r.Context())
	o, err := h.orgs.FindByID(r.Context(), orgID)
	if err != nil {
		httputil.WriteError(w, http.StatusNotFound, "org not found")
		return
	}

	url, err := h.service.CreateCheckoutSession(r.Context(), orgID, o.Name)
	if err != nil {
		log.Printf("billing: checkout session error: %v", err)
		httputil.WriteError(w, http.StatusInternalServerError, "could not start checkout")
		return
	}
	httputil.WriteJSON(w, http.StatusOK, map[string]string{"checkout_url": url})
}

// Usage returns the caller's org plan and current-period usage against plan limits.
func (h *Handler) Usage(w http.ResponseWriter, r *http.Request) {
	orgID := middleware.OrgID(r.Context())
	status, err := h.service.UsageStatus(r.Context(), orgID)
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "could not load usage")
		return
	}
	httputil.WriteJSON(w, http.StatusOK, status)
}

// Webhook handles Stripe events for subscription lifecycle changes.
// It must be mounted with the raw request body (no JSON-decoding middleware in front of it).
func (h *Handler) Webhook(w http.ResponseWriter, r *http.Request) {
	const maxBodyBytes = int64(65536)
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, maxBodyBytes))
	if err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "request body too large")
		return
	}

	event, err := webhook.ConstructEvent(body, r.Header.Get("Stripe-Signature"), h.webhookSecret)
	if err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid webhook signature")
		return
	}

	ctx := r.Context()
	switch string(event.Type) {
	case "checkout.session.completed":
		var sess stripe.CheckoutSession
		if err := json.Unmarshal(event.Data.Raw, &sess); err != nil {
			log.Printf("billing: could not parse checkout session: %v", err)
			break
		}
		orgID := sess.Metadata["org_id"]
		if orgID == "" || sess.Customer == nil || sess.Subscription == nil {
			log.Printf("billing: checkout session missing org_id/customer/subscription")
			break
		}
		if err := h.orgs.UpdateStripeInfo(ctx, orgID, sess.Customer.ID, sess.Subscription.ID, "pro"); err != nil {
			log.Printf("billing: could not update org after checkout: %v", err)
		}

	case "customer.subscription.deleted":
		var sub stripe.Subscription
		if err := json.Unmarshal(event.Data.Raw, &sub); err != nil {
			log.Printf("billing: could not parse subscription: %v", err)
			break
		}
		if sub.Customer == nil {
			break
		}
		if err := h.downgradeByCustomerID(ctx, sub.Customer.ID); err != nil {
			log.Printf("billing: could not downgrade org after subscription cancellation: %v", err)
		}
	}

	w.WriteHeader(http.StatusOK)
}

func (h *Handler) downgradeByCustomerID(ctx context.Context, customerID string) error {
	// Downgrading looks up the org by its saved Stripe customer ID.
	// Kept as a small helper to make the webhook switch above easy to read.
	return h.orgs.DowngradeByStripeCustomerID(ctx, customerID)
}
