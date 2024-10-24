package metrics

import "log/slog"

type logService struct {
}

func newLogService() RecordService {
	return &logService{}
}

func (s *logService) RecordMetric(metric *Metric, labels map[string]string, value float64) {
	slog.Info("recording metric", "metric", metric.Name, "labels", labels, "value", value)
}

func (s *logService) PushMetrics() error {
	return nil
}
