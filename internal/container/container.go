package container

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/sashabeketov/avito-go-labworks/internal/config"
	apphttp "github.com/sashabeketov/avito-go-labworks/internal/http"
	"github.com/sashabeketov/avito-go-labworks/internal/http/health"
	createhttp "github.com/sashabeketov/avito-go-labworks/internal/http/trip/create"
	finishhttp "github.com/sashabeketov/avito-go-labworks/internal/http/trip/finish"
	gethttp "github.com/sashabeketov/avito-go-labworks/internal/http/trip/get"
	"github.com/sashabeketov/avito-go-labworks/internal/infra/clock"
	"github.com/sashabeketov/avito-go-labworks/internal/infra/identity"
	"github.com/sashabeketov/avito-go-labworks/internal/infra/postgres"
	"github.com/sashabeketov/avito-go-labworks/internal/infra/postgres/transaction"
	"github.com/sashabeketov/avito-go-labworks/internal/infra/telemetry"
	"github.com/sashabeketov/avito-go-labworks/internal/readiness"
	triprepo "github.com/sashabeketov/avito-go-labworks/internal/repository/trip"
	createuc "github.com/sashabeketov/avito-go-labworks/internal/usecase/trip/create"
	finishuc "github.com/sashabeketov/avito-go-labworks/internal/usecase/trip/finish"
	getuc "github.com/sashabeketov/avito-go-labworks/internal/usecase/trip/get"
)

type App struct {
	Handler   http.Handler
	Logger    *slog.Logger
	pool      *pgxpool.Pool
	telemetry *telemetry.Telemetry
}

func New(ctx context.Context, cfg config.Config) (*App, error) {
	logger := telemetry.NewLogger(cfg.Log.LogLevel)
	sdk, err := telemetry.New(ctx, cfg.Telemetry)
	if err != nil {
		return nil, fmt.Errorf("initialize telemetry: %w", err)
	}
	pool, err := postgres.CreateNewPool(ctx, cfg.Database, sdk.TracerProvider, sdk.MeterProvider)
	if err != nil {
		_ = sdk.Shutdown(context.WithoutCancel(ctx))
		return nil, fmt.Errorf("open PostgreSQL: %w", err)
	}

	if err := postgres.RegisterActiveTripsGauge(sdk.Meter("trip-service/trips"), pool, cfg.Database.QueryTimeout); err != nil {
		_ = sdk.Shutdown(context.WithoutCancel(ctx))
		pool.Close()
		return nil, fmt.Errorf("initialize active trips gauge: %w", err)
	}

	events, err := telemetry.NewTripEvents(sdk.Meter("trip-service/trips"), logger)
	if err != nil {
		_ = sdk.Shutdown(context.WithoutCancel(ctx))
		pool.Close()
		return nil, fmt.Errorf("initialize trip metrics: %w", err)
	}
	tripRepository := triprepo.NewTripRepository(pool, cfg.Database.QueryTimeout)
	historyRepository := triprepo.NewHistoryRepository(pool, cfg.Database.QueryTimeout)
	txManager := transaction.NewTxManager(pool, cfg.Database.QueryTimeout)
	clock := clock.System{}
	ids := identity.Generator{}

	createTrip := createuc.New(tripRepository, historyRepository, txManager, clock, ids, events)
	getTrip := getuc.New(tripRepository)
	finishTrip := finishuc.New(tripRepository, historyRepository, txManager, clock, events)
	ready := readiness.CreateReadinessCheckers(postgres.NewPostgresChecker(pool, cfg.Database.PingTimeout))

	handler, err := apphttp.NewRouter(
		createhttp.New(createTrip, logger),
		gethttp.New(getTrip, logger),
		finishhttp.New(finishTrip, logger),
		health.New(ready),
		logger, sdk.TracerProvider, sdk.MeterProvider, sdk.Propagator,
	)
	if err != nil {
		_ = sdk.Shutdown(context.WithoutCancel(ctx))
		pool.Close()
		return nil, fmt.Errorf("initialize HTTP router: %w", err)
	}
	return &App{Handler: handler, Logger: logger, pool: pool, telemetry: sdk}, nil
}

func (a *App) Close(ctx context.Context) {
	if err := a.telemetry.Shutdown(ctx); err != nil {
		a.Logger.WarnContext(ctx, "telemetry shutdown failed", "error", err)
	}
	a.pool.Close()
}
