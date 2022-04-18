package webapi

import (
	"context"
	"net/http"

	"github.com/Sugar-pack/users-manager/pkg/logging"
)

const ErrMsgWritingResponse = "Error while writing response"

func BadRequest(ctx context.Context, writer http.ResponseWriter, msg string) {
	logger := logging.FromContext(ctx)
	writer.WriteHeader(http.StatusBadRequest)
	_, wErr := writer.Write([]byte(msg))
	if wErr != nil {
		logger.WithError(wErr).Error(ErrMsgWritingResponse)
	}
}

func InternalError(ctx context.Context, writer http.ResponseWriter, s string) {
	logger := logging.FromContext(ctx)
	writer.WriteHeader(http.StatusInternalServerError)
	_, wErr := writer.Write([]byte(s))
	if wErr != nil {
		logger.WithError(wErr).Error(ErrMsgWritingResponse)
	}
}

func StatusOk(ctx context.Context, writer http.ResponseWriter, s string) {
	logger := logging.FromContext(ctx)
	writer.WriteHeader(http.StatusOK)
	_, wErr := writer.Write([]byte(s))
	if wErr != nil {
		logger.WithError(wErr).Error(ErrMsgWritingResponse)
	}
}

func StatusAccepted(ctx context.Context, writer http.ResponseWriter, s, backgroundID string) {
	logger := logging.FromContext(ctx)
	writer.Header().Add("x-background-id", backgroundID)
	writer.WriteHeader(http.StatusAccepted)
	_, wErr := writer.Write([]byte(s))
	if wErr != nil {
		logger.WithError(wErr).Error(ErrMsgWritingResponse)
	}
}

func rawResponse(ctx context.Context, w http.ResponseWriter, httpCode int, httpHeaders http.Header, body []byte) {
	logger := logging.FromContext(ctx)
	for k, vs := range httpHeaders {
		for _, v := range vs {
			w.Header().Add(k, v)
		}
	}
	w.WriteHeader(httpCode)
	if _, wErr := w.Write(body); wErr != nil {
		logger.WithError(wErr).Error(ErrMsgWritingResponse)
	}
}
