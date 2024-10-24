package metrics

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type RecordServiceWithHTTPServer interface {
	RunHTTPServer(address string, readTimeout time.Duration)
	RecordService
}

type prometheusService struct {
	registry *prometheus.Registry
	gauges   map[*Metric]*prometheus.GaugeVec
}

func NewPrometheusService() RecordServiceWithHTTPServer {

	slog.Debug("using prometheus exporter")

	registry := prometheus.NewRegistry()

	var gauges = make(map[*Metric]*prometheus.GaugeVec)

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

	return &prometheusService{registry: registry, gauges: gauges}
}

func (s *prometheusService) RunHTTPServer(address string, readTimeout time.Duration) {

	pMux := http.NewServeMux()
	promHandler := promhttp.HandlerFor(s.registry, promhttp.HandlerOpts{})
	pMux.Handle("/metrics", promHandler)

	pMux.Handle("/health", http.HandlerFunc(healthHandler))

	server := &http.Server{
		Addr:        address,
		ReadTimeout: readTimeout,
		Handler:     pMux,
	}

	slog.Info("http server started", "address", server.Addr)
	if err := server.ListenAndServe(); err != nil {
		slog.Error("http server stopped", "error", err)
		os.Exit(1)
	}
}

func healthHandler(w http.ResponseWriter, _ *http.Request) {
	_, _ = fmt.Fprintf(w, "OK\n")
}

func (s *prometheusService) RecordMetric(metric *Metric, labels map[string]string, value float64) {
	s.gauges[metric].With(labels).Set(value)
}

func (s *prometheusService) PushMetrics() error {
	return nil
}
