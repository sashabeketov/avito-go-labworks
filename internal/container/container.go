package container

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/sashabeketov/avito-go-labworks/internal/config"
	"github.com/sashabeketov/avito-go-labworks/internal/gateway/push"
	apphttp "github.com/sashabeketov/avito-go-labworks/internal/http"
	"github.com/sashabeketov/avito-go-labworks/internal/http/health"
	positioncreatehttp "github.com/sashabeketov/avito-go-labworks/internal/http/position/create"
	positionlisthttp "github.com/sashabeketov/avito-go-labworks/internal/http/position/list"
	createhttp "github.com/sashabeketov/avito-go-labworks/internal/http/trip/create"
	finishhttp "github.com/sashabeketov/avito-go-labworks/internal/http/trip/finish"
	gethttp "github.com/sashabeketov/avito-go-labworks/internal/http/trip/get"
	"github.com/sashabeketov/avito-go-labworks/internal/infra/clock"
	"github.com/sashabeketov/avito-go-labworks/internal/infra/identity"
	"github.com/sashabeketov/avito-go-labworks/internal/infra/postgres"
	"github.com/sashabeketov/avito-go-labworks/internal/infra/postgres/transaction"
	"github.com/sashabeketov/avito-go-labworks/internal/infra/telemetry"
	"github.com/sashabeketov/avito-go-labworks/internal/readiness"
	positionrepo "github.com/sashabeketov/avito-go-labworks/internal/repository/position"
	triprepo "github.com/sashabeketov/avito-go-labworks/internal/repository/trip"
	positioncreateuc "github.com/sashabeketov/avito-go-labworks/internal/usecase/position/create"
	positionlistuc "github.com/sashabeketov/avito-go-labworks/internal/usecase/position/list"
	createuc "github.com/sashabeketov/avito-go-labworks/internal/usecase/trip/create"
	finishuc "github.com/sashabeketov/avito-go-labworks/internal/usecase/trip/finish"
	getuc "github.com/sashabeketov/avito-go-labworks/internal/usecase/trip/get"
	"github.com/sashabeketov/avito-go-labworks/internal/worker/positions"
)

type App struct {
	Handler   http.Handler
	Logger    *slog.Logger
	pool      *pgxpool.Pool
	telemetry *telemetry.Telemetry
	positions *positions.Pool
	push      *push.Client
}

func New(ctx context.Context, cfg config.Config) (*App, error) {
	logger := telemetry.NewLogger(cfg.Log.LogLevel)
	sdk, err := telemetry.New(ctx, cfg.Telemetry)
	if err != nil {
		return nil, fmt.Errorf("initialize telemetry: %w", err)
	}
	app := &App{Logger: logger, telemetry: sdk}
	started := false
	defer func() {
		if !started {
			closeCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), cfg.App.ShutdownTimeout)
			defer cancel()
			_ = app.closeResources(closeCtx)
		}
	}()

	pool, err := postgres.CreateNewPool(ctx, cfg.Database, sdk.TracerProvider, sdk.MeterProvider)
	if err != nil {
		return nil, fmt.Errorf("open PostgreSQL: %w", err)
	}
	app.pool = pool

	if err := postgres.RegisterActiveTripsGauge(sdk.Meter("trip-service/trips"), pool, cfg.Database.QueryTimeout); err != nil {
		return nil, fmt.Errorf("initialize active trips gauge: %w", err)
	}

	events, err := telemetry.NewTripEvents(sdk.Meter("trip-service/trips"), logger)
	if err != nil {
		return nil, fmt.Errorf("initialize trip metrics: %w", err)
	}
	tripRepository := triprepo.NewTripRepository(pool, cfg.Database.QueryTimeout)
	positionRepository := positionrepo.New(pool, cfg.Database.QueryTimeout)
	historyRepository := triprepo.NewHistoryRepository(pool, cfg.Database.QueryTimeout)
	txManager := transaction.NewTxManager(pool, cfg.Database.QueryTimeout)
	clock := clock.System{}
	ids := identity.Generator{}

	pushClient, err := push.New(cfg.Positions.PushHTTPURL, cfg.Positions.PushTimeout, sdk.TracerProvider, sdk.Propagator)
	if err != nil {
		return nil, fmt.Errorf("initialize Push Service client: %w", err)
	}
	app.push = pushClient
	meter := sdk.Meter("trip-service/positions")
	scanner, err := positions.NewScanner(tripRepository, positions.ScanConfig{
		Interval: cfg.Positions.CheckInterval, StaleAfter: cfg.Positions.StaleAfter, QueueSize: cfg.Positions.QueueSize,
	}, logger, sdk.Tracer("trip-service/positions"), meter)
	if err != nil {
		return nil, fmt.Errorf("initialize position scanner: %w", err)
	}
	positionPool, err := positions.NewPool(scanner, pushClient, cfg.Positions.Workers, cfg.Positions.PushTimeout, meter)
	if err != nil {
		return nil, fmt.Errorf("initialize position pool: %w", err)
	}

	createTrip := createuc.New(tripRepository, historyRepository, txManager, clock, ids, events)
	getTrip := getuc.New(tripRepository)
	finishTrip := finishuc.New(tripRepository, historyRepository, txManager, clock, events)
	createPosition := positioncreateuc.New(positionRepository, txManager, clock, positionPool)
	listPositions := positionlistuc.New(tripRepository, positionRepository)
	ready := readiness.CreateReadinessCheckers(postgres.NewPostgresChecker(pool, cfg.Database.PingTimeout))

	handler, err := apphttp.NewRouter(
		createhttp.New(createTrip, logger),
		gethttp.New(getTrip, logger),
		finishhttp.New(finishTrip, logger),
		positioncreatehttp.New(createPosition, logger),
		positionlisthttp.New(listPositions, logger),
		health.New(ready),
		logger, sdk.TracerProvider, sdk.MeterProvider, sdk.Propagator,
	)
	if err != nil {
		return nil, fmt.Errorf("initialize HTTP router: %w", err)
	}
	app.Handler = handler
	app.positions = positionPool
	positionPool.Start(ctx)
	started = true
	return app, nil
}

func (a *App) Close(ctx context.Context) error {
	err := a.positions.Shutdown(ctx)
	if err != nil {
		err = fmt.Errorf("shutdown position pool: %w", err)
	}
	return errors.Join(err, a.closeResources(ctx))
}

func (a *App) closeResources(ctx context.Context) error {
	if a.push != nil {
		a.push.Close()
	}
	err := a.telemetry.Shutdown(ctx)
	if a.pool != nil {
		a.pool.Close()
	}
	if err != nil {
		return fmt.Errorf("shutdown telemetry: %w", err)
	}
	return nil
}
