package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	ServiceName string
	LogLevel    string

	HTTPAddr string
	APIKeys  []string

	DatabaseURL string

	NATSURL           string
	NATSStreamPrefix  string
	NATSMaxDeliver    int
	NATSAckWait       time.Duration

	OutboxPollInterval time.Duration
	OutboxBatchSize    int

	SchedulerPollInterval time.Duration

	WorkerConcurrency int
	WorkerChannels    []string

	MaxRetryAttempts int
	RetryBaseDelay   time.Duration
	RetryMaxDelay    time.Duration

	IdempotencyTTL time.Duration

	CircuitBreakerThreshold int
	CircuitBreakerWindow    time.Duration
	CircuitBreakerCooldown  time.Duration

	OTLPEndpoint  string
	MetricsAddr   string
	TracesSampler string
	EnableOTLP    bool
}

func Load() (*Config, error) {
	cfg := &Config{
		ServiceName: getEnv("SERVICE_NAME", "gocourier"),
		LogLevel:    getEnv("LOG_LEVEL", "info"),
		HTTPAddr:    getEnv("HTTP_ADDR", ":8080"),

		DatabaseURL: getEnv("DATABASE_URL", "postgres://gocourier:gocourier@localhost:5432/gocourier?sslmode=disable"),
		NATSURL:     getEnv("NATS_URL", "nats://localhost:4222"),

		NATSStreamPrefix: getEnv("NATS_STREAM_PREFIX", "notifications"),
		NATSMaxDeliver:   getEnvInt("NATS_MAX_DELIVER", 8),
		NATSAckWait:      getEnvDuration("NATS_ACK_WAIT", 30*time.Second),

		OutboxPollInterval: getEnvDuration("OUTBOX_POLL_INTERVAL", 500*time.Millisecond),
		OutboxBatchSize:      getEnvInt("OUTBOX_BATCH_SIZE", 50),

		SchedulerPollInterval: getEnvDuration("SCHEDULER_POLL_INTERVAL", 5*time.Second),

		WorkerConcurrency: getEnvInt("WORKER_CONCURRENCY", 10),
		WorkerChannels:    splitCSV(getEnv("WORKER_CHANNELS", "email,sms,push,webhook")),

		MaxRetryAttempts: getEnvInt("MAX_RETRY_ATTEMPTS", 8),
		RetryBaseDelay:   getEnvDuration("RETRY_BASE_DELAY", time.Second),
		RetryMaxDelay:    getEnvDuration("RETRY_MAX_DELAY", 30*time.Minute),

		IdempotencyTTL: getEnvDuration("IDEMPOTENCY_TTL", 24*time.Hour),

		CircuitBreakerThreshold: getEnvInt("CIRCUIT_BREAKER_THRESHOLD", 5),
		CircuitBreakerWindow:    getEnvDuration("CIRCUIT_BREAKER_WINDOW", 60*time.Second),
		CircuitBreakerCooldown:  getEnvDuration("CIRCUIT_BREAKER_COOLDOWN", 30*time.Second),
	}

	cfg.OTLPEndpoint = getEnv("OTEL_EXPORTER_OTLP_ENDPOINT", "")
	cfg.TracesSampler = getEnv("OTEL_TRACES_SAMPLER", "parentbased_always_on")
	cfg.EnableOTLP = cfg.OTLPEndpoint != "" && strings.ToLower(getEnv("OTEL_SDK_DISABLED", "")) != "true"

	defaultMetrics := ":9090"
	switch cfg.ServiceName {
	case "worker":
		defaultMetrics = ":9091"
	case "scheduler":
		defaultMetrics = ":9092"
	}
	cfg.MetricsAddr = getEnv("METRICS_ADDR", defaultMetrics)

	keys := getEnv("API_KEYS", "dev-api-key")
	cfg.APIKeys = splitCSV(keys)
	if len(cfg.APIKeys) == 0 {
		return nil, fmt.Errorf("API_KEYS must contain at least one key")
	}

	return cfg, nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return n
}

func getEnvDuration(key string, fallback time.Duration) time.Duration {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return fallback
	}
	return d
}

func splitCSV(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
