package metrics

import (
	"strings"
	"sync"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"go.opentelemetry.io/otel/sdk/instrumentation"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

// must be safe under the collector's concurrent goroutines (run with -race).
func TestOtlpRecordMetricConcurrent(t *testing.T) {
	s := &OtlpService{metrics: metricdata.ResourceMetrics{
		ScopeMetrics: []metricdata.ScopeMetrics{{
			Scope:   instrumentation.Scope{},
			Metrics: make([]metricdata.Metrics, 0),
		}},
	}}

	const goroutines, perGoroutine = 20, 50
	var wg sync.WaitGroup
	for range goroutines {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range perGoroutine {
				s.RecordMetric(ConsumerGroupLag, map[string]string{
					labelNamespace:     "ns",
					labelEventhub:      "eh",
					labelConsumerGroup: "cg",
				}, 1.0)
			}
		}()
	}
	wg.Wait()

	if got := len(s.metrics.ScopeMetrics[0].Metrics); got != goroutines*perGoroutine {
		t.Fatalf("expected %d recorded metrics, got %d", goroutines*perGoroutine, got)
	}
}

func TestPrometheusResetMetricsDropsStaleSeries(t *testing.T) {
	s := newTestPrometheusService(t)

	s.RecordMetric(ConsumerGroupPartitionOwner, map[string]string{
		labelNamespace:     "ns",
		labelEventhub:      "eh",
		labelConsumerGroup: "cg",
		labelPartitionID:   "0",
		labelOwner:         "instance-uuid-1",
	}, 1.0)

	if got := countSeries(t, s.building.registry); got != 1 {
		t.Fatalf("expected 1 series in building set before reset, got %d", got)
	}

	s.StartCycle()

	if got := countSeries(t, s.building.registry); got != 0 {
		t.Fatalf("expected 0 series in building set after reset, got %d", got)
	}
}

// The scraped ("active") registry must never expose a half-collected cycle:
// while a new cycle is being built (post-ResetMetrics, pre-PushMetrics) the
// active set must still serve the previous complete snapshot.
func TestPrometheusActiveServesCompleteSnapshotDuringCollection(t *testing.T) {
	s := newTestPrometheusService(t)

	record := func(owner string) {
		s.RecordMetric(ConsumerGroupPartitionOwner, map[string]string{
			labelNamespace:     "ns",
			labelEventhub:      "eh",
			labelConsumerGroup: "cg",
			labelPartitionID:   "0",
			labelOwner:         owner,
		}, 1.0)
	}

	// cycle 1: collect and publish
	s.StartCycle()
	record("uuid-1")
	if err := s.PushMetrics(); err != nil {
		t.Fatalf("push failed: %v", err)
	}
	if got := countSeries(t, s.active.registry); got != 1 {
		t.Fatalf("expected 1 active series after cycle 1, got %d", got)
	}

	// cycle 2 starts: building is reset and only partially filled. The active
	// registry a scraper reads must STILL show cycle 1's complete snapshot.
	s.StartCycle()
	if got := countSeries(t, s.active.registry); got != 1 {
		t.Fatalf("expected active to still serve cycle 1 (1 series) mid-collection, got %d", got)
	}
	record("uuid-2")
	if got := countSeries(t, s.active.registry); got != 1 {
		t.Fatalf("expected active unchanged while cycle 2 builds, got %d", got)
	}

	// publish cycle 2: active flips to the new complete snapshot
	if err := s.PushMetrics(); err != nil {
		t.Fatalf("push failed: %v", err)
	}
	if got := countSeries(t, s.active.registry); got != 1 {
		t.Fatalf("expected 1 active series after cycle 2 (uuid-1 dropped), got %d", got)
	}
}

// distinct per-instance owner labels must each create a series (the leak this fixes),
// and ResetMetrics must drop all of them.
func TestPrometheusOwnerLabelAccumulatesThenResets(t *testing.T) {
	s := newTestPrometheusService(t)

	for _, owner := range []string{"uuid-1", "uuid-2", "uuid-3"} {
		s.RecordMetric(ConsumerGroupPartitionOwner, map[string]string{
			labelNamespace:     "ns",
			labelEventhub:      "eh",
			labelConsumerGroup: "cg",
			labelPartitionID:   "0",
			labelOwner:         owner,
		}, 1.0)
	}

	if got := countSeries(t, s.building.registry); got != 3 {
		t.Fatalf("expected 3 series for 3 distinct owners, got %d", got)
	}

	s.StartCycle()

	if got := countSeries(t, s.building.registry); got != 0 {
		t.Fatalf("expected 0 series after reset, got %d", got)
	}
}

func TestPushGatewayResetMetricsDropsStaleSeries(t *testing.T) {
	s, ok := NewPushGatewayService("http://localhost:9091").(*pushGatewayService)
	if !ok {
		t.Fatal("NewPushGatewayService did not return *pushGatewayService")
	}

	s.RecordMetric(ConsumerGroupPartitionOwner, map[string]string{
		labelNamespace:     "ns",
		labelEventhub:      "eh",
		labelConsumerGroup: "cg",
		labelPartitionID:   "0",
		labelOwner:         "instance-uuid-1",
	}, 1.0)

	if got := countSeries(t, s.registry); got != 1 {
		t.Fatalf("expected 1 series before reset, got %d", got)
	}

	s.StartCycle()

	if got := countSeries(t, s.registry); got != 0 {
		t.Fatalf("expected 0 series after reset, got %d", got)
	}
}

func TestNewOtlpServiceUnsupportedProtocol(t *testing.T) {
	_, err := NewOtlpService("http://localhost:4317", "carrier-pigeon")
	if err == nil {
		t.Fatal("expected error for unsupported protocol, got nil")
	}
}

func newTestPrometheusService(t *testing.T) *prometheusService {
	t.Helper()
	s, ok := NewPrometheusService().(*prometheusService)
	if !ok {
		t.Fatal("NewPrometheusService did not return *prometheusService")
	}
	return s
}

func countSeries(t *testing.T, registry *prometheus.Registry) int {
	t.Helper()
	mfs, err := registry.Gather()
	if err != nil {
		t.Fatalf("gather failed: %v", err)
	}
	count := 0
	for _, mf := range mfs {
		if strings.HasPrefix(mf.GetName(), metricPrefix) {
			count += len(mf.GetMetric())
		}
	}
	return count
}
