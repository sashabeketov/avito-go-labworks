package main

import (
	"context"
	"fmt"
	"net/http"

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
	err := run(context.Background())
	if err != nil {
		fmt.Println(err)
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
		Addr:    cfg.HTTP.Addr,
		Handler: router,
	}

	if err := server.ListenAndServe(); err != nil {
		return fmt.Errorf("HTTP server: %w", err)
	}

	fmt.Println("NICE")
	return nil
}
