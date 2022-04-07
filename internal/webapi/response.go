package webapi

import (
	"context"
	"net/http"

	"github.com/Sugar-pack/users-manager/pkg/logging"
)

func BadRequest(ctx context.Context, writer http.ResponseWriter, msg string) {
	logger := logging.FromContext(ctx)
	writer.WriteHeader(http.StatusBadRequest)
	_, wErr := writer.Write([]byte(msg))
	if wErr != nil {
		logger.WithError(wErr).Error("Error while writing response")
	}
}

func InternalError(ctx context.Context, writer http.ResponseWriter, s string) {
	logger := logging.FromContext(ctx)
	writer.WriteHeader(http.StatusInternalServerError)
	_, wErr := writer.Write([]byte(s))
	if wErr != nil {
		logger.WithError(wErr).Error("Error while writing response")
	}
}

func StatusOk(ctx context.Context, writer http.ResponseWriter, s string) {
	logger := logging.FromContext(ctx)
	writer.WriteHeader(http.StatusOK)
	_, wErr := writer.Write([]byte(s))
	if wErr != nil {
		logger.WithError(wErr).Error("Error while writing response")
	}
}
