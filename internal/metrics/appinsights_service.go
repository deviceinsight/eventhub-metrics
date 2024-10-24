package metrics

import (
	"fmt"
	"log/slog"

	"github.com/microsoft/ApplicationInsights-Go/appinsights"
)

type appInsightsService struct {
	client appinsights.TelemetryClient
}

func NewAppInsightsService(instrumentationKey string) RecordService {

	slog.Debug("using appInsights exporter")

	return &appInsightsService{
		client: appinsights.NewTelemetryClient(instrumentationKey),
	}
}

func (s *appInsightsService) RecordMetric(metric *Metric, labels map[string]string, value float64) {

	metricTelemetry := appinsights.NewMetricTelemetry(fmt.Sprintf("%s_%s", metricPrefix, metric.Name), value)

	for key, value := range labels {
		metricTelemetry.Properties[key] = value
	}

	s.client.Track(metricTelemetry)
}

func (s *appInsightsService) PushMetrics() error {
	return nil
}
