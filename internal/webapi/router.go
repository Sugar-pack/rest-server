package webapi

import (
	"fmt"
	"net/http"

	"github.com/Sugar-pack/users-manager/pkg/logging"
	"github.com/go-chi/chi/v5"
	httpSwagger "github.com/swaggo/http-swagger"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"

	"github.com/Sugar-pack/rest-server/internal/responsecache"
	"github.com/Sugar-pack/rest-server/internal/trace"
)

const TracerNameServer = "public-api"

func CreateRouter(logger logging.Logger, handler *Handler, cacheConn *responsecache.Cache) *chi.Mux {
	err := trace.InitJaegerTracing(logger)
	if err != nil {
		logger.WithError(err).Error("failed to init Jaeger Tracing")

		return nil
	}
	router := chi.NewRouter()
	router.Use(
		LoggingMiddleware(logger),
		WithLogRequestBoundaries(),
		AsyncMw(cacheConn),
	)

	router.Post("/send", handler.SendMessage)
	router.Get("/swagger/*", httpSwagger.Handler(
		httpSwagger.URL("/swagger/doc.json"),
	))
	router.Get("/durable", handler.Durable)
	router.Get("/bg-responses/{bg_id}", CachedResponse(cacheConn))

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
