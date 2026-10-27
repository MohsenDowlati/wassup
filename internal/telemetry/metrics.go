package telemetry

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

type Metrics struct {
	ActiveConnections prometheus.Gauge

	MessagesReceived *prometheus.CounterVec

	MessagesRouted *prometheus.CounterVec

	DeliveryDuration *prometheus.HistogramVec

	OfflineMessagesPersisted prometheus.Counter

	ReplayBatchDuration prometheus.Histogram

	RateLimitDrops *prometheus.CounterVec

	SlowClientDisconnects prometheus.Counter
}

func NewMetrics(namespace string) *Metrics {
	if namespace == "" {
		namespace = "wassup"
	}

	return &Metrics{
		ActiveConnections: promauto.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: "gateway",
			Name:      "active_connections",
			Help:      "Current count of active WebSocket sessions connected to this node.",
		}),

		MessagesReceived: promauto.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: "gateway",
			Name:      "messages_received_total",
			Help:      "Total inbound WebSocket frames received by message type.",
		}, []string{"type"}),

		MessagesRouted: promauto.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: "dispatcher",
			Name:      "messages_routed_total",
			Help:      "Total messages processed by routing target (local, remote_redis, offline_mongo).",
		}, []string{"target", "status"}),

		DeliveryDuration: promauto.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: namespace,
			Subsystem: "dispatcher",
			Name:      "delivery_duration_seconds",
			Help:      "End-to-end delivery latency from gateway ingress to delivery/offline storage.",
			Buckets:   []float64{0.005, 0.010, 0.025, 0.050, 0.100, 0.250, 0.500, 1.000, 2.500},
		}, []string{"target"}),

		OfflineMessagesPersisted: promauto.NewCounter(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: "storage",
			Name:      "offline_messages_persisted_total",
			Help:      "Total messages saved into MongoDB ephemeral store.",
		}),

		ReplayBatchDuration: promauto.NewHistogram(prometheus.HistogramOpts{
			Namespace: namespace,
			Subsystem: "replay",
			Name:      "batch_duration_seconds",
			Help:      "Time taken to claim and push replayed messages to reconnected client.",
			Buckets:   []float64{0.010, 0.050, 0.100, 0.250, 0.500, 1.000},
		}),

		RateLimitDrops: promauto.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: "security",
			Name:      "ratelimit_drops_total",
			Help:      "Total requests rejected by distributed rate limiting.",
		}, []string{"scope"}),

		SlowClientDisconnects: promauto.NewCounter(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: "gateway",
			Name:      "slow_client_disconnects_total",
			Help:      "Total WebSocket clients evicted due to outbox buffer saturation.",
		}),
	}
}
