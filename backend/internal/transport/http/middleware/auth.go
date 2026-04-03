package middleware

import (
	"net/http"
	"strings"

	domainerr "backend/internal/domain/errors"
	"backend/pkg/httputil"
)

func AdminAuth(adminToken string) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authz := strings.TrimSpace(r.Header.Get("Authorization"))
			token := ""
			if strings.HasPrefix(strings.ToLower(authz), "bearer ") {
				token = strings.TrimSpace(authz[len("Bearer "):])
			}
			if token == "" {
				token = r.Header.Get("X-Admin-Token")
			}
			if token == "" || token != adminToken {
				httputil.WriteAPIError(
					w,
					http.StatusUnauthorized,
					string(domainerr.CodeUnauthorized),
					"unauthorized",
					RequestID(r.Context()),
					nil,
				)
				return
			}
			actor := r.Header.Get("X-Actor")
			if actor == "" {
				actor = "api-admin"
			}
			next.ServeHTTP(w, r.WithContext(WithActor(r.Context(), actor)))
		})
	}
}
