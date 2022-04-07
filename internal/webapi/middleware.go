package webapi

import (
	"net/http"

	"github.com/Sugar-pack/users-manager/pkg/logging"
)

func LoggingMiddleware(logger logging.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			ctxWithLogger := logging.WithContext(ctx, logger)
			r.WithContext(ctxWithLogger)
			next.ServeHTTP(w, r)
		})
	}
}
