package webapi

import (
	"github.com/Sugar-pack/rest-server/internal/responsecache"
	"github.com/Sugar-pack/users-manager/pkg/logging"
	"github.com/go-chi/chi/v5"
	httpSwagger "github.com/swaggo/http-swagger"
)

func CreateRouter(logger logging.Logger, handler *Handler, cacheConn *responsecache.Cache) *chi.Mux {
	router := chi.NewRouter()
	router.Use(
		LoggingMiddleware(logger),
		WithLogRequestBoundaries(),
		AsyncMw(handler.BgResponses, cacheConn),
	)

	router.Post("/send", handler.SendMessage)
	router.Get("/swagger/*", httpSwagger.Handler(
		httpSwagger.URL("/swagger/doc.json"),
	))
	router.Get("/durable", handler.Durable)

	return router
}
