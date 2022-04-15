package webapi

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Sugar-pack/users-manager/pkg/logging"
)

func TestAsync(t *testing.T) {
	logger := logging.GetLogger()
	handler := NewHandler(nil, nil)
	router := CreateRouter(logger, handler)
	response := makeTestRequest(t, router, http.MethodGet, "/durable")
}

func makeTestRequest(t *testing.T, router http.Handler, httpMethod string, uri string) *httptest.ResponseRecorder {
	t.Helper()
	testRecorder := httptest.NewRecorder()
	testRequest := httptest.NewRequest(httpMethod, uri, nil)
	router.ServeHTTP(testRecorder, testRequest)
	return testRecorder
}
