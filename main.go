package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Sugar-pack/users-manager/pkg/logging"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/Sugar-pack/rest-server/internal/config"
	"github.com/Sugar-pack/rest-server/internal/webapi"
)

const shutdownTime = 5 * time.Minute

// @title Swagger Example API
// @version 1.0
// @description This is a sample server Petstore server.
// @termsOfService http://swagger.io/terms/

// @contact.name API Support
// @contact.url http://www.swagger.io/support
// @contact.email support@swagger.io

// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html

// @host petstore.swagger.io
// @BasePath /v2

func main() {
	logger := logging.GetLogger()
	ctx := logging.WithContext(context.Background(), logger)

	appConfig, err := config.GetConfig()
	if err != nil {
		log.Fatal(err)

		return
	}

	userConn, err := grpc.Dial(appConfig.User.Address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatal(err)

		return
	}
	defer func(userConn *grpc.ClientConn) {
		errClose := userConn.Close()
		if errClose != nil {
			log.Fatal(errClose)
		}
	}(userConn)

	orderConn, err := grpc.Dial(appConfig.Order.Address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatal(err)

		return
	}
	defer func(orderConn *grpc.ClientConn) {
		errClose := orderConn.Close()
		if errClose != nil {
			log.Fatal(errClose)
		}
	}(orderConn)

	handler := webapi.NewHandler(userConn, orderConn)
	router := webapi.CreateRouter(logger, handler)
	server := http.Server{
		Addr:    appConfig.App.Bind,
		Handler: router,
	}

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(shutdown)

	go func() {
		logger.Info("Server is listening on ", appConfig.App.Bind)
		errLaS := server.ListenAndServe()
		if errLaS != nil && errors.Is(errLaS, http.ErrServerClosed) {
			logger.Fatal(errLaS)
		}
	}()

	<-shutdown

	logger.Info("Shutdown signal received")

	ctx, cancel := context.WithTimeout(context.Background(), shutdownTime)
	defer func() {
		cancel()
	}()

	if errShutdown := server.Shutdown(ctx); errShutdown != nil {
		logger.WithError(errShutdown).Fatal("Server shutdown error")
	}

	logger.Info("Server stopped gracefully")
}
