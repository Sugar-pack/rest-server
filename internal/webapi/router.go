package webapi

import (
	"fmt"
	"net/http"

	"github.com/Sugar-pack/users-manager/pkg/logging"
	"github.com/go-chi/chi/v5"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"

	"github.com/Sugar-pack/rest-server/internal/trace"
)

const TracerNameServer = "monitor_server"

func CreateRouter(logger logging.Logger, handler *Handler) *chi.Mux {
	err := trace.InitJaegerTracing(logger)
	if err != nil {
		logger.WithError(err).Error("failed to init Jaeger Tracing")

		return nil
	}
	router := chi.NewRouter()
	router.Use(LoggingMiddleware(logger))
	router.Post("/send", handler.SendMessage)

	return router
}

func TraceWrapRouter(router *chi.Mux) http.Handler {
	return otelhttp.NewHandler(
		router,
		TracerNameServer,
		otelhttp.WithSpanNameFormatter(func(operation string, r *http.Request) string {
			return fmt.Sprintf("%s %s", r.Method, r.URL.Path)
		}))
}
