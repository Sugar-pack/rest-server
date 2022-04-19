package webapi

import (
	"context"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Sugar-pack/rest-server/internal/responsecache"
	"github.com/Sugar-pack/users-manager/pkg/logging"
	"github.com/agiledragon/gomonkey/v2"
	"github.com/go-redis/redismock/v8"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestAsyncMw_HasAsyncHeader_DefaultTTL_InBackground(t *testing.T) {
	logger := logging.GetLogger()
	ctx := context.Background()
	ctx = logging.WithContext(ctx, logger)
	httpHeaders := make(http.Header)
	httpHeaders.Add(HTTPHeaderXBackground, "true")

	mockedUUID := uuid.MustParse("ef24471b-e968-40f0-b4d4-c9d0410565c8")
	gomonkey.ApplyFunc(uuid.New, func() uuid.UUID {
		return mockedUUID
	})

	bgResponses := make(map[string][]byte)
	redisClient, mockedCacheConn := redismock.NewClientMock()
	cacheConn := &responsecache.Cache{
		Client: redisClient,
	}
	mockedCacheConn.ExpectSet(mockedUUID.String(), &responsecache.HTTPResponse{
		Code:    http.StatusOK,
		Headers: make(map[string][]string),
		Body:    []byte{97, 32, 108, 111, 110, 103, 32, 116, 105, 109, 101, 32, 97, 103, 111},
	}, time.Duration(0)).SetVal("val")

	mw := AsyncMw(bgResponses, cacheConn)
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
	redisClient, mockedCacheConn := redismock.NewClientMock()
	cacheConn := &responsecache.Cache{
		Client: redisClient,
	}
	mockedCacheConn.ExpectSet("", "", time.Duration(0)).RedisNil()

	mw := AsyncMw(bgResponses, cacheConn)
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
	redisClient, mockedCacheConn := redismock.NewClientMock()
	cacheConn := &responsecache.Cache{
		Client: redisClient,
	}
	mockedCacheConn.ExpectSet("", "", time.Duration(0)).RedisNil()

	mw := AsyncMw(bgResponses, cacheConn)
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
	redisClient, mockedCacheConn := redismock.NewClientMock()
	cacheConn := &responsecache.Cache{
		Client: redisClient,
	}
	mockedCacheConn.ExpectSet("", "", time.Duration(0)).RedisNil()

	mw := AsyncMw(bgResponses, cacheConn)
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
