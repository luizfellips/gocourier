//go:build integration

package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/gocourier/internal/adapters/nats"
	"github.com/gocourier/internal/adapters/postgres"
	"github.com/gocourier/internal/adapters/providers/mock"
	httphandler "github.com/gocourier/internal/adapters/http"
	"github.com/gocourier/internal/application/dispatch"
	"github.com/gocourier/internal/application/ingest"
	"github.com/gocourier/internal/application/outbox"
	"github.com/gocourier/internal/application/replay"
	"github.com/gocourier/internal/domain/notification"
	"github.com/gocourier/internal/ports"
	"github.com/gocourier/pkg/logger"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

func startNATS(ctx context.Context, t *testing.T) string {
	t.Helper()
	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        "nats:2.10-alpine",
			Cmd:          []string{"-js", "-m", "8222"},
			ExposedPorts: []string{"4222/tcp"},
			WaitingFor: wait.ForLog("Listening for client connections").
				WithStartupTimeout(60 * time.Second),
		},
		Started: true,
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = container.Terminate(ctx) })

	host, err := container.Host(ctx)
	require.NoError(t, err)
	port, err := container.MappedPort(ctx, "4222")
	require.NoError(t, err)
	return fmt.Sprintf("nats://%s:%s", host, port.Port())
}

func TestFullLifecycle(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()

	pgContainer, err := tcpostgres.Run(ctx,
		"postgres:16-alpine",
		tcpostgres.WithDatabase("gocourier"),
		tcpostgres.WithUsername("gocourier"),
		tcpostgres.WithPassword("gocourier"),
		testcontainers.WithWaitStrategy(wait.ForLog("database system is ready to accept connections").WithOccurrence(2)),
	)
	require.NoError(t, err)
	t.Cleanup(func() { _ = pgContainer.Terminate(ctx) })

	pgURL, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)

	natsURL := startNATS(ctx, t)

	pool, err := postgres.NewPool(ctx, pgURL)
	require.NoError(t, err)
	t.Cleanup(pool.Close)

	broker, err := nats.NewBroker(nats.Config{
		URL:          natsURL,
		StreamPrefix: "notifications",
		MaxDeliver:   8,
		AckWait:      30 * time.Second,
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = broker.Close() })
	require.NoError(t, broker.EnsureStreams(ctx))

	log := logger.New("error")
	store := postgres.NewStore(pool)
	deliveryRepo := postgres.NewDeliveryRepo(pool)
	outboxRepo := postgres.NewOutboxRepo(pool)
	auditRepo := postgres.NewAuditRepo(pool)
	clock := ports.SystemClock{}

	ingestSvc := ingest.NewService(store, deliveryRepo, auditRepo, clock, "notifications", 24*time.Hour)
	dispatchSvc := dispatch.NewService(
		deliveryRepo, auditRepo, broker, mock.All(), clock, log,
		dispatch.Config{
			MaxAttempts: 8, RetryBase: time.Second, RetryMax: 30 * time.Minute,
			StreamPrefix: "notifications", CBThreshold: 5, CBWindow: 60 * time.Second, CBCooldown: 30 * time.Second,
		},
	)
	replaySvc := replay.NewService(dispatchSvc)

	publisher := outbox.NewPublisher(outboxRepo, broker, log, 100*time.Millisecond, 50)
	pubCtx, pubCancel := context.WithCancel(ctx)
	defer pubCancel()
	go func() { _ = publisher.Run(pubCtx) }()

	workerCtx, workerCancel := context.WithCancel(ctx)
	defer workerCancel()
	go func() { _ = dispatchSvc.Run(workerCtx, 10, nil) }()

	dashboardRepo := postgres.NewDashboardRepo(pool)
	srv := httphandler.NewServer(ingestSvc, replaySvc, dashboardRepo, []string{"test-key"}, log, "api")
	go func() {
		_ = srv.ListenAndServe(ctx, ":0")
	}()

	// Use ingest service directly for deterministic test without binding HTTP port race
	req := notification.IngestRequest{
		SchemaVersion:  "1.0",
		IdempotencyKey: fmt.Sprintf("integration-%d", time.Now().UnixNano()),
		Channel:        "email",
		Priority:       "normal",
		Recipient:      json.RawMessage(`{"address":"user@example.com"}`),
		Template:       json.RawMessage(`{"id":"welcome"}`),
	}
	resp, err := ingestSvc.Ingest(ctx, req)
	require.NoError(t, err)
	require.NotEmpty(t, resp.DeliveryID)

	require.NoError(t, publisher.FlushOnce(ctx))

	// Allow JetStream consumer to attach, then verify async dispatch
	time.Sleep(500 * time.Millisecond)

	payload, _ := json.Marshal(map[string]string{"delivery_id": resp.DeliveryID})
	require.Eventually(t, func() bool {
		if err := dispatchSvc.HandleMessage(ctx, "", payload, nil); err != nil {
			return false
		}
		d, err := deliveryRepo.FindByID(ctx, resp.DeliveryID)
		if err != nil {
			return false
		}
		return d.Status == notification.StatusSucceeded
	}, 10*time.Second, 200*time.Millisecond)

	// Idempotency: duplicate returns same delivery
	dup, err := ingestSvc.Ingest(ctx, req)
	require.NoError(t, err)
	require.True(t, dup.Duplicate)
	require.Equal(t, resp.DeliveryID, dup.DeliveryID)
}

func TestIngestHTTPEndpoint(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()
	pgContainer, err := tcpostgres.Run(ctx,
		"postgres:16-alpine",
		tcpostgres.WithDatabase("gocourier"),
		tcpostgres.WithUsername("gocourier"),
		tcpostgres.WithPassword("gocourier"),
		testcontainers.WithWaitStrategy(wait.ForLog("database system is ready to accept connections").WithOccurrence(2)),
	)
	require.NoError(t, err)
	t.Cleanup(func() { _ = pgContainer.Terminate(ctx) })
	pgURL, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)

	pool, err := postgres.NewPool(ctx, pgURL)
	require.NoError(t, err)
	t.Cleanup(pool.Close)

	store := postgres.NewStore(pool)
	deliveryRepo := postgres.NewDeliveryRepo(pool)
	auditRepo := postgres.NewAuditRepo(pool)
	clock := ports.SystemClock{}
	ingestSvc := ingest.NewService(store, deliveryRepo, auditRepo, clock, "notifications", 24*time.Hour)

	log := logger.New("error")
	dashboardRepo := postgres.NewDashboardRepo(pool)
	srv := httphandler.NewServer(ingestSvc, nil, dashboardRepo, []string{"test-key"}, log, "api")

	// Start on ephemeral port via httptest pattern
	listenerDone := make(chan string, 1)
	go func() {
		_ = srv.ListenAndServe(ctx, "127.0.0.1:18080")
	}()
	listenerDone <- "http://127.0.0.1:18080"
	baseURL := <-listenerDone
	time.Sleep(100 * time.Millisecond)

	body := map[string]any{
		"schema_version":  "1.0",
		"idempotency_key": "http-test-1",
		"channel":         "email",
		"recipient":       map[string]string{"address": "a@b.com"},
		"template":        map[string]string{"id": "t1"},
	}
	b, _ := json.Marshal(body)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/v1/notifications", bytes.NewReader(b))
	require.NoError(t, err)
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("X-API-Key", "test-key")

	resp, err := http.DefaultClient.Do(httpReq)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusAccepted, resp.StatusCode)

	data, _ := io.ReadAll(resp.Body)
	var result notification.IngestResponse
	require.NoError(t, json.Unmarshal(data, &result))
	require.NotEmpty(t, result.DeliveryID)
}

func TestPermanentFailureMovesToDLQ(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()
	pgContainer, err := tcpostgres.Run(ctx,
		"postgres:16-alpine",
		tcpostgres.WithDatabase("gocourier"),
		tcpostgres.WithUsername("gocourier"),
		tcpostgres.WithPassword("gocourier"),
		testcontainers.WithWaitStrategy(wait.ForLog("database system is ready to accept connections").WithOccurrence(2)),
	)
	require.NoError(t, err)
	t.Cleanup(func() { _ = pgContainer.Terminate(ctx) })
	pgURL, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)

	natsURL := startNATS(ctx, t)

	pool, err := postgres.NewPool(ctx, pgURL)
	require.NoError(t, err)
	t.Cleanup(pool.Close)

	broker, err := nats.NewBroker(nats.Config{
		URL: natsURL, StreamPrefix: "notifications", MaxDeliver: 3, AckWait: 10 * time.Second,
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = broker.Close() })
	require.NoError(t, broker.EnsureStreams(ctx))

	log := logger.New("error")
	store := postgres.NewStore(pool)
	deliveryRepo := postgres.NewDeliveryRepo(pool)
	outboxRepo := postgres.NewOutboxRepo(pool)
	auditRepo := postgres.NewAuditRepo(pool)
	clock := ports.SystemClock{}

	ingestSvc := ingest.NewService(store, deliveryRepo, auditRepo, clock, "notifications", 24*time.Hour)
	dispatchSvc := dispatch.NewService(
		deliveryRepo, auditRepo, broker, mock.All(), clock, log,
		dispatch.Config{
			MaxAttempts: 3, RetryBase: time.Millisecond, RetryMax: time.Second,
			StreamPrefix: "notifications", CBThreshold: 100, CBWindow: time.Second, CBCooldown: time.Second,
		},
	)

	pubCtx, pubCancel := context.WithCancel(ctx)
	defer pubCancel()
	go func() {
		publisher := outbox.NewPublisher(outboxRepo, broker, log, 50*time.Millisecond, 50)
		_ = publisher.Run(pubCtx)
	}()

	workerCtx, workerCancel := context.WithCancel(ctx)
	defer workerCancel()
	go func() { _ = dispatchSvc.Run(workerCtx, 5, nil) }()

	req := notification.IngestRequest{
		SchemaVersion:  "1.0",
		IdempotencyKey: fmt.Sprintf("dlq-%d", time.Now().UnixNano()),
		Channel:        "email",
		Recipient:      json.RawMessage(`{"address":"fail-permanent@example.com"}`),
		Template:       json.RawMessage(`{"id":"x"}`),
	}
	resp, err := ingestSvc.Ingest(ctx, req)
	require.NoError(t, err)

	publisher := outbox.NewPublisher(outboxRepo, broker, log, 50*time.Millisecond, 50)
	require.NoError(t, publisher.FlushOnce(ctx))

	payload, _ := json.Marshal(map[string]string{"delivery_id": resp.DeliveryID})
	require.NoError(t, dispatchSvc.HandleMessage(ctx, "", payload, nil))

	d, err := deliveryRepo.FindByID(ctx, resp.DeliveryID)
	require.NoError(t, err)
	require.Equal(t, notification.StatusDLQ, d.Status)
}
