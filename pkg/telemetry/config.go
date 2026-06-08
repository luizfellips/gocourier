package telemetry

import (
	"os"
	"strings"
)

// Config holds OpenTelemetry and metrics settings.
type Config struct {
	ServiceName       string
	OTLPEndpoint      string
	MetricsAddr       string
	TracesSampler     string
	EnableOTLP        bool
}

func ConfigFromEnv(serviceName, defaultMetricsAddr string) Config {
	endpoint := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
	if name := os.Getenv("OTEL_SERVICE_NAME"); name != "" {
		serviceName = name
	}
	metricsAddr := os.Getenv("METRICS_ADDR")
	if metricsAddr == "" {
		metricsAddr = defaultMetricsAddr
	}
	sampler := os.Getenv("OTEL_TRACES_SAMPLER")
	if sampler == "" {
		sampler = "parentbased_always_on"
	}
	enable := endpoint != "" && strings.ToLower(os.Getenv("OTEL_SDK_DISABLED")) != "true"
	return Config{
		ServiceName:   serviceName,
		OTLPEndpoint:  endpoint,
		MetricsAddr:   metricsAddr,
		TracesSampler: sampler,
		EnableOTLP:    enable,
	}
}
