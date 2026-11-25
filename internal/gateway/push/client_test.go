package push_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"
	"uuid"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace/noop"

	"github.com/sashabeketov/avito-go-labworks/internal/gateway/push"
	"github.com/sashabeketov/avito-go-labworks/internal/model"
)

func TestRequestPositionHTTPContract(t *testing.T) {
	trip := model.StaleTrip{TripID: uuid.New(), DriverID: uuid.New()}
	for _, code := range []int{http.StatusAccepted, http.StatusTooManyRequests, http.StatusServiceUnavailable} {
		t.Run(http.StatusText(code), func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, http.MethodPost, r.Method)
				assert.Equal(t, "/api/v1/push", r.URL.Path)
				assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
				var body struct {
					RequestID   string            `json:"request_id"`
					RecipientID string            `json:"recipient_id"`
					Kind        string            `json:"kind"`
					Data        map[string]string `json:"data"`
				}
				assert.NoError(t, json.NewDecoder(r.Body).Decode(&body))
				assert.NotEmpty(t, body.RequestID)
				assert.Equal(t, trip.DriverID.String(), body.RecipientID)
				assert.Equal(t, "REQUEST_POSITION", body.Kind)
				assert.Equal(t, map[string]string{"trip_id": trip.TripID.String()}, body.Data)
				w.WriteHeader(code)
			}))
			defer server.Close()
			client, err := push.New(server.URL, time.Second, noop.NewTracerProvider(), propagation.TraceContext{})
			require.NoError(t, err)
			defer client.Close()

			err = client.RequestPosition(t.Context(), trip)
			if code == http.StatusAccepted {
				require.NoError(t, err)
			} else {
				var statusErr *push.StatusError
				require.ErrorAs(t, err, &statusErr)
				require.Equal(t, code, statusErr.Code)
			}

			server.Close()
			var transportErr *url.Error
			require.ErrorAs(t, client.RequestPosition(t.Context(), trip), &transportErr)
		})
	}
}
