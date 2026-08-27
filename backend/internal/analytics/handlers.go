package analytics

import (
	"net/http"
	"strconv"

	"saas-project/backend/internal/httputil"
	"saas-project/backend/internal/middleware"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// Summary returns click totals grouped by day and top referrers for the last N days (default 30).
func (h *Handler) Summary(w http.ResponseWriter, r *http.Request) {
	days := 30
	if v := r.URL.Query().Get("days"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 365 {
			days = n
		}
	}

	orgID := middleware.OrgID(r.Context())
	summary, err := h.service.Summary(r.Context(), orgID, days)
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "could not load analytics summary")
		return
	}
	httputil.WriteJSON(w, http.StatusOK, summary)
}
