package http

import (
	"log/slog"
	stdhttp "net/http"

	"github.com/go-chi/chi/v5"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"

	api "github.com/sashabeketov/avito-go-labworks/internal/generated"
	"github.com/sashabeketov/avito-go-labworks/internal/http/domainerror"
	"github.com/sashabeketov/avito-go-labworks/internal/http/health"
	"github.com/sashabeketov/avito-go-labworks/internal/http/middleware"
	"github.com/sashabeketov/avito-go-labworks/internal/http/trip/create"
	"github.com/sashabeketov/avito-go-labworks/internal/http/trip/finish"
	"github.com/sashabeketov/avito-go-labworks/internal/http/trip/get"
)

type routerHandler struct {
	api.Unimplemented
	create *create.Handler
	get    *get.Handler
	finish *finish.Handler
	health *health.Handler
}

func NewRouter(
	create *create.Handler,
	get *get.Handler,
	finish *finish.Handler,
	health *health.Handler,
	logger *slog.Logger,
	tp trace.TracerProvider,
	mp metric.MeterProvider,
	propagator propagation.TextMapPropagator,
) (stdhttp.Handler, error) {
	router := chi.NewRouter()
	metrics, err := middleware.Metrics(mp.Meter("trip-service/http"), router)
	if err != nil {
		return nil, err
	}
	router.Use(
		middleware.Recover(logger),
		middleware.Tracing(tp, mp, propagator),
		middleware.Logging(logger),
		metrics,
	)
	handler := &routerHandler{
		create: create,
		get:    get,
		finish: finish,
		health: health,
	}
	return api.HandlerWithOptions(handler, api.ChiServerOptions{
		BaseRouter: router,
		ErrorHandlerFunc: func(w stdhttp.ResponseWriter, r *stdhttp.Request, _ error) {
			domainerror.WriteInvalidRequest(w, r, logger)
		},
	}), nil
}

func (h *routerHandler) CreateTrip(
	w stdhttp.ResponseWriter,
	r *stdhttp.Request,
	params api.CreateTripParams,
) {
	h.create.Handle(w, r, params)
}

func (h *routerHandler) GetTrip(
	w stdhttp.ResponseWriter,
	r *stdhttp.Request,
	id api.TripId,
) {
	h.get.Handle(w, r, id)
}

func (h *routerHandler) FinishTrip(
	w stdhttp.ResponseWriter,
	r *stdhttp.Request,
	id api.TripId,
) {
	h.finish.Handle(w, r, id)
}

func (h *routerHandler) Health(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	h.health.Health(w, r)
}

func (h *routerHandler) Ready(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	h.health.Ready(w, r)
}

var _ api.ServerInterface = (*routerHandler)(nil)
