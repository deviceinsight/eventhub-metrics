package metrics

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/sdk/instrumentation"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

type OtlpService struct {
	baseURL  string
	exporter metric.Exporter
	metrics  metricdata.ResourceMetrics
	mu       sync.Mutex
}

func NewOtlpService(baseURL string, protocol string) (RecordService, error) {
	slog.Debug("using otlp exporter", "baseURL", baseURL, "protocol", protocol)

	var exporter metric.Exporter
	var err error

	switch protocol {
	case "grpc":
		exporter, err = otlpmetricgrpc.New(context.Background(),
			otlpmetricgrpc.WithEndpointURL(baseURL),
		)
	case "http":
		exporter, err = otlpmetrichttp.New(context.Background(),
			otlpmetrichttp.WithEndpointURL(baseURL),
		)
	default:
		return nil, fmt.Errorf("unsupported otlp protocol: %q", protocol)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create otlp exporter: %w", err)
	}

	metrics := metricdata.ResourceMetrics{}
	metrics.ScopeMetrics = make([]metricdata.ScopeMetrics, 1)
	metrics.ScopeMetrics[0] =
		metricdata.ScopeMetrics{
			Scope:   instrumentation.Scope{},
			Metrics: make([]metricdata.Metrics, 0),
		}

	return &OtlpService{baseURL: baseURL, exporter: exporter, metrics: metrics}, nil
}

func (s *OtlpService) RecordMetric(metric *Metric, labels map[string]string, value float64) {
	attributes := make([]attribute.KeyValue, 0, len(labels))
	for k, v := range labels {
		attributes = append(attributes, attribute.String(k, v))
	}

	datapoint := metricdata.DataPoint[float64]{
		Value:      value,
		Time:       time.Now(),
		Attributes: attribute.NewSet(attributes...),
		Exemplars:  make([]metricdata.Exemplar[float64], 0),
	}

	gauge := metricdata.Gauge[float64]{
		DataPoints: []metricdata.DataPoint[float64]{datapoint},
	}

	otelMetric := metricdata.Metrics{
		Name:        metricPrefix + "_" + metric.Name,
		Description: metric.Help,
		Data:        gauge,
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.metrics.ScopeMetrics[0].Metrics = append(s.metrics.ScopeMetrics[0].Metrics, otelMetric)
}

func (s *OtlpService) StartCycle() {
	// drop the previous cycle's accumulated metrics before collecting fresh ones
	s.mu.Lock()
	defer s.mu.Unlock()
	s.metrics.ScopeMetrics[0].Metrics = make([]metricdata.Metrics, 0)
}

func (s *OtlpService) PushMetrics() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.exporter.Export(context.Background(), &s.metrics)
}
