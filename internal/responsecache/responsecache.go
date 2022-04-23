package responsecache

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/Sugar-pack/users-manager/pkg/logging"
	"github.com/go-redis/redis/extra/redisotel/v8"
	"github.com/go-redis/redis/v8"
)

type HTTPResponse struct {
	Code    int         `json:"code"`
	Headers http.Header `json:"headers"`
	Body    []byte      `json:"body"`
}

func (h *HTTPResponse) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, h)
}

func (h *HTTPResponse) MarshalBinary() (data []byte, err error) {
	return json.Marshal(h)
}

type Cache struct {
	Client *redis.Client
}

type CacheOption func(rOpt *redis.Options)

func WithAddr(addr string) CacheOption {
	return func(rOpt *redis.Options) {
		rOpt.Addr = addr
	}
}

func NewCache(ctx context.Context, clientOpts ...CacheOption) (*Cache, error) {
	logger := logging.FromContext(ctx)
	redisOpts := new(redis.Options)
	for i := range clientOpts {
		funcOpt := clientOpts[i]
		funcOpt(redisOpts)
	}
	rdb := redis.NewClient(redisOpts)
	if err := rdb.Ping(ctx).Err(); err != nil {
		logger.WithError(err).Error("ping failed")
		return nil, err
	}
	rdb.AddHook(redisotel.NewTracingHook())
	cache := &Cache{Client: rdb}
	return cache, nil
}

func SaveResponse(ctx context.Context, c *Cache, k string, resp *HTTPResponse) error {
	noTTL := time.Duration(0)
	return c.Client.Set(ctx, k, resp, noTTL).Err()
}

func GetResponse(ctx context.Context, c *Cache, k string) (*HTTPResponse, error) {
	httpResp := new(HTTPResponse)
	err := c.Client.Get(ctx, k).Scan(httpResp)
	return httpResp, err
}

func DeleteResponse(ctx context.Context, c *Cache, k string) error {
	return c.Client.Del(ctx, k).Err()
}
