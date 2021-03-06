package webapi

import (
	"errors"
	"net/http"

	"github.com/Sugar-pack/users-manager/pkg/logging"
	"github.com/go-chi/chi/v5"
	"github.com/go-redis/redis/v8"

	"github.com/Sugar-pack/rest-server/internal/responsecache"
)

func CachedResponse(cacheConn *responsecache.Cache) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		logger := logging.FromContext(ctx)
		bgID := chi.URLParam(r, "bg_id")
		logger = logger.WithField("bg_id", bgID)
		httpResp, err := responsecache.GetResponse(ctx, cacheConn, bgID)
		if err != nil {
			errRedisNil := redis.Nil
			if errors.As(err, &errRedisNil) {
				logger.Warn("background id not found")
				NotFound(ctx, w, "background id not found")
				return
			}
			logger.WithError(err).Error("get response failed")
			InternalError(ctx, w, "get response failed")
			return
		}
		if err = responsecache.DeleteResponse(ctx, cacheConn, bgID); err != nil {
			logger.WithError(err).Warn("drop cache key failed")
		} else {
			logger.Trace("response purged")
		}
		rawResponse(ctx, w, httpResp.Code, httpResp.Headers, httpResp.Body)
	}
}
