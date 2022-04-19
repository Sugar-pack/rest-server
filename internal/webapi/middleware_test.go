package webapi

import (
	"context"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Sugar-pack/users-manager/pkg/logging"
	"github.com/stretchr/testify/assert"
)

func TestAsyncMw_HasAsyncHeader_DefaultTTL_InBackground(t *testing.T) {
	logger := logging.GetLogger()
	ctx := context.Background()
	ctx = logging.WithContext(ctx, logger)
	httpHeaders := make(http.Header)
	httpHeaders.Add(HTTPHeaderXBackground, "true")

	bgResponses := make(map[string][]byte)
	mw := Async(bgResponses)
	fakeHandler := new(backgroundResponse)
	handlerFn := mw(fakeHandler)

	testRecorder := httptest.NewRecorder()
	testRequest := httptest.NewRequest(http.MethodGet, "/any", nil)
	testRequest = testRequest.WithContext(ctx)
	testRequest.Header = httpHeaders

	handlerFn.ServeHTTP(testRecorder, testRequest)
	gotHttpCode := testRecorder.Code
	gotResponseBody, err := ioutil.ReadAll(testRecorder.Body)
	if err != nil {
		t.Fatal(err)
	}

	expectedHTTPCode := http.StatusAccepted
	expectedBody := "request will be executed in the background"
	assert.Equal(t, expectedHTTPCode, gotHttpCode)
	assert.Equal(t, expectedBody, string(gotResponseBody))

	gotHeaderBackgroundID := testRecorder.Header().Get("x-background-id")
	assert.NotEmpty(t, gotHeaderBackgroundID)

	<-time.NewTimer(150 * time.Millisecond).C // need to wait till handler completion
	realResponse, ok := bgResponses[gotHeaderBackgroundID]
	assert.True(t, ok, "background response must exist")
	expectedRealResponse := "a long time ago"
	assert.Equal(t, expectedRealResponse, string(realResponse), "background response must match")
}

type backgroundResponse struct{}

func (s *backgroundResponse) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	time.Sleep(120 * time.Millisecond) //nolint:revive,gomnd // this is temporary and should be removed
	StatusOk(ctx, w, "a long time ago")
}

type handlerResponse struct{}

func (h *handlerResponse) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	StatusOk(ctx, w, "fast and furious")
}

func TestAsyncMw_HasAsyncHeader_DefaultTTL_ResponseByHandler(t *testing.T) {
	logger := logging.GetLogger()
	ctx := context.Background()
	ctx = logging.WithContext(ctx, logger)
	httpHeaders := make(http.Header)
	httpHeaders.Add(HTTPHeaderXBackground, "true")

	bgResponses := make(map[string][]byte)
	mw := Async(bgResponses)
	fakeHandler := new(handlerResponse)
	handlerFn := mw(fakeHandler)

	testRecorder := httptest.NewRecorder()
	testRequest := httptest.NewRequest(http.MethodGet, "/any", nil)
	testRequest = testRequest.WithContext(ctx)
	testRequest.Header = httpHeaders

	handlerFn.ServeHTTP(testRecorder, testRequest)

	gotHttpCode := testRecorder.Code
	gotResponseBody, err := ioutil.ReadAll(testRecorder.Body)
	if err != nil {
		t.Fatal(err)
	}
	expectedHTTPCode := http.StatusOK
	expectedBody := "fast and furious"
	assert.Equal(t, expectedHTTPCode, gotHttpCode)
	assert.Equal(t, expectedBody, string(gotResponseBody))

	gotHeaderBackgroundID := testRecorder.Header().Get("x-background-id")
	assert.Empty(t, gotHeaderBackgroundID)
}

func TestAsync_HasEmptyAsyncHeader(t *testing.T) {
	logger := logging.GetLogger()
	ctx := context.Background()
	ctx = logging.WithContext(ctx, logger)
	httpHeaders := make(http.Header)
	httpHeaders.Add(HTTPHeaderXBackground, "")

	bgResponses := make(map[string][]byte)
	mw := Async(bgResponses)
	fakeHandler := new(backgroundResponse)
	handlerFn := mw(fakeHandler)

	testRecorder := httptest.NewRecorder()
	testRequest := httptest.NewRequest(http.MethodGet, "/any", nil)
	testRequest = testRequest.WithContext(ctx)
	testRequest.Header = httpHeaders

	handlerFn.ServeHTTP(testRecorder, testRequest)
	gotHttpCode := testRecorder.Code
	gotResponseBody, err := ioutil.ReadAll(testRecorder.Body)
	if err != nil {
		t.Fatal(err)
	}

	expectedHTTPCode := http.StatusOK
	expectedBody := "a long time ago"
	assert.Equal(t, expectedHTTPCode, gotHttpCode)
	assert.Equal(t, expectedBody, string(gotResponseBody))

	gotHeaderBackgroundID := testRecorder.Header().Get("x-background-id")
	assert.Empty(t, gotHeaderBackgroundID)
}

func TestAsyncMw_HasAsyncHeader_HasTTLHeader(t *testing.T) {
	logger := logging.GetLogger()
	ctx := context.Background()
	ctx = logging.WithContext(ctx, logger)
	httpHeaders := make(http.Header)
	httpHeaders.Add(HTTPHeaderXBackground, "true")
	requestTTL := 50 * time.Millisecond
	httpHeaders.Add(HTTPHeaderXBackgroundTTL, requestTTL.String())

	bgResponses := make(map[string][]byte)
	mw := Async(bgResponses)
	fakeHandler := new(withRequestTTL)
	handlerFn := mw(fakeHandler)

	testRecorder := httptest.NewRecorder()
	testRequest := httptest.NewRequest(http.MethodGet, "/any", nil)
	testRequest = testRequest.WithContext(ctx)
	testRequest.Header = httpHeaders

	handlerFn.ServeHTTP(testRecorder, testRequest)

	gotHttpCode := testRecorder.Code
	gotResponseBody, err := ioutil.ReadAll(testRecorder.Body)
	if err != nil {
		t.Fatal(err)
	}
	expectedHTTPCode := http.StatusAccepted
	expectedBody := "request will be executed in the background"
	assert.Equal(t, expectedHTTPCode, gotHttpCode)
	assert.Equal(t, expectedBody, string(gotResponseBody))

	gotHeaderBackgroundID := testRecorder.Header().Get("x-background-id")
	assert.NotEmpty(t, gotHeaderBackgroundID)
}

type withRequestTTL struct{}

func (s *withRequestTTL) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	time.Sleep(60 * time.Millisecond) //nolint:revive,gomnd // this is temporary and should be removed
	StatusOk(ctx, w, "a long time ago")
}
