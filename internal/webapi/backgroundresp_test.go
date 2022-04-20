package webapi

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Sugar-pack/rest-server/internal/responsecache"
	"github.com/Sugar-pack/users-manager/pkg/logging"
	"github.com/go-chi/chi/v5"
	"github.com/go-redis/redismock/v8"
	"github.com/stretchr/testify/assert"
)

func TestCachedResponse_OK(t *testing.T) {
	ctx := context.Background()
	logger := logging.GetLogger()
	ctx = logging.WithContext(ctx, logger)
	bgID := "uniq_id"
	mockedBody := []byte("from cache")
	redisClient, mockedCacheConn := redismock.NewClientMock()
	cacheConn := &responsecache.Cache{
		Client: redisClient,
	}
	mockedCachedResp := &responsecache.HTTPResponse{
		Code:    http.StatusOK,
		Headers: nil,
		Body:    mockedBody,
	}
	mockedRedisValue, err := json.Marshal(mockedCachedResp)
	if err != nil {
		t.Fatal(err)
	}
	mockedCacheConn.ExpectGet(bgID).SetVal(string(mockedRedisValue))
	mockedCacheConn.ExpectDel(bgID).SetVal(0)

	handlerFn := CachedResponse(cacheConn)
	testRecorder := httptest.NewRecorder()
	testRequest := httptest.NewRequest(http.MethodGet, "/bg-responses/{bg_id}", nil)
	testRequest = testRequest.WithContext(ctx)

	newChiCtx := chi.NewRouteContext()
	newChiCtx.URLParams.Add("bg_id", bgID)
	testRequest = testRequest.WithContext(context.WithValue(testRequest.Context(), chi.RouteCtxKey, newChiCtx))

	handlerFn.ServeHTTP(testRecorder, testRequest)

	assert.NoError(t, mockedCacheConn.ExpectationsWereMet(), "all redis expectations should be met")

	gotHttpCode := testRecorder.Code
	gotResponseBody, err := ioutil.ReadAll(testRecorder.Body)
	if err != nil {
		t.Fatal(err)
	}
	expectedHTTPCode := http.StatusOK
	expectedBody := mockedBody
	assert.Equal(t, expectedHTTPCode, gotHttpCode)
	assert.Equal(t, expectedBody, gotResponseBody)
}

func TestCachedResponse_NotFound(t *testing.T) {
	ctx := context.Background()
	logger := logging.GetLogger()
	ctx = logging.WithContext(ctx, logger)
	bgID := "uniq_id"
	redisClient, mockedCacheConn := redismock.NewClientMock()
	cacheConn := &responsecache.Cache{
		Client: redisClient,
	}
	mockedCacheConn.ExpectGet(bgID).RedisNil()

	handlerFn := CachedResponse(cacheConn)
	testRecorder := httptest.NewRecorder()
	testRequest := httptest.NewRequest(http.MethodGet, "/bg-responses/{bg_id}", nil)
	testRequest = testRequest.WithContext(ctx)

	newChiCtx := chi.NewRouteContext()
	newChiCtx.URLParams.Add("bg_id", bgID)
	testRequest = testRequest.WithContext(context.WithValue(testRequest.Context(), chi.RouteCtxKey, newChiCtx))

	handlerFn.ServeHTTP(testRecorder, testRequest)

	assert.NoError(t, mockedCacheConn.ExpectationsWereMet(), "all redis expectations should be met")

	gotHttpCode := testRecorder.Code
	gotResponseBody, err := ioutil.ReadAll(testRecorder.Body)
	if err != nil {
		t.Fatal(err)
	}
	expectedHTTPCode := http.StatusNotFound
	expectedBody := []byte("background id not found")
	assert.Equal(t, expectedHTTPCode, gotHttpCode)
	assert.Equal(t, expectedBody, gotResponseBody)
}
