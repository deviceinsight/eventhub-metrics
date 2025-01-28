package metrics

import (
	"context"
	"log/slog"
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
}

func NewOtlpService(baseURL string, protocol string) RecordService {
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
		slog.Error("unsupported protocol", "protocol", protocol)
		return nil
	}

	if err != nil {
		slog.Error("failed to create exporter", "error", err)
		return nil
	}

	metrics := metricdata.ResourceMetrics{}
	metrics.ScopeMetrics = make([]metricdata.ScopeMetrics, 1)
	metrics.ScopeMetrics[0] =
		metricdata.ScopeMetrics{
			Scope:   instrumentation.Scope{},
			Metrics: make([]metricdata.Metrics, 0),
		}

	return &OtlpService{baseURL: baseURL, exporter: exporter, metrics: metrics}
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

	s.metrics.ScopeMetrics[0].Metrics = append(s.metrics.ScopeMetrics[0].Metrics, otelMetric)
}

func (s *OtlpService) PushMetrics() error {
	err := s.exporter.Export(context.Background(), &s.metrics)
	// clear the metrics after exporting
	s.metrics.ScopeMetrics[0].Metrics = make([]metricdata.Metrics, 0)

	return err
}
