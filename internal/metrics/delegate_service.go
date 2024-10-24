package metrics

import "log/slog"

type delegateService struct {
	delegates []RecordService
}

func NewDelegateService(delegates ...RecordService) Service {

	if len(delegates) == 0 {
		slog.Warn("no metric exporters configured. only logging gauges")
		delegates = append(delegates, newLogService())
	}

	return &service{recorder: &delegateService{delegates: delegates}}
}

func (s *delegateService) RecordMetric(metric *Metric, labels map[string]string, value float64) {
	for _, delegate := range s.delegates {
		delegate.RecordMetric(metric, labels, value)
	}
}

func (s *delegateService) PushMetrics() error {
	for _, delegate := range s.delegates {
		if err := delegate.PushMetrics(); err != nil {
			return err
		}
	}
	return nil
}
