package webapi

import (
	"github.com/Sugar-pack/users-manager/pkg/logging"
	"github.com/go-chi/chi/v5"
)

func CreateRouter(logger logging.Logger, handler *Handler) *chi.Mux {
	router := chi.NewRouter()
	router.Use(LoggingMiddleware(logger))

	router.Post("send", handler.SendMessage)

	return router
}
