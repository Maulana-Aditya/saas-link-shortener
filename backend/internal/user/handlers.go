package user

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/google/uuid"

	"saas-project/backend/internal/auth"
	"saas-project/backend/internal/httputil"
	"saas-project/backend/internal/middleware"
	"saas-project/backend/internal/org"
)

type Handler struct {
	users   *Repository
	orgs    *org.Repository
	issuer  *auth.Issuer
}

func NewHandler(users *Repository, orgs *org.Repository, issuer *auth.Issuer) *Handler {
	return &Handler{users: users, orgs: orgs, issuer: issuer}
}

type signupRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Name     string `json:"name"`
	OrgName  string `json:"org_name"`
}

type authResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	User         struct {
		ID    string `json:"id"`
		Email string `json:"email"`
		Name  string `json:"name"`
	} `json:"user"`
	Org struct {
		ID   string `json:"id"`
		Name string `json:"name"`
		Plan string `json:"plan"`
	} `json:"org"`
}

func (h *Handler) Signup(w http.ResponseWriter, r *http.Request) {
	var req signupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	if req.Email == "" || len(req.Password) < 8 || req.Name == "" {
		httputil.WriteError(w, http.StatusBadRequest, "email, name and a password of at least 8 characters are required")
		return
	}
	if req.OrgName == "" {
		req.OrgName = req.Name + "'s Org"
	}

	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "could not process password")
		return
	}

	ctx := r.Context()
	u, err := h.users.Create(ctx, req.Email, hash, req.Name)
	if err != nil {
		if errors.Is(err, ErrEmailTaken) {
			httputil.WriteError(w, http.StatusConflict, "email already registered")
			return
		}
		httputil.WriteError(w, http.StatusInternalServerError, "could not create user")
		return
	}

	slug := slugify(req.OrgName) + "-" + uuid.NewString()[:8]
	o, err := h.orgs.Create(ctx, req.OrgName, slug)
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "could not create org")
		return
	}
	if err := h.orgs.AddMember(ctx, o.ID, u.ID, "owner"); err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "could not attach user to org")
		return
	}

	h.respondWithTokens(w, u.ID, u.Email, u.Name, o.ID, o.Name, o.Plan)
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))

	ctx := r.Context()
	u, err := h.users.FindByEmail(ctx, req.Email)
	if err != nil || !auth.VerifyPassword(u.PasswordHash, req.Password) {
		httputil.WriteError(w, http.StatusUnauthorized, "invalid email or password")
		return
	}

	o, err := h.orgs.FirstOrgForUser(ctx, u.ID)
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "user has no organization")
		return
	}

	h.respondWithTokens(w, u.ID, u.Email, u.Name, o.ID, o.Name, o.Plan)
}

type refreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

func (h *Handler) Refresh(w http.ResponseWriter, r *http.Request) {
	var req refreshRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.RefreshToken == "" {
		httputil.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	claims, err := h.issuer.ParseRefreshToken(req.RefreshToken)
	if err != nil {
		httputil.WriteError(w, http.StatusUnauthorized, "invalid or expired refresh token")
		return
	}

	access, err := h.issuer.IssueAccessToken(claims.UserID, claims.OrgID)
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "could not issue token")
		return
	}
	refresh, err := h.issuer.IssueRefreshToken(claims.UserID, claims.OrgID)
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "could not issue token")
		return
	}

	httputil.WriteJSON(w, http.StatusOK, map[string]string{
		"access_token":  access,
		"refresh_token": refresh,
	})
}

func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := middleware.UserID(ctx)
	orgID := middleware.OrgID(ctx)

	u, err := h.users.FindByID(ctx, userID)
	if err != nil {
		httputil.WriteError(w, http.StatusNotFound, "user not found")
		return
	}
	o, err := h.orgs.FindByID(ctx, orgID)
	if err != nil {
		httputil.WriteError(w, http.StatusNotFound, "org not found")
		return
	}

	httputil.WriteJSON(w, http.StatusOK, map[string]any{
		"user": map[string]string{"id": u.ID, "email": u.Email, "name": u.Name},
		"org":  map[string]string{"id": o.ID, "name": o.Name, "plan": o.Plan},
	})
}

func (h *Handler) respondWithTokens(w http.ResponseWriter, userID, email, name, orgID, orgName, plan string) {
	access, err := h.issuer.IssueAccessToken(userID, orgID)
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "could not issue token")
		return
	}
	refresh, err := h.issuer.IssueRefreshToken(userID, orgID)
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "could not issue token")
		return
	}

	var resp authResponse
	resp.AccessToken = access
	resp.RefreshToken = refresh
	resp.User.ID = userID
	resp.User.Email = email
	resp.User.Name = name
	resp.Org.ID = orgID
	resp.Org.Name = orgName
	resp.Org.Plan = plan

	httputil.WriteJSON(w, http.StatusOK, resp)
}

func slugify(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	var b strings.Builder
	lastDash := false
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z' || r >= '0' && r <= '9':
			b.WriteRune(r)
			lastDash = false
		default:
			if !lastDash {
				b.WriteByte('-')
				lastDash = true
			}
		}
	}
	return strings.Trim(b.String(), "-")
}
