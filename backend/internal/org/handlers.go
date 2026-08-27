package org

import (
	"net/http"

	"saas-project/backend/internal/httputil"
	"saas-project/backend/internal/middleware"
)

type Handler struct {
	orgs *Repository
}

func NewHandler(orgs *Repository) *Handler {
	return &Handler{orgs: orgs}
}

// ListMembers returns the members of the caller's active org.
func (h *Handler) ListMembers(w http.ResponseWriter, r *http.Request) {
	orgID := middleware.OrgID(r.Context())
	members, err := h.orgs.ListMembers(r.Context(), orgID)
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "could not list members")
		return
	}
	httputil.WriteJSON(w, http.StatusOK, map[string]any{"members": members})
}
