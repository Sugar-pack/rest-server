package webapi

import (
	"net/http"
	"time"
)

func (h *Handler) Durable(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	time.Sleep(200 * time.Millisecond) //nolint:revive,gomnd // this is tempprary and should be removed
	StatusOk(ctx, w, "a long time ago")
}

func (h *Handler) FastAndFurious(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	StatusOk(ctx, w, "fast and furious")
}