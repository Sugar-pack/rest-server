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
			r.WithContext(ctxWithLogger)
			next.ServeHTTP(w, r)
		})
	}
}

func WithLogRequestBoundaries() func(next http.Handler) http.Handler {
	handler := func(next http.Handler) http.Handler {
		mw := func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			logger := logging.FromContext(ctx)
			requestURI := r.RequestURI
			requestMethod := r.Method
			logRequest := fmt.Sprintf("%s %s", requestMethod, requestURI)
			logger.WithField("request", logRequest).Trace("REQUEST_STARTED")
			next.ServeHTTP(w, r)
			logger.WithField("request", logRequest).Trace("REQUEST_COMPLETED")
		}
		return http.HandlerFunc(mw)
	}
	return handler
}

type asyncResponseWriter struct {
	id             uuid.UUID
	w              http.ResponseWriter
	buf            *bytes.Buffer
	code           int
	responseIsSent bool
	storage        map[string][]byte
	headers        http.Header
}

func (a *asyncResponseWriter) Header() http.Header {
	return a.headers
}

func (a *asyncResponseWriter) Write(i []byte) (int, error) {
	if !a.responseIsSent {
		return a.buf.Write(i)
	} else {
		a.storage[a.id.String()] = i
	}
	return 0, nil
}

func (a *asyncResponseWriter) WriteHeader(statusCode int) {
	if !a.responseIsSent {
		a.responseIsSent = true
		a.code = statusCode
		//a.w.WriteHeader(statusCode)
	}
}

func Async(bgResponses map[string][]byte) func(http.Handler) http.Handler {
	mw := func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			logger := logging.FromContext(ctx)
			timeout := 100 * time.Millisecond
			timer := time.NewTimer(timeout)
			timeNow := time.Now()
			catchTimeoutCh := make(chan struct{})
			//catchResponseCh := make(chan struct{})
			rid := uuid.New()
			var buffBytes []byte
			responseBuffer := bytes.NewBuffer(buffBytes)
			headers := http.Header{}
			aw := &asyncResponseWriter{
				id:      rid,
				storage: bgResponses,
				w:       w,
				buf:     responseBuffer,
				headers: headers,
			}
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

			go func() {
				next.ServeHTTP(aw, r)
				//select {
				//case catchResponseCh <- struct{}{}:
				//default:
				//}
			}()
			select {
			case <-catchTimeoutCh:
				StatusAccepted(ctx, w, "request will be executed in the background", rid.String())
				//case <-catchResponseCh:
			}
		}
		handlerFn := http.HandlerFunc(fn)
		return handlerFn
	}
	return mw
}
