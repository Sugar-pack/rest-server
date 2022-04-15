package webapi

import (
	"net/http"
	"time"
)

func (h *Handler) Durable(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	time.Sleep(200 * time.Second)
	StatusOk(ctx, w, "a long time ago")
}
