package telemetry

import (
	"github.com/prometheus/client_golang/prometheus"
)

// Metrics holds Prometheus business and operational metrics.
type Metrics struct {
	Registry *prometheus.Registry

	NotificationsReceived *prometheus.CounterVec
	NotificationsSent     *prometheus.CounterVec
	NotificationsFailed   *prometheus.CounterVec
	NotificationsRetried  prometheus.Counter
	NotificationsDLQ      *prometheus.CounterVec
	QueueDepth            prometheus.Gauge
	QueueLag              prometheus.Gauge
	WorkerActiveJobs      prometheus.Gauge
	WorkerFailures        prometheus.Counter
	APIRequestDuration    *prometheus.HistogramVec
	APIRequestsTotal      *prometheus.CounterVec
	BrokerPublishTotal    *prometheus.CounterVec
	BrokerPublishDuration *prometheus.HistogramVec
	BrokerConsumeTotal    *prometheus.CounterVec
	BrokerConsumeDuration *prometheus.HistogramVec
	BrokerPublishFailed   prometheus.Counter
	BrokerConsumeFailed   prometheus.Counter
	DBQueryDuration       *prometheus.HistogramVec
	DBPoolAcquired        prometheus.Gauge
	DBPoolIdle            prometheus.Gauge
	DBPoolMax             prometheus.Gauge
	OutboxPublishDuration prometheus.Histogram
	DispatchDuration      *prometheus.HistogramVec
}

func newMetrics(reg *prometheus.Registry) *Metrics {
	m := &Metrics{Registry: reg}

	m.NotificationsReceived = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "notifications_received_total",
		Help: "Total notifications accepted via ingest.",
	}, []string{"channel", "duplicate"})
	m.NotificationsSent = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "notifications_sent_total",
		Help: "Total notifications successfully dispatched to providers.",
	}, []string{"channel"})
	m.NotificationsFailed = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "notifications_failed_total",
		Help: "Total notification dispatch failures.",
	}, []string{"channel", "permanent"})
	m.NotificationsRetried = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "notifications_retried_total",
		Help: "Total notification retry attempts.",
	})
	m.NotificationsDLQ = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "notifications_dlq_total",
		Help: "Total notifications moved to DLQ.",
	}, []string{"channel"})
	m.QueueDepth = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "queue_depth",
		Help: "Pending outbox messages awaiting publish.",
	})
	m.QueueLag = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "queue_lag",
		Help: "Approximate JetStream consumer pending messages.",
	})
	m.WorkerActiveJobs = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "worker_active_jobs",
		Help: "Deliveries currently being processed by workers.",
	})
	m.WorkerFailures = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "worker_failures_total",
		Help: "Total worker processing failures.",
	})
	m.APIRequestDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "api_request_duration_seconds",
		Help:    "HTTP request duration in seconds.",
		Buckets: prometheus.DefBuckets,
	}, []string{"method", "route", "status"})
	m.APIRequestsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "api_requests_total",
		Help: "Total HTTP requests.",
	}, []string{"method", "route", "status"})
	m.BrokerPublishTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "broker_publish_total",
		Help: "Total broker publish operations.",
	}, []string{"subject", "result"})
	m.BrokerPublishDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "broker_publish_duration_seconds",
		Help:    "Broker publish latency.",
		Buckets: prometheus.DefBuckets,
	}, []string{"subject"})
	m.BrokerConsumeTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "broker_consume_total",
		Help: "Total broker messages consumed.",
	}, []string{"subject", "result"})
	m.BrokerConsumeDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "broker_consume_duration_seconds",
		Help:    "Broker consume handler latency.",
		Buckets: prometheus.DefBuckets,
	}, []string{"subject"})
	m.BrokerPublishFailed = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "broker_publish_failed_total",
		Help: "Failed broker publish operations.",
	})
	m.BrokerConsumeFailed = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "broker_consume_failed_total",
		Help: "Failed broker consume handlers.",
	})
	m.DBQueryDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "db_query_duration_seconds",
		Help:    "Database query duration.",
		Buckets: prometheus.DefBuckets,
	}, []string{"operation"})
	m.DBPoolAcquired = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "db_pool_acquired_connections",
		Help: "Acquired database pool connections.",
	})
	m.DBPoolIdle = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "db_pool_idle_connections",
		Help: "Idle database pool connections.",
	})
	m.DBPoolMax = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "db_pool_max_connections",
		Help: "Max database pool connections.",
	})
	m.OutboxPublishDuration = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:    "outbox_publish_duration_seconds",
		Help:    "Outbox flush publish duration.",
		Buckets: prometheus.DefBuckets,
	})
	m.DispatchDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "dispatch_duration_seconds",
		Help:    "Dispatch operation duration.",
		Buckets: prometheus.DefBuckets,
	}, []string{"channel", "result"})

	collectors := []prometheus.Collector{
		m.NotificationsReceived, m.NotificationsSent, m.NotificationsFailed,
		m.NotificationsRetried, m.NotificationsDLQ,
		m.QueueDepth, m.QueueLag, m.WorkerActiveJobs, m.WorkerFailures,
		m.APIRequestDuration, m.APIRequestsTotal,
		m.BrokerPublishTotal, m.BrokerPublishDuration,
		m.BrokerConsumeTotal, m.BrokerConsumeDuration,
		m.BrokerPublishFailed, m.BrokerConsumeFailed,
		m.DBQueryDuration, m.DBPoolAcquired, m.DBPoolIdle, m.DBPoolMax,
		m.OutboxPublishDuration, m.DispatchDuration,
	}
	for _, c := range collectors {
		reg.MustRegister(c)
	}
	return m
}
