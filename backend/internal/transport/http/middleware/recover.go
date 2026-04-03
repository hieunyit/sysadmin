package middleware

import (
	"net/http"
	"runtime/debug"

	"github.com/rs/zerolog"

	domainerr "backend/internal/domain/errors"
	"backend/pkg/httputil"
)

func Recoverer(logger zerolog.Logger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rec := recover(); rec != nil {
					logger.Error().
						Interface("panic", rec).
						Bytes("stack", debug.Stack()).
						Str("request_id", RequestID(r.Context())).
						Str("method", r.Method).
						Str("path", r.URL.Path).
						Msg("panic recovered")

					httputil.WriteAPIError(
						w,
						http.StatusInternalServerError,
						string(domainerr.CodeInternal),
						"internal error",
						RequestID(r.Context()),
						nil,
					)
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}
