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

func TestAsync_Timeout(t *testing.T) {
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

	<-time.NewTimer(200 * time.Millisecond).C // need to wait till handler completion
	bgResponses := handler.BgResponses
	realResponse, ok := bgResponses[gotHeaderBackgroundID]
	assert.True(t, ok, "background response must exist")
	expectedRealResponse := "a long time ago"
	assert.Equal(t, expectedRealResponse, string(realResponse), "background response must match")
}

func TestAsync_ResponseByHandler(t *testing.T) {
	logger := logging.GetLogger()
	handler := NewHandler(nil, nil)
	router := CreateRouter(logger, handler)
	response := makeTestRequest(t, router, http.MethodGet, "/fast")

	gotHttpCode := response.Code
	gotResponseBody, err := ioutil.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	expectedHTTPCode := http.StatusOK
	expectedBody := "fast and furious"
	assert.Equal(t, expectedHTTPCode, gotHttpCode)
	assert.Equal(t, expectedBody, string(gotResponseBody))

	gotHeaderBackgroundID := response.Header().Get("x-background-id")
	assert.Empty(t, gotHeaderBackgroundID)
}

func makeTestRequest(t *testing.T, router http.Handler, httpMethod string, uri string) *httptest.ResponseRecorder {
	t.Helper()
	testRecorder := httptest.NewRecorder()
	testRequest := httptest.NewRequest(httpMethod, uri, nil)
	router.ServeHTTP(testRecorder, testRequest)
	return testRecorder
}
