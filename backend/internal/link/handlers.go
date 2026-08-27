package link

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strings"

	"github.com/go-chi/chi/v5"

	"saas-project/backend/internal/db"
	"saas-project/backend/internal/httputil"
	"saas-project/backend/internal/middleware"
)

// ClickRecorder decouples the redirect handler from the analytics package's
// storage details; analytics.Service implements this.
type ClickRecorder interface {
	RecordAsync(orgID, linkID string, r *http.Request)
}

// PlanLimiter reports whether an org may create another link under its current plan.
type PlanLimiter interface {
	CanCreateLink(ctx context.Context, orgID string, currentCount int) (bool, error)
}

type Handler struct {
	links     *Repository
	recorder  ClickRecorder
	limiter   PlanLimiter
	baseURL   string
}

func NewHandler(links *Repository, recorder ClickRecorder, limiter PlanLimiter, baseURL string) *Handler {
	return &Handler{links: links, recorder: recorder, limiter: limiter, baseURL: baseURL}
}

type createLinkRequest struct {
	LongURL string `json:"long_url"`
	Slug    string `json:"slug"` // optional custom slug
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req createLinkRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	req.LongURL = strings.TrimSpace(req.LongURL)
	parsed, err := url.ParseRequestURI(req.LongURL)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		httputil.WriteError(w, http.StatusBadRequest, "long_url must be a valid http(s) URL")
		return
	}

	ctx := r.Context()
	orgID := middleware.OrgID(ctx)
	userID := middleware.UserID(ctx)

	count, err := h.links.CountByOrg(ctx, orgID)
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "could not check plan limits")
		return
	}
	allowed, err := h.limiter.CanCreateLink(ctx, orgID, count)
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "could not check plan limits")
		return
	}
	if !allowed {
		httputil.WriteError(w, http.StatusPaymentRequired, "link limit reached for your plan, please upgrade")
		return
	}

	slug := strings.TrimSpace(req.Slug)
	var l *db.Link
	for range 5 {
		if slug == "" {
			slug, err = GenerateSlug()
			if err != nil {
				httputil.WriteError(w, http.StatusInternalServerError, "could not generate slug")
				return
			}
		}
		created, err := h.links.Create(ctx, orgID, slug, req.LongURL, userID)
		if err != nil {
			if errors.Is(err, ErrSlugTaken) {
				if req.Slug != "" {
					httputil.WriteError(w, http.StatusConflict, "slug already in use")
					return
				}
				slug = "" // retry with a freshly generated slug
				continue
			}
			httputil.WriteError(w, http.StatusInternalServerError, "could not create link")
			return
		}
		l = created
		break
	}
	if l == nil {
		httputil.WriteError(w, http.StatusInternalServerError, "could not generate a unique slug, try again")
		return
	}

	httputil.WriteJSON(w, http.StatusCreated, map[string]string{
		"id":        l.ID,
		"slug":      l.Slug,
		"long_url":  l.LongURL,
		"short_url": h.baseURL + "/" + l.Slug,
	})
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	orgID := middleware.OrgID(r.Context())
	links, err := h.links.ListByOrg(r.Context(), orgID)
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "could not list links")
		return
	}

	out := make([]map[string]string, 0, len(links))
	for _, l := range links {
		out = append(out, map[string]string{
			"id":        l.ID,
			"slug":      l.Slug,
			"long_url":  l.LongURL,
			"short_url": h.baseURL + "/" + l.Slug,
		})
	}
	httputil.WriteJSON(w, http.StatusOK, map[string]any{"links": out})
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	orgID := middleware.OrgID(r.Context())
	if err := h.links.DeleteByIDAndOrg(r.Context(), id, orgID); err != nil {
		if errors.Is(err, ErrNotFound) {
			httputil.WriteError(w, http.StatusNotFound, "link not found")
			return
		}
		httputil.WriteError(w, http.StatusInternalServerError, "could not delete link")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// Redirect is the public, unauthenticated endpoint that resolves a short slug.
func (h *Handler) Redirect(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	l, err := h.links.FindBySlug(r.Context(), slug)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	h.recorder.RecordAsync(l.OrgID, l.ID, r)
	http.Redirect(w, r, l.LongURL, http.StatusFound)
}
