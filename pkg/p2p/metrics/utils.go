package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

const (
	namespace = "ddc"
	subsystem = "p2p"
)

// NewCounter creates a Counter metrics under the global namespace returns nop if metrics are disabled.
func NewCounter(name, help string, labels []string) *prometheus.CounterVec {
	return promauto.NewCounterVec(prometheus.CounterOpts{Namespace: namespace, Subsystem: subsystem, Name: name, Help: help}, labels)
}

// NewGauge creates a Gauge metrics under the global namespace returns nop if metrics are disabled.
func NewGauge(name, help string, labels []string) *prometheus.GaugeVec {
	return promauto.NewGaugeVec(prometheus.GaugeOpts{Namespace: namespace, Subsystem: subsystem, Name: name, Help: help}, labels)
}

// NewHistogram creates a Histogram metrics under the global namespace returns nop if metrics are disabled.
func NewHistogram(name, help string, labels []string) *prometheus.HistogramVec {
	return promauto.NewHistogramVec(prometheus.HistogramOpts{Namespace: namespace, Subsystem: subsystem, Name: name, Help: help}, labels)
}

// NewHistogramWithBuckets creates a Histogram metrics with custom buckets.
func NewHistogramWithBuckets(name, subsystem, help string, labels []string, buckets []float64) *prometheus.HistogramVec {
	return promauto.NewHistogramVec(prometheus.HistogramOpts{Namespace: namespace, Subsystem: subsystem, Name: name, Help: help, Buckets: buckets}, labels)
}
