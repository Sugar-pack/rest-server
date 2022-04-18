package webapi

import (
	"bytes"
	"fmt"
	"net/http"
	"time"

	"github.com/Sugar-pack/users-manager/pkg/logging"
	"github.com/google/uuid"
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
	storage map[string][]byte
}

func (a *asyncResponseWriter) Header() http.Header {
	return a.headers
}

func (a *asyncResponseWriter) Write(i []byte) (int, error) {
	a.storage[a.id.String()] = i
	return a.buf.Write(i)
}

func (a *asyncResponseWriter) WriteHeader(statusCode int) {
	a.code = statusCode
}

const HTTPHeaderXBackground = "x-background"

func Async(bgResponses map[string][]byte) func(http.Handler) http.Handler {
	httpMw := func(next http.Handler) http.Handler {
		handlerFn := func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			logger := logging.FromContext(ctx)
			timeout := 100 * time.Millisecond //nolint:revive,gomnd // this is tempprary and should be removed
			timer := time.NewTimer(timeout)
			timeNow := time.Now()
			catchTimeoutCh := make(chan struct{})
			catchResponseCh := make(chan struct{})
			rid := uuid.New()
			var buffBytes []byte
			responseBuffer := bytes.NewBuffer(buffBytes)
			headers := http.Header{}
			asyncRespWriter := &asyncResponseWriter{
				id:      rid,
				storage: bgResponses,
				buf:     responseBuffer,
				headers: headers,
			}
			hasAsyncHeader := r.Header.Get(HTTPHeaderXBackground)
			if hasAsyncHeader != "" {
				go func() {
					select {
					case tt := <-timer.C:
						logger.WithField("timer", tt.Sub(timeNow)).Warn("timeout occurred")
						select {
						case catchTimeoutCh <- struct{}{}:
						default:
						}
					}
				}()
			} else {
				logger.Trace("it is sync mode")
			}

			go func() {
				next.ServeHTTP(asyncRespWriter, r)
				select {
				case catchResponseCh <- struct{}{}:
					timer.Stop() // timer is not required any more. stop it.
				default:
				}
			}()
			select {
			case <-catchTimeoutCh:
				StatusAccepted(ctx, w, "request will be executed in the background", rid.String())
			case <-catchResponseCh:
				rawResponse(ctx, w, asyncRespWriter.code, asyncRespWriter.headers, asyncRespWriter.buf.Bytes())
			}
		}
		httpHandlerFn := http.HandlerFunc(handlerFn)
		return httpHandlerFn
	}
	return httpMw
}
