package metrics

import (
	"log/slog"
	"net/http"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// PrometheusService exposes recorded metrics via an HTTP handler that is
// mounted onto the shared HTTP server.
type PrometheusService interface {
	RecordService
	MetricsHandler() http.Handler
}

// gaugeSet is one complete registry snapshot; double-buffered so scrapers never observe a half-collected cycle.
type gaugeSet struct {
	registry *prometheus.Registry
	gauges   map[*Metric]*prometheus.GaugeVec
}

func newGaugeSet() *gaugeSet {
	registry := prometheus.NewRegistry()

	gauges := make(map[*Metric]*prometheus.GaugeVec)
	for _, metric := range allMetrics {
		gauges[metric] = prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: metricPrefix,
			Name:      metric.Name,
			Help:      metric.Help,
		}, metric.Labels)
		registry.MustRegister(gauges[metric])
	}

	// add default metrics
	registry.MustRegister(collectors.NewGoCollector())

	return &gaugeSet{registry: registry, gauges: gauges}
}

type prometheusService struct {
	mu       sync.RWMutex
	active   *gaugeSet
	building *gaugeSet
}

func NewPrometheusService() PrometheusService {

	slog.Debug("using prometheus exporter")

	set := newGaugeSet()

	return &prometheusService{active: set, building: set}
}

func (s *prometheusService) MetricsHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.mu.RLock()
		registry := s.active.registry
		s.mu.RUnlock()
		promhttp.HandlerFor(registry, promhttp.HandlerOpts{}).ServeHTTP(w, r)
	})
}

func (s *prometheusService) RecordMetric(metric *Metric, labels map[string]string, value float64) {
	s.mu.RLock()
	building := s.building
	s.mu.RUnlock()
	building.gauges[metric].With(labels).Set(value)
}

// StartCycle begins a fresh building set so stale series don't accumulate.
func (s *prometheusService) StartCycle() {
	set := newGaugeSet()
	s.mu.Lock()
	s.building = set
	s.mu.Unlock()
}

// PushMetrics publishes the collected cycle by atomically promoting building to active.
func (s *prometheusService) PushMetrics() error {
	s.mu.Lock()
	s.active = s.building
	s.mu.Unlock()
	return nil
}
