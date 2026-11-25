package positions

import (
	"sync"
	"uuid"

	"go.opentelemetry.io/otel/trace"

	"github.com/sashabeketov/avito-go-labworks/internal/model"
)

type enqueueResult string

const (
	queued    enqueueResult = "queued"
	duplicate enqueueResult = "duplicate"
	dropped   enqueueResult = "dropped"
)

type job struct {
	trip     model.StaleTrip
	scanSpan trace.SpanContext
	started  bool
}

type queue struct {
	jobs chan *job

	mu      sync.Mutex
	pending map[uuid.UUID]*job
}

func newQueue(size int) *queue {
	return &queue{
		jobs:    make(chan *job, size),
		pending: make(map[uuid.UUID]*job, size),
	}
}

func (q *queue) tryEnqueue(task job) enqueueResult {
	q.mu.Lock()
	defer q.mu.Unlock()

	id := task.trip.TripID
	if _, exists := q.pending[id]; exists {
		return duplicate
	}
	select {
	case q.jobs <- &task:
		q.pending[id] = &task
		return queued
	default:
		return dropped
	}
}

func (q *queue) claim(task *job) bool {
	q.mu.Lock()
	defer q.mu.Unlock()

	if q.pending[task.trip.TripID] != task {
		return false
	}
	task.started = true
	return true
}

func (q *queue) cancelWaiting(id uuid.UUID) {
	q.mu.Lock()
	defer q.mu.Unlock()

	if task := q.pending[id]; task != nil && !task.started {
		delete(q.pending, id)
	}
}

func (q *queue) complete(task *job) {
	q.mu.Lock()
	defer q.mu.Unlock()

	id := task.trip.TripID
	if q.pending[id] == task {
		delete(q.pending, id)
	}
}
