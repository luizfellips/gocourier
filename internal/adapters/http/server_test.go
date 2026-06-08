package http

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gocourier/internal/application/dispatch"
	"github.com/gocourier/internal/application/ingest"
	"github.com/gocourier/internal/application/replay"
	"github.com/gocourier/internal/domain/notification"
	"github.com/gocourier/internal/ports"
	"github.com/gocourier/pkg/apperrors"
	"github.com/gocourier/pkg/logger"
)

const testAPIKey = "test-key"

type httpFakeClock struct{ now time.Time }

func (f httpFakeClock) Now() time.Time { return f.now }

type httpDeliveryRepo struct {
	byKey map[string]*notification.Delivery
}

func newHTTPDeliveryRepo() *httpDeliveryRepo {
	return &httpDeliveryRepo{byKey: map[string]*notification.Delivery{}}
}

func (r *httpDeliveryRepo) Save(_ context.Context, d *notification.Delivery) error {
	r.byKey[d.IdempotencyKey] = d
	return nil
}
func (r *httpDeliveryRepo) Update(_ context.Context, d *notification.Delivery) error {
	r.byKey[d.IdempotencyKey] = d
	return nil
}
func (r *httpDeliveryRepo) UpdateIfStatus(_ context.Context, d *notification.Delivery, expected notification.Status) (bool, error) {
	cur, ok := r.byKey[d.IdempotencyKey]
	if !ok {
		return false, apperrors.ErrNotFound
	}
	if cur.Status != expected {
		return false, nil
	}
	r.byKey[d.IdempotencyKey] = d
	return true, nil
}
func (r *httpDeliveryRepo) FindByID(_ context.Context, id string) (*notification.Delivery, error) {
	for _, d := range r.byKey {
		if d.ID == id {
			cp := *d
			return &cp, nil
		}
	}
	return nil, apperrors.ErrNotFound
}
func (r *httpDeliveryRepo) FindByIdempotencyKey(_ context.Context, key string, _ notification.Channel) (*notification.Delivery, error) {
	d, ok := r.byKey[key]
	if !ok {
		return nil, apperrors.ErrNotFound
	}
	return d, nil
}
func (r *httpDeliveryRepo) RecordAttempt(_ context.Context, _ string, _ notification.Attempt) error {
	return nil
}

type httpStore struct {
	repo *httpDeliveryRepo
}

func (s *httpStore) IngestTransactional(_ context.Context, d *notification.Delivery, _ time.Time, _ string, _ []byte, _ map[string]string, _ bool) error {
	if _, ok := s.repo.byKey[d.IdempotencyKey]; ok {
		return apperrors.ErrDuplicate
	}
	s.repo.byKey[d.IdempotencyKey] = d
	return nil
}
func (s *httpStore) TryRegisterIdempotency(_ context.Context, _ string, _ notification.Channel, _ string, _ time.Time) (bool, error) {
	return true, nil
}

type httpAudit struct{}

func (httpAudit) Append(_ context.Context, _ string, _ string, _ map[string]any) error { return nil }

type httpBroker struct {
	published []string
}

func (b *httpBroker) Publish(_ context.Context, subject string, _ []byte, _ map[string]string) error {
	b.published = append(b.published, subject)
	return nil
}
func (b *httpBroker) Subscribe(_ context.Context, _, _ string, _ ports.MessageHandler) error { return nil }
func (b *httpBroker) EnsureStreams(_ context.Context) error                                   { return nil }
func (b *httpBroker) Close() error                                                            { return nil }

type httpProvider struct {
	channel notification.Channel
}

func (p *httpProvider) Channel() notification.Channel { return p.channel }
func (p *httpProvider) Send(_ context.Context, _ *notification.Delivery) (ports.ProviderResult, error) {
	return ports.ProviderResult{ProviderMessageID: "msg-1"}, nil
}

type fakeDashboard struct {
	summary *ports.DashboardSummary
	detail  *ports.DeliveryDetail
	err     error
}

func (f *fakeDashboard) Summary(_ context.Context, _ int) (*ports.DashboardSummary, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.summary, nil
}
func (f *fakeDashboard) Detail(_ context.Context, id string) (*ports.DeliveryDetail, error) {
	if f.err != nil {
		return nil, f.err
	}
	if f.detail == nil {
		return nil, apperrors.ErrNotFound
	}
	return f.detail, nil
}

type testServer struct {
	srv   *Server
	repo  *httpDeliveryRepo
	broker *httpBroker
}

func newTestServer(t *testing.T, dashboard ports.DashboardReader) *testServer {
	t.Helper()
	now := time.Now().UTC()
	repo := newHTTPDeliveryRepo()
	store := &httpStore{repo: repo}
	ingestSvc := ingest.NewService(store, repo, httpAudit{}, httpFakeClock{now: now}, "notifications", time.Hour)

	broker := &httpBroker{}
	provider := &httpProvider{channel: notification.ChannelEmail}
	dispatchSvc := dispatch.NewService(
		repo, &dispatchMemAudit{}, broker, []ports.ChannelProvider{provider},
		httpFakeClock{now: now}, logger.New("error"),
		dispatch.Config{
			MaxAttempts: 3, RetryBase: time.Millisecond, RetryMax: time.Second,
			StreamPrefix: "notifications", CBThreshold: 100, CBWindow: time.Second, CBCooldown: time.Second,
		},
	)
	replaySvc := replay.NewService(dispatchSvc)

	srv := NewServer(ingestSvc, replaySvc, dashboard, []string{testAPIKey}, logger.New("error"), "api")
	return &testServer{srv: srv, repo: repo, broker: broker}
}

type dispatchMemAudit struct{ events []string }

func (a *dispatchMemAudit) Append(_ context.Context, _ string, eventType string, _ map[string]any) error {
	a.events = append(a.events, eventType)
	return nil
}

func sampleIngestBody(key string) []byte {
	b, _ := json.Marshal(map[string]any{
		"schema_version":  notification.SchemaVersion,
		"idempotency_key": key,
		"channel":         "email",
		"recipient":       map[string]string{"address": "user@example.com"},
		"template":        map[string]string{"id": "welcome"},
	})
	return b
}

func TestHealth(t *testing.T) {
	ts := newTestServer(t, &fakeDashboard{summary: &ports.DashboardSummary{}})
	rec := httptest.NewRecorder()
	ts.srv.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/health", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status: %d", rec.Code)
	}
}

func TestMetricsEndpoint(t *testing.T) {
	ts := newTestServer(t, &fakeDashboard{summary: &ports.DashboardSummary{}})
	rec := httptest.NewRecorder()
	ts.srv.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status: %d", rec.Code)
	}
}

func TestCORSOptions(t *testing.T) {
	ts := newTestServer(t, &fakeDashboard{summary: &ports.DashboardSummary{}})
	rec := httptest.NewRecorder()
	ts.srv.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodOptions, "/v1/notifications", nil))
	if rec.Code != http.StatusNoContent {
		t.Fatalf("status: %d", rec.Code)
	}
	if rec.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Fatal("expected CORS header")
	}
}

func TestIngestUnauthorized(t *testing.T) {
	ts := newTestServer(t, &fakeDashboard{summary: &ports.DashboardSummary{}})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/notifications", bytes.NewReader(sampleIngestBody("k1")))
	req.Header.Set("Content-Type", "application/json")
	ts.srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status: %d body: %s", rec.Code, rec.Body.String())
	}
}

func TestIngestSuccess(t *testing.T) {
	ts := newTestServer(t, &fakeDashboard{summary: &ports.DashboardSummary{}})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/notifications", bytes.NewReader(sampleIngestBody("new-key")))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", testAPIKey)
	ts.srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusAccepted {
		t.Fatalf("status: %d body: %s", rec.Code, rec.Body.String())
	}
}

func TestIngestDuplicate(t *testing.T) {
	ts := newTestServer(t, &fakeDashboard{summary: &ports.DashboardSummary{}})
	body := sampleIngestBody("dup-key")
	for _, want := range []int{http.StatusAccepted, http.StatusOK} {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/v1/notifications", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-API-Key", testAPIKey)
		ts.srv.Handler().ServeHTTP(rec, req)
		if rec.Code != want {
			t.Fatalf("expected %d got %d body: %s", want, rec.Code, rec.Body.String())
		}
	}
}

func TestIngestMalformedJSON(t *testing.T) {
	ts := newTestServer(t, &fakeDashboard{summary: &ports.DashboardSummary{}})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/notifications", bytes.NewBufferString("{bad"))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", testAPIKey)
	ts.srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status: %d", rec.Code)
	}
}

func TestReplaySuccess(t *testing.T) {
	ts := newTestServer(t, &fakeDashboard{summary: &ports.DashboardSummary{}})
	now := time.Now().UTC()
	d := &notification.Delivery{
		ID:             "dlq-1",
		IdempotencyKey: "dlq-key",
		Channel:        notification.ChannelEmail,
		Priority:       notification.PriorityNormal,
		Recipient:      json.RawMessage(`{"address":"a@b.com"}`),
		Template:       json.RawMessage(`{"id":"t"}`),
		Status:         notification.StatusDLQ,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	ts.repo.byKey["dlq-key"] = d

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/notifications/dlq-1/replay", nil)
	req.Header.Set("X-API-Key", testAPIKey)
	ts.srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status: %d body: %s", rec.Code, rec.Body.String())
	}
	if len(ts.broker.published) == 0 {
		t.Fatal("expected broker publish")
	}
}

func TestReplayNotFound(t *testing.T) {
	ts := newTestServer(t, &fakeDashboard{summary: &ports.DashboardSummary{}})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/notifications/missing/replay", nil)
	req.Header.Set("X-API-Key", testAPIKey)
	ts.srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status: %d", rec.Code)
	}
}

func TestDashboardSummary(t *testing.T) {
	dash := &fakeDashboard{summary: &ports.DashboardSummary{StatusCounts: map[string]int{"queued": 1}}}
	ts := newTestServer(t, dash)
	rec := httptest.NewRecorder()
	ts.srv.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/v1/dashboard/summary?limit=10", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status: %d", rec.Code)
	}
	body, _ := io.ReadAll(rec.Body)
	if !bytes.Contains(body, []byte("queued")) {
		t.Fatalf("unexpected body: %s", body)
	}
}

func TestDashboardDetailFound(t *testing.T) {
	dash := &fakeDashboard{
		detail: &ports.DeliveryDetail{
			Delivery: ports.DeliveryRow{ID: "d1", Status: string(notification.StatusQueued)},
		},
	}
	ts := newTestServer(t, dash)
	rec := httptest.NewRecorder()
	ts.srv.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/v1/dashboard/deliveries/d1", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status: %d", rec.Code)
	}
}

func TestDashboardDetailNotFound(t *testing.T) {
	ts := newTestServer(t, &fakeDashboard{})
	rec := httptest.NewRecorder()
	ts.srv.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/v1/dashboard/deliveries/missing", nil))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status: %d", rec.Code)
	}
}

func TestDashboardSummaryError(t *testing.T) {
	ts := newTestServer(t, &fakeDashboard{err: fmt.Errorf("db down")})
	rec := httptest.NewRecorder()
	ts.srv.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/v1/dashboard/summary", nil))
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status: %d", rec.Code)
	}
}

func TestAuthBearerToken(t *testing.T) {
	ts := newTestServer(t, &fakeDashboard{summary: &ports.DashboardSummary{}})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/notifications", bytes.NewReader(sampleIngestBody("bearer-key")))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", testAPIKey))
	ts.srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusAccepted {
		t.Fatalf("status: %d", rec.Code)
	}
}
