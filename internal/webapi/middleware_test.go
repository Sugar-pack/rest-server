package webapi

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Sugar-pack/users-manager/pkg/logging"
	"github.com/stretchr/testify/assert"
)

func TestAsync(t *testing.T) {
	logger := logging.GetLogger()
	handler := NewHandler(nil, nil)
	router := CreateRouter(logger, handler)
	response := makeTestRequest(t, router, http.MethodGet, "/durable")

	gotHttpCode := response.Code
	gotResponseBody, err := ioutil.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	expectedHTTPCode := http.StatusAccepted
	expectedBody := "request will be executed in the background"
	assert.Equal(t, expectedHTTPCode, gotHttpCode)
	assert.Equal(t, expectedBody, string(gotResponseBody))

	gotHeaderBackgroundID := response.Header().Get("x-background-id")
	assert.NotEmpty(t, gotHeaderBackgroundID)

	<-time.NewTimer(2000 * time.Millisecond).C
	bgResponses := handler.BgResponses
	realResponse, ok := bgResponses[gotHeaderBackgroundID]
	assert.True(t, ok)
	expectedRealResponse := "text"
	assert.Equal(t, expectedRealResponse, string(realResponse))
}

func makeTestRequest(t *testing.T, router http.Handler, httpMethod string, uri string) *httptest.ResponseRecorder {
	t.Helper()
	testRecorder := httptest.NewRecorder()
	testRequest := httptest.NewRequest(httpMethod, uri, nil)
	router.ServeHTTP(testRecorder, testRequest)
	return testRecorder
}
