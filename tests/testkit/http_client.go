package testkit

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/gocourier/internal/domain/notification"
	"github.com/stretchr/testify/require"
)

// HTTPClient wraps notification API calls for tests.
type HTTPClient struct {
	BaseURL string
	APIKey  string
	Client  *http.Client
}

func NewHTTPClient(baseURL, apiKey string) *HTTPClient {
	return &HTTPClient{
		BaseURL: baseURL,
		APIKey:  apiKey,
		Client:  http.DefaultClient,
	}
}

func (c *HTTPClient) PostNotification(ctx context.Context, t *testing.T, req notification.IngestRequest) (*notification.IngestResponse, int) {
	t.Helper()
	body, err := json.Marshal(req)
	require.NoError(t, err)

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+"/v1/notifications", bytes.NewReader(body))
	require.NoError(t, err)
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("X-API-Key", c.APIKey)

	resp, err := c.Client.Do(httpReq)
	require.NoError(t, err)
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	var result notification.IngestResponse
	if len(data) > 0 {
		require.NoError(t, json.Unmarshal(data, &result))
	}
	return &result, resp.StatusCode
}

func (c *HTTPClient) Replay(ctx context.Context, t *testing.T, deliveryID string) (int, map[string]any) {
	t.Helper()
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+"/v1/notifications/"+deliveryID+"/replay", nil)
	require.NoError(t, err)
	httpReq.Header.Set("X-API-Key", c.APIKey)

	resp, err := c.Client.Do(httpReq)
	require.NoError(t, err)
	defer resp.Body.Close()

	data, _ := io.ReadAll(resp.Body)
	var result map[string]any
	if len(data) > 0 {
		_ = json.Unmarshal(data, &result)
	}
	return resp.StatusCode, result
}

func (c *HTTPClient) Health(ctx context.Context, t *testing.T) int {
	t.Helper()
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, c.BaseURL+"/health", nil)
	require.NoError(t, err)
	resp, err := c.Client.Do(httpReq)
	require.NoError(t, err)
	defer resp.Body.Close()
	return resp.StatusCode
}
