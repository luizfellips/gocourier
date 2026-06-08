package http

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/gocourier/internal/application/ingest"
	"github.com/gocourier/internal/application/replay"
	"github.com/gocourier/internal/domain/notification"
	"github.com/gocourier/internal/ports"
	"github.com/gocourier/pkg/apperrors"
	"github.com/gocourier/pkg/telemetry"
)

type Server struct {
	ingest      *ingest.Service
	replay      *replay.Service
	dashboard   ports.DashboardReader
	apiKeys     map[string]struct{}
	log         *slog.Logger
	serviceName string
	mux         *http.ServeMux
}

func NewServer(
	ingestSvc *ingest.Service,
	replaySvc *replay.Service,
	dashboard ports.DashboardReader,
	apiKeys []string,
	log *slog.Logger,
	serviceName string,
) *Server {
	keys := make(map[string]struct{}, len(apiKeys))
	for _, k := range apiKeys {
		keys[k] = struct{}{}
	}
	if serviceName == "" {
		serviceName = "api"
	}
	s := &Server{
		ingest:      ingestSvc,
		replay:      replaySvc,
		dashboard:   dashboard,
		apiKeys:     keys,
		log:         log,
		serviceName: serviceName,
		mux:         http.NewServeMux(),
	}
	s.routes()
	return s
}

func (s *Server) routes() {
	s.mux.Handle("GET /metrics", telemetry.MetricsHandler())
	s.mux.HandleFunc("GET /health", s.handleHealth)
	s.mux.HandleFunc("GET /v1/dashboard/summary", s.handleDashboardSummary)
	s.mux.HandleFunc("GET /v1/dashboard/deliveries/{id}", s.handleDashboardDetail)
	s.mux.HandleFunc("POST /v1/notifications", s.auth(s.handleIngest))
	s.mux.HandleFunc("POST /v1/notifications/{id}/replay", s.auth(s.handleReplay))
}

func (s *Server) Handler() http.Handler {
	return withMetrics(withCORS(s.mux))
}

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

func withMetrics(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		sw := &statusWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(sw, r)
		route := r.Pattern
		if route == "" {
			route = r.URL.Path
		}
		status := http.StatusText(sw.status)
		if sw.status >= 100 && sw.status < 600 {
			status = fmt.Sprintf("%d", sw.status)
		}
		telemetry.RecordAPIRequest(r.Method, route, status, time.Since(start))
	})
}

func (s *Server) auth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		key := r.Header.Get("X-API-Key")
		if key == "" {
			key = strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		}
		if _, ok := s.apiKeys[key]; !ok {
			writeError(w, http.StatusUnauthorized, apperrors.ErrUnauthorized)
			return
		}
		next(w, r)
	}
}

func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"status": "ok",
		"time":   time.Now().UTC().Format(time.RFC3339),
	})
}

func (s *Server) handleIngest(w http.ResponseWriter, r *http.Request) {
	var req notification.IngestRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	resp, err := s.ingest.Ingest(r.Context(), req)
	if err != nil {
		status := http.StatusInternalServerError
		switch {
		case errors.Is(err, apperrors.ErrValidation):
			status = http.StatusBadRequest
		case errors.Is(err, apperrors.ErrUnauthorized):
			status = http.StatusUnauthorized
		}
		writeError(w, status, err)
		return
	}

	status := http.StatusAccepted
	if resp.Duplicate {
		status = http.StatusOK
	}
	writeJSON(w, status, resp)
}

func (s *Server) handleReplay(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, apperrors.ErrValidation)
		return
	}
	if err := s.replay.Replay(r.Context(), id); err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, apperrors.ErrNotFound) {
			status = http.StatusNotFound
		}
		writeError(w, status, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"delivery_id": id, "status": "queued"})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, err error) {
	writeJSON(w, status, map[string]string{"error": err.Error()})
}

func (s *Server) ListenAndServe(ctx context.Context, addr string) error {
	handler := telemetry.WrapHTTP(s.serviceName, s.Handler())
	srv := &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
	}
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdownCtx)
	}()
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}
