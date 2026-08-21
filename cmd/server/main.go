package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/wyw14/cry-071/internal/application"
	"github.com/wyw14/cry-071/internal/config"
	"github.com/wyw14/cry-071/internal/platform/files"
	"github.com/wyw14/cry-071/internal/platform/identifier"
	"github.com/wyw14/cry-071/internal/platform/notification"
	"github.com/wyw14/cry-071/internal/repository/memory"
	"github.com/wyw14/cry-071/internal/repository/postgres"
	"github.com/wyw14/cry-071/internal/service"
	httpapi "github.com/wyw14/cry-071/internal/transport/http"
	"go.uber.org/zap"
)

type repositorySet interface {
	application.Repositories
	application.TransactionManager
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load configuration: %w", err)
	}
	logger, err := buildLogger(cfg.Environment)
	if err != nil {
		return fmt.Errorf("build logger: %w", err)
	}
	defer func() { _ = logger.Sync() }()
	startupContext, cancel := context.WithTimeout(context.Background(), cfg.RequestTimeout)
	defer cancel()
	repositories, closeStore, ready, err := buildRepositories(startupContext, cfg)
	if err != nil {
		return err
	}
	defer closeStore()
	objectStore, err := files.NewLocalStore(cfg.AttachmentRoot)
	if err != nil {
		return fmt.Errorf("open attachment store: %w", err)
	}
	tokenCodec, err := service.NewTokenCodec(cfg.QueryTokenPepper)
	if err != nil {
		return err
	}
	idGenerator := identifier.Random{}
	services, err := application.NewServices(application.Dependencies{
		Transactions: repositories, Repositories: repositories, Clock: wallClock{}, IDs: idGenerator,
		Tokens: tokenCodec, Notifications: notification.NewLocalSink(), Objects: objectStore,
		Redactor: service.Redactor{}, Duplicates: service.NewDuplicateMatcher(repositories),
	})
	if err != nil {
		return fmt.Errorf("build services: %w", err)
	}
	router := httpapi.NewRouter(httpapi.RouterDependencies{Services: services, Config: cfg, Logger: logger, Ready: ready})
	server := &http.Server{
		Addr: cfg.HTTPAddress, Handler: router, ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout: cfg.RequestTimeout + time.Second, WriteTimeout: cfg.RequestTimeout + time.Second, IdleTimeout: 60 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}
	serverErrors := make(chan error, 1)
	go func() {
		logger.Info("server_started", zap.String("address", cfg.HTTPAddress), zap.String("store", cfg.StoreMode))
		serverErrors <- server.ListenAndServe()
	}()
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(signals)
	select {
	case signalValue := <-signals:
		logger.Info("shutdown_requested", zap.String("signal", signalValue.String()))
	case serverErr := <-serverErrors:
		if !errors.Is(serverErr, http.ErrServerClosed) {
			return fmt.Errorf("http server: %w", serverErr)
		}
	}
	shutdownContext, shutdownCancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer shutdownCancel()
	if err := server.Shutdown(shutdownContext); err != nil {
		_ = server.Close()
		return fmt.Errorf("graceful shutdown: %w", err)
	}
	logger.Info("server_stopped")
	return nil
}

func buildRepositories(ctx context.Context, cfg config.Config) (repositorySet, func(), func(context.Context) error, error) {
	if cfg.StoreMode == "postgres" {
		store, err := postgres.Open(ctx, cfg.DatabaseURL)
		if err != nil {
			return nil, func() {}, nil, err
		}
		return store, store.Close, store.Ping, nil
	}
	store := memory.NewStore()
	if cfg.SeedDemoData {
		store = memory.NewStoreWithDemo(time.Now().UTC())
	}
	return store, func() {}, func(context.Context) error { return nil }, nil
}

func buildLogger(environment string) (*zap.Logger, error) {
	if environment == "development" {
		return zap.NewDevelopment()
	}
	return zap.NewProduction()
}

type wallClock struct{}

func (wallClock) Now() time.Time { return time.Now().UTC() }
