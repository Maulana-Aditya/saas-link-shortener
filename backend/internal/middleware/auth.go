// Package middleware holds chi-compatible HTTP middleware: auth, tenant scoping, rate limiting.
package middleware

import (
	"context"
	"net/http"
	"strings"

	"saas-project/backend/internal/auth"
)

type ctxKey string

const (
	ctxUserID ctxKey = "user_id"
	ctxOrgID  ctxKey = "org_id"
)

// RequireAuth validates the Bearer access token and injects user_id/org_id into the request context.
func RequireAuth(issuer *auth.Issuer) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := r.Header.Get("Authorization")
			if !strings.HasPrefix(header, "Bearer ") {
				http.Error(w, `{"error":"missing bearer token"}`, http.StatusUnauthorized)
				return
			}
			token := strings.TrimPrefix(header, "Bearer ")
			claims, err := issuer.ParseAccessToken(token)
			if err != nil {
				http.Error(w, `{"error":"invalid or expired token"}`, http.StatusUnauthorized)
				return
			}
			ctx := context.WithValue(r.Context(), ctxUserID, claims.UserID)
			ctx = context.WithValue(ctx, ctxOrgID, claims.OrgID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func UserID(ctx context.Context) string {
	v, _ := ctx.Value(ctxUserID).(string)
	return v
}

func OrgID(ctx context.Context) string {
	v, _ := ctx.Value(ctxOrgID).(string)
	return v
}
