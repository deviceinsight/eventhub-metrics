package metrics

import (
	"log/slog"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/push"
)

type pushGatewayService struct {
	baseURL  string
	registry *prometheus.Registry
	gauges   map[*Metric]*prometheus.GaugeVec
}

func NewPushGatewayService(baseURL string) RecordService {

	slog.Debug("using PushGateway exporter")

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

	return &pushGatewayService{baseURL: baseURL, registry: registry, gauges: gauges}
}

func (s *pushGatewayService) RecordMetric(metric *Metric, labels map[string]string, value float64) {
	s.gauges[metric].With(labels).Set(value)
}

func (s *pushGatewayService) PushMetrics() error {
	return push.New(s.baseURL, "eventhub-metrics").Gatherer(s.registry).Push()
}
