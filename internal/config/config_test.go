package config

import (
	"testing"
	"time"
)

func TestLoadDefaults(t *testing.T) {
	t.Setenv("API_KEYS", "dev-key")
	t.Setenv("SERVICE_NAME", "")
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.ServiceName != "gocourier" {
		t.Fatalf("service name: %s", cfg.ServiceName)
	}
	if cfg.HTTPAddr != ":8080" {
		t.Fatalf("http addr: %s", cfg.HTTPAddr)
	}
	if cfg.OutboxPollInterval != 500*time.Millisecond {
		t.Fatalf("outbox interval: %s", cfg.OutboxPollInterval)
	}
	if cfg.MetricsAddr != ":9090" {
		t.Fatalf("metrics addr: %s", cfg.MetricsAddr)
	}
}

func TestLoadWorkerMetricsDefault(t *testing.T) {
	t.Setenv("API_KEYS", "dev-key")
	t.Setenv("SERVICE_NAME", "worker")
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.MetricsAddr != ":9091" {
		t.Fatalf("metrics addr: %s", cfg.MetricsAddr)
	}
}

func TestLoadSchedulerMetricsDefault(t *testing.T) {
	t.Setenv("API_KEYS", "dev-key")
	t.Setenv("SERVICE_NAME", "scheduler")
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.MetricsAddr != ":9092" {
		t.Fatalf("metrics addr: %s", cfg.MetricsAddr)
	}
}

func TestLoadEmptyAPIKeys(t *testing.T) {
	t.Setenv("API_KEYS", " , ")
	_, err := Load()
	if err == nil {
		t.Fatal("expected error for empty API keys")
	}
}

func TestLoadInvalidIntUsesFallback(t *testing.T) {
	t.Setenv("API_KEYS", "dev-key")
	t.Setenv("WORKER_CONCURRENCY", "not-a-number")
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.WorkerConcurrency != 10 {
		t.Fatalf("worker concurrency: %d", cfg.WorkerConcurrency)
	}
}

func TestLoadInvalidDurationUsesFallback(t *testing.T) {
	t.Setenv("API_KEYS", "dev-key")
	t.Setenv("OUTBOX_POLL_INTERVAL", "bad")
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.OutboxPollInterval != 500*time.Millisecond {
		t.Fatalf("outbox interval: %s", cfg.OutboxPollInterval)
	}
}

func TestLoadCustomEnv(t *testing.T) {
	t.Setenv("API_KEYS", "k1,k2")
	t.Setenv("HTTP_ADDR", ":9099")
	t.Setenv("NATS_MAX_DELIVER", "5")
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.APIKeys) != 2 {
		t.Fatalf("api keys: %v", cfg.APIKeys)
	}
	if cfg.HTTPAddr != ":9099" {
		t.Fatalf("http addr: %s", cfg.HTTPAddr)
	}
	if cfg.NATSMaxDeliver != 5 {
		t.Fatalf("max deliver: %d", cfg.NATSMaxDeliver)
	}
}
