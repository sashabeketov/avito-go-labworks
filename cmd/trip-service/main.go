package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/go-chi/chi/v5"
	"github.com/sashabeketov/avito-go-labworks/internal/config"
	httpcontroller "github.com/sashabeketov/avito-go-labworks/internal/controller/http"
	api "github.com/sashabeketov/avito-go-labworks/internal/generated"
	"github.com/sashabeketov/avito-go-labworks/internal/readiness"
	postgresrepo "github.com/sashabeketov/avito-go-labworks/internal/repository/postgres"
	"github.com/sashabeketov/avito-go-labworks/internal/service"
	"github.com/sashabeketov/avito-go-labworks/internal/storage/postgres"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := run(ctx); err != nil {
		slog.Error("trip service stopped", "error", err)
		os.Exit(1)
	}
}

func run(ctx context.Context) error {
	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	pool, err := postgres.CreateNewPool(ctx, cfg.Database)
	if err != nil {
		return fmt.Errorf("open PostgreSQL: %w", err)
	}
	defer pool.Close()

	postgresChecker := readiness.NewPostgresChecker(
		pool,
		cfg.Database.PingTimeout)

	readyAll := readiness.CreateReadinessCheckers(postgresChecker)

	tripRepository := postgresrepo.NewTripRepository(pool, cfg.Database.QueryTimeout)
	historyRepository := postgresrepo.NewTripStatusHistoryRepository(pool, cfg.Database.QueryTimeout)
	txManager := postgresrepo.NewTxManager(pool, cfg.Database.QueryTimeout)
	tripService := service.NewTripService(tripRepository, historyRepository, txManager)

	handler := httpcontroller.NewHandler(readyAll, tripService)

	router := chi.NewRouter()

	api.HandlerWithOptions(handler, api.ChiServerOptions{
		BaseRouter:       router,
		ErrorHandlerFunc: httpcontroller.WriteParameterError,
	})

	server := &http.Server{
		Addr:              cfg.HTTP.Addr,
		Handler:           router,
		ReadTimeout:       cfg.HTTP.ReadTimeout,
		ReadHeaderTimeout: cfg.HTTP.ReadHeaderTimeout,
		WriteTimeout:      cfg.HTTP.WriteTimeout,
		IdleTimeout:       cfg.HTTP.IdleTimeout,
	}

	serveErr := make(chan error, 1)
	go func() {
		serveErr <- server.ListenAndServe()
	}()

	select {
	case err := <-serveErr:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return fmt.Errorf("HTTP server: %w", err)
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(
			context.WithoutCancel(ctx),
			cfg.App.ShutdownTimeout,
		)
		defer cancel()

		if err := server.Shutdown(shutdownCtx); err != nil {
			return errors.Join(
				fmt.Errorf("shutdown HTTP server: %w", err),
				server.Close(),
			)
		}

		if err := <-serveErr; err != nil && !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("HTTP server: %w", err)
		}
		return nil
	}
}
