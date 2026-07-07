package metrics

import (
	"fmt"
)

type Service interface {
	RecordNamespaceInfo(namespace, endpoint string)
	RecordEventhubInfo(namespace, eventhub string, partitionCount, messageRetentionInDays int)
	RecordEventhubPartitionSequenceNumber(namespace, eventhub, partitionID string, seqMin, seqMax int64)
	RecordEventhubSequenceNumberSum(namespace, eventhub string, seqMin, seqMax int64)
	RecordConsumerGroupInfo(namespace, eventhub, consumerGroup string, state string)
	RecordConsumerGroupOwners(namespace, eventhub, consumerGroup string, ownerCount int)
	RecordConsumerGroupEvents(namespace, eventhub, consumerGroup string, eventCount int64)
	RecordConsumerGroupPartitionOwner(namespace, eventhub, consumerGroup, partitionID, owner string, expired bool)
	RecordConsumerGroupPartitionLag(namespace, eventhub, consumerGroup, partitionID string, lag int64)
	RecordConsumerGroupLag(namespace, eventhub, consumerGroup string, lag int64)
	StartCollectionCycle()
	PushMetrics() error
}

type RecordService interface {
	StartCycle()
	RecordMetric(metric *Metric, labels map[string]string, value float64)
	PushMetrics() error
}

type service struct {
	recorder RecordService
}

func (s *service) RecordNamespaceInfo(namespace, endpoint string) {
	s.recorder.RecordMetric(NamespaceInfo, map[string]string{
		labelNamespace: namespace,
		"eh_endpoint":  endpoint},
		1.0)
}

func (s *service) RecordEventhubInfo(namespace, eventhub string, partitionCount int, messageRetentionInDays int) {
	s.recorder.RecordMetric(EventhubInfo, map[string]string{
		labelNamespace:      namespace,
		labelEventhub:       eventhub,
		"partition_count":   fmt.Sprintf("%d", partitionCount),
		"retention_in_days": fmt.Sprintf("%d", messageRetentionInDays)},
		1.0)
}

func (s *service) RecordEventhubPartitionSequenceNumber(namespace, eventhub, partitionID string, seqMin, seqMax int64) {
	s.recorder.RecordMetric(EventhubPartitionSequenceNumberMin, map[string]string{
		labelNamespace:   namespace,
		labelEventhub:    eventhub,
		labelPartitionID: partitionID},
		float64(seqMin))

	s.recorder.RecordMetric(EventhubPartitionSequenceNumberMax, map[string]string{
		labelNamespace:   namespace,
		labelEventhub:    eventhub,
		labelPartitionID: partitionID},
		float64(seqMax))
}

func (s *service) RecordEventhubSequenceNumberSum(namespace, eventhub string, seqMin, seqMax int64) {
	s.recorder.RecordMetric(EventhubSequenceNumberMinSum, map[string]string{
		labelNamespace: namespace,
		labelEventhub:  eventhub},
		float64(seqMin))

	s.recorder.RecordMetric(EventhubSequenceNumberMaxSum, map[string]string{
		labelNamespace: namespace,
		labelEventhub:  eventhub},
		float64(seqMax))
}

func (s *service) RecordConsumerGroupInfo(namespace, eventhub string, consumerGroup string, state string) {

	value := 1.0
	if state != "stable" {
		value = 0.0
	}

	s.recorder.RecordMetric(ConsumerGroupInfo, map[string]string{
		labelNamespace:     namespace,
		labelEventhub:      eventhub,
		labelConsumerGroup: consumerGroup,
		"state":            state},
		value)
}

func (s *service) RecordConsumerGroupOwners(namespace, eventhub, consumerGroup string, ownerCount int) {
	s.recorder.RecordMetric(ConsumerGroupOwners, map[string]string{
		labelNamespace:     namespace,
		labelEventhub:      eventhub,
		labelConsumerGroup: consumerGroup},
		float64(ownerCount))
}

func (s *service) RecordConsumerGroupEvents(namespace, eventhub, consumerGroup string, eventCount int64) {
	s.recorder.RecordMetric(ConsumerGroupEventsSum, map[string]string{
		labelNamespace:     namespace,
		labelEventhub:      eventhub,
		labelConsumerGroup: consumerGroup},
		float64(eventCount))
}

func (s *service) RecordConsumerGroupPartitionOwner(namespace, eventhub, consumerGroup, partitionID, owner string,
	expired bool) {

	value := 1.0
	if expired {
		value = 0.0
	}

	s.recorder.RecordMetric(ConsumerGroupPartitionOwner, map[string]string{
		labelNamespace:     namespace,
		labelEventhub:      eventhub,
		labelConsumerGroup: consumerGroup,
		labelPartitionID:   partitionID,
		labelOwner:         owner},
		value)
}

func (s *service) RecordConsumerGroupPartitionLag(namespace, eventhub, consumerGroup, partitionID string, lag int64) {
	s.recorder.RecordMetric(ConsumerGroupPartitionLag, map[string]string{
		labelNamespace:     namespace,
		labelEventhub:      eventhub,
		labelConsumerGroup: consumerGroup,
		labelPartitionID:   partitionID},
		float64(lag))
}

func (s *service) RecordConsumerGroupLag(namespace, eventhub, consumerGroup string, lag int64) {
	s.recorder.RecordMetric(ConsumerGroupLag, map[string]string{
		labelNamespace:     namespace,
		labelEventhub:      eventhub,
		labelConsumerGroup: consumerGroup},
		float64(lag))
}

func (s *service) StartCollectionCycle() {
	s.recorder.StartCycle()
}

func (s *service) PushMetrics() error {
	return s.recorder.PushMetrics()
}
