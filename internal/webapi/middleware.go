package webapi

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/textproto"
	"time"

	"github.com/Sugar-pack/users-manager/pkg/logging"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel/trace"

	"github.com/Sugar-pack/rest-server/internal/responsecache"
)

func LoggingMiddleware(logger logging.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			ctxWithLogger := logging.WithContext(ctx, logger)
			r = r.WithContext(ctxWithLogger)
			next.ServeHTTP(w, r)
		})
	}
}

func WithLogRequestBoundaries() func(next http.Handler) http.Handler {
	httpMw := func(next http.Handler) http.Handler {
		handlerFn := func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			logger := logging.FromContext(ctx)
			requestURI := r.RequestURI
			requestMethod := r.Method
			logRequest := fmt.Sprintf("%s %s", requestMethod, requestURI)
			logger.WithField("request", logRequest).Trace("REQUEST_STARTED")
			next.ServeHTTP(w, r)
			logger.WithField("request", logRequest).Trace("REQUEST_COMPLETED")
		}
		return http.HandlerFunc(handlerFn)
	}
	return httpMw
}

type asyncResponseWriter struct {
	id      uuid.UUID
	buf     *bytes.Buffer
	code    int
	headers http.Header
}

func (a *asyncResponseWriter) Header() http.Header {
	return a.headers
}

func (a *asyncResponseWriter) Write(i []byte) (int, error) {
	return a.buf.Write(i)
}

func (a *asyncResponseWriter) WriteHeader(statusCode int) {
	a.code = statusCode
}

const (
	HTTPHeaderXBackground    = "x-background"
	HTTPHeaderXBackgroundTTL = "x-background-ttl"
	DefaultTimeout           = 100 * time.Millisecond
)

func NewAsyncResponseWriter() *asyncResponseWriter {
	rid := uuid.New()
	var buffBytes []byte
	responseBuffer := bytes.NewBuffer(buffBytes)
	headers := make(http.Header)
	return &asyncResponseWriter{
		id:      rid,
		buf:     responseBuffer,
		headers: headers,
	}
}

//nolint:gocognit,cyclop // need to refactor to decrease cyclo complexity
func AsyncMw(cacheConn *responsecache.Cache) func(http.Handler) http.Handler {
	httpMw := func(next http.Handler) http.Handler {
		handlerFn := func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			logger := logging.FromContext(ctx)
			catchTimeoutCh := make(chan uuid.UUID)
			catchResponseCh := make(chan *asyncResponseWriter)
			asyncRespWriter := NewAsyncResponseWriter()
			var timer *time.Timer
			if timeout, ok := hasBackgroundHeader(ctx, r.Header, DefaultTimeout); ok {
				timer = time.NewTimer(timeout)
				timeNow := time.Now().UTC()
				go func() {
					select {
					case tt := <-timer.C:
						logger.WithField("timer", tt.Sub(timeNow)).Warn("timeout occurred")
						select {
						case catchTimeoutCh <- asyncRespWriter.id:
						default:
						}
					}
				}()
			}

			go func() {
				asyncCtx := context.Background()
				lCtx := logging.FromContext(ctx)
				asyncCtx = logging.WithContext(asyncCtx, lCtx)
				newChiCtx := chi.NewRouteContext()
				r = r.WithContext(context.WithValue(asyncCtx, chi.RouteCtxKey, newChiCtx))
				span := trace.SpanFromContext(ctx)
				r = r.WithContext(trace.ContextWithSpan(r.Context(), span))
				next.ServeHTTP(asyncRespWriter, r)
				select {
				case catchResponseCh <- asyncRespWriter:
					if timer != nil {
						timer.Stop() // timer is not required any more. stop it.
					}
				default: // if response already sent, then save real response in the cache
					saveErr := responsecache.SaveResponse(asyncCtx, cacheConn, asyncRespWriter.id.String(),
						&responsecache.HTTPResponse{
							Code:    asyncRespWriter.code,
							Headers: asyncRespWriter.headers,
							Body:    asyncRespWriter.buf.Bytes(),
						})
					if saveErr != nil {
						logger.WithError(saveErr).Error("save response in cache failed")
					}
				}
			}()
			select {
			case backgroundID := <-catchTimeoutCh:
				StatusAccepted(ctx, w, "request will be executed in the background", backgroundID.String())
			case syncResponse := <-catchResponseCh:
				rawResponse(ctx, w, syncResponse.code, syncResponse.headers, syncResponse.buf.Bytes())
			}
		}
		httpHandlerFn := http.HandlerFunc(handlerFn)
		return httpHandlerFn
	}
	return httpMw
}

func hasBackgroundHeader(ctx context.Context, httpHeader http.Header, defaultTTL time.Duration) (time.Duration, bool) {
	logger := logging.FromContext(ctx)
	backgroundHeaders, hasBgHeader := httpHeader[textproto.CanonicalMIMEHeaderKey(HTTPHeaderXBackground)]
	if !hasBgHeader {
		logger.Trace("http background header is not passed. it is sync mode")
		return defaultTTL, false
	}

	if len(backgroundHeaders) == 0 {
		logger.Trace("http background header is empty list. it is sync mode")
		return defaultTTL, false
	}

	hasAsyncHeader := backgroundHeaders[0]
	if hasAsyncHeader == "" {
		logger.Trace("http background header is empty string. it is sync mode")
		return defaultTTL, false
	}

	timeout := backgroundTTL(ctx, httpHeader.Get(HTTPHeaderXBackgroundTTL), defaultTTL)
	return timeout, true
}

func backgroundTTL(ctx context.Context, rawTTL string, defaultTTL time.Duration) time.Duration {
	logger := logging.FromContext(ctx)
	ttl, err := time.ParseDuration(rawTTL)
	if err != nil {
		logger.WithError(err).WithField("raw_ttl", rawTTL).Warn("parse raw ttl failed, use default value")
		return defaultTTL
	}
	return ttl
}
