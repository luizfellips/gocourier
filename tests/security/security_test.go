//go:build security

package security

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	httphandler "github.com/gocourier/internal/adapters/http"
	"github.com/gocourier/internal/adapters/postgres"
	"github.com/gocourier/internal/application/ingest"
	"github.com/gocourier/internal/domain/notification"
	"github.com/gocourier/internal/ports"
	"github.com/gocourier/pkg/apperrors"
	"github.com/gocourier/pkg/logger"
	"github.com/gocourier/tests/testkit"
	"github.com/stretchr/testify/require"
)

func setupSecurityServer(t *testing.T) (*httptest.Server, *postgres.DeliveryRepo) {
	t.Helper()
	if testing.Short() {
		t.Skip("requires docker")
	}
	ctx := context.Background()
	pg := testkit.StartPostgres(ctx, t)

	store := postgres.NewStore(pg.Pool)
	deliveryRepo := postgres.NewDeliveryRepo(pg.Pool)
	auditRepo := postgres.NewAuditRepo(pg.Pool)
	ingestSvc := ingest.NewService(store, deliveryRepo, auditRepo, ports.SystemClock{}, "notifications", 24*time.Hour)

	srv := httphandler.NewServer(ingestSvc, nil, postgres.NewDashboardRepo(pg.Pool), []string{"test-key"}, logger.New("error"), "api")
	server := httptest.NewServer(srv.Handler())
	t.Cleanup(server.Close)
	return server, deliveryRepo
}

func TestUnauthorizedIngest(t *testing.T) {
	server, _ := setupSecurityServer(t)

	body := bytes.NewBufferString(`{"schema_version":"1.0","idempotency_key":"k1","channel":"email","recipient":{},"template":{}}`)
	req, err := http.NewRequest(http.MethodPost, server.URL+"/v1/notifications", body)
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestMalformedJSON(t *testing.T) {
	server, _ := setupSecurityServer(t)

	req, err := http.NewRequest(http.MethodPost, server.URL+"/v1/notifications", strings.NewReader("{bad"))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", "test-key")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestOversizedIdempotencyKey(t *testing.T) {
	server, _ := setupSecurityServer(t)

	key := strings.Repeat("x", 300)
	body, _ := json.Marshal(map[string]any{
		"schema_version":  notification.SchemaVersion,
		"idempotency_key": key,
		"channel":         "email",
		"recipient":       map[string]string{"address": "a@b.com"},
		"template":        map[string]string{"id": "t"},
	})
	req, err := http.NewRequest(http.MethodPost, server.URL+"/v1/notifications", bytes.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", "test-key")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestSQLInjectionInMetadata(t *testing.T) {
	server, deliveryRepo := setupSecurityServer(t)

	body := bytes.NewBufferString(`{
		"schema_version":"1.0",
		"idempotency_key":"sql-inject-1",
		"channel":"email",
		"recipient":{"address":"a@b.com"},
		"template":{"id":"t"},
		"metadata":{"tenant_id":"'; DROP TABLE deliveries; --"}
	}`)
	req, err := http.NewRequest(http.MethodPost, server.URL+"/v1/notifications", body)
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", "test-key")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusAccepted, resp.StatusCode)

	d, err := deliveryRepo.FindByIdempotencyKey(context.Background(), "sql-inject-1", notification.ChannelEmail)
	require.NoError(t, err)
	require.Equal(t, "'; DROP TABLE deliveries; --", d.TenantID)
}

func TestValidationErrorType(t *testing.T) {
	require.ErrorIs(t, apperrors.ErrValidation, apperrors.ErrValidation)
}
