package metrics

import "github.com/prometheus/client_golang/prometheus"

var (
	ConsumerLag = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "consumer_lag",
			Help: "Difference between the latest Kafka offset and the committed consumer offset.",
		},
		[]string{"topic", "partition"},
	)

	EventsProcessedTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "events_processed_total",
			Help: "Total number of warehouse events processed successfully.",
		},
		[]string{"event_type"},
	)

	EventProcessingDurationSeconds = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "event_processing_duration_seconds",
			Help:    "Warehouse event processing duration in seconds.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"event_type"},
	)

	CassandraWriteErrorsTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "cassandra_write_errors_total",
			Help: "Total number of Cassandra write/read errors while processing warehouse events.",
		},
	)
)

func init() {
	prometheus.MustRegister(
		ConsumerLag,
		EventsProcessedTotal,
		EventProcessingDurationSeconds,
		CassandraWriteErrorsTotal,
	)
}
