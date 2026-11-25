package push

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
	"uuid"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"

	"github.com/sashabeketov/avito-go-labworks/internal/model"
)

type Client struct {
	endpoint  string
	http      *http.Client
	transport *http.Transport
}

type StatusError struct {
	Code int
}

func (e *StatusError) Error() string {
	return fmt.Sprintf("Push Service returned HTTP %d", e.Code)
}

func New(baseURL string, timeout time.Duration, tp trace.TracerProvider, propagator propagation.TextMapPropagator) (*Client, error) {
	endpoint, err := url.JoinPath(baseURL, "api", "v1", "push")
	if err != nil {
		return nil, fmt.Errorf("build Push Service endpoint: %w", err)
	}
	transport := &http.Transport{IdleConnTimeout: timeout}
	return &Client{
		endpoint:  endpoint,
		transport: transport,
		http: &http.Client{
			Timeout: timeout,
			Transport: otelhttp.NewTransport(transport,
				otelhttp.WithTracerProvider(tp),
				otelhttp.WithPropagators(propagator),
			),
			CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
		},
	}, nil
}

func (c *Client) RequestPosition(ctx context.Context, trip model.StaleTrip) error {
	body, err := json.Marshal(struct {
		RequestID   string            `json:"request_id"`
		RecipientID string            `json:"recipient_id"`
		Kind        string            `json:"kind"`
		Data        map[string]string `json:"data"`
	}{
		RequestID:   uuid.New().String(),
		RecipientID: trip.DriverID.String(),
		Kind:        "REQUEST_POSITION",
		Data:        map[string]string{"trip_id": trip.TripID.String()},
	})
	if err != nil {
		return fmt.Errorf("encode position request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("build position request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("send position request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 8<<10))
	if resp.StatusCode != http.StatusAccepted {
		return &StatusError{Code: resp.StatusCode}
	}
	return nil
}

func (c *Client) Close() {
	c.transport.CloseIdleConnections()
}
