package positions

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
	"uuid"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

type Pool struct {
	scanner *Scanner
	push    pushGateway
	workers int
	timeout time.Duration
	active  atomic.Int64
	metrics poolMetrics

	stopScan    context.CancelFunc
	stopWorkers context.CancelFunc
	done        chan struct{}
}

func NewPool(scanner *Scanner, push pushGateway, workers int, timeout time.Duration, meter metric.Meter) (*Pool, error) {
	if workers <= 0 || timeout <= 0 {
		return nil, errors.New("POSITION_WORKERS and PUSH_REQUEST_TIMEOUT must be positive")
	}
	p := &Pool{scanner: scanner, push: push, workers: workers, timeout: timeout}
	metrics, err := newPoolMetrics(meter, p.queueSize, p.active.Load)
	if err != nil {
		return nil, fmt.Errorf("initialize position worker metrics: %w", err)
	}
	p.metrics = metrics
	return p, nil
}

func (p *Pool) queueSize() int64 {
	return int64(len(p.scanner.queue.jobs))
}

// Start begins the scanner and fixed set of workers. The application owns Shutdown.
func (p *Pool) Start(ctx context.Context) {
	p.start(ctx, p.scanner.Run)
}

func (p *Pool) start(ctx context.Context, scan func(context.Context)) {
	ctx = context.WithoutCancel(ctx)
	scanCtx, stopScan := context.WithCancel(ctx)
	workerCtx, stopWorkers := context.WithCancel(ctx)
	p.stopScan, p.stopWorkers = stopScan, stopWorkers
	p.done = make(chan struct{})

	var group sync.WaitGroup
	group.Go(func() { scan(scanCtx) })
	for range p.workers {
		group.Go(func() { p.work(workerCtx) })
	}
	go func() {
		group.Wait()
		close(p.done)
	}()
}

// Shutdown stops production first. Workers drain the closed queue until the deadline.
func (p *Pool) Shutdown(ctx context.Context) error {
	p.stopScan()
	defer p.stopWorkers()

	select {
	case <-p.done:
		return nil
	case <-ctx.Done():
		p.stopWorkers()
		<-p.done
		return ctx.Err()
	}
}

func (p *Pool) PositionSaved(ctx context.Context, tripID uuid.UUID) {
	p.scanner.queue.cancelWaiting(tripID)
	p.metrics.saved.Add(ctx, 1)
}

func (p *Pool) work(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case task, ok := <-p.scanner.queue.jobs:
			if !ok || ctx.Err() != nil {
				return
			}
			p.process(ctx, task)
		}
	}
}

func (p *Pool) process(ctx context.Context, task *job) {
	ctx, span := p.scanner.tracer.Start(ctx, "positions.request",
		trace.WithNewRoot(), trace.WithLinks(trace.Link{SpanContext: task.scanSpan}))
	defer span.End()

	if !p.scanner.queue.claim(task) {
		p.metrics.requests.Add(ctx, 1, metric.WithAttributes(attribute.String("result", "cancelled")))
		return
	}
	defer p.scanner.queue.complete(task)

	p.active.Add(1)
	defer p.active.Add(-1)
	requestCtx, cancel := context.WithTimeout(ctx, p.timeout)
	defer cancel()

	start := time.Now()
	err := p.push.RequestPosition(requestCtx, task.trip)
	elapsed := time.Since(start).Seconds()
	result := "success"
	if err != nil {
		result = "error"
		span.RecordError(err)
		span.SetStatus(codes.Error, "request position")
		p.scanner.logger.WarnContext(ctx, "position request failed", "trip_id", task.trip.TripID, "error", err)
	}
	labels := metric.WithAttributes(attribute.String("result", result))
	p.metrics.requests.Add(ctx, 1, labels)
	p.metrics.duration.Record(ctx, elapsed, labels)
}
