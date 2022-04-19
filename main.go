package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/Sugar-pack/rest-server/internal/responsecache"
	"github.com/Sugar-pack/users-manager/pkg/logging"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/Sugar-pack/rest-server/docs"
	"github.com/Sugar-pack/rest-server/internal/config"
	"github.com/Sugar-pack/rest-server/internal/webapi"
)

// @title Server Example
// @version 0.1
// @description This is a sample server.

func main() {
	logger := logging.GetLogger()
	ctx := logging.WithContext(context.Background(), logger)

	appConfig, err := config.GetConfig()
	if err != nil {
		log.Fatal(err)

		return
	}
	docs.SwaggerInfo.Host = appConfig.App.Bind
	shutdownTime := appConfig.Server.ShutdownTimeout

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

	cacheConn, err := responsecache.NewCache(ctx, responsecache.WithAddr(appConfig.App.CacheAddr))
	if err != nil {
		logger.WithError(err).Error("cache connect failed")
		return
	}

	handler := webapi.NewHandler(userConn, orderConn)
	router := webapi.CreateRouter(logger, handler, cacheConn)
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
