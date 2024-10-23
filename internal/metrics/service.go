package metrics

import (
	"fmt"
)

type Service interface {
	RecordNamespaceInfo(namespace, endpoint string) error
	RecordEventhubInfo(namespace, eventhub string, partitionCount, messageRetentionInDays int) error
	RecordEventhubPartitionSequenceNumber(namespace, eventhub, partitionID string, seqMin, seqMax int64) error
	RecordEventhubSequenceNumberSum(namespace, eventhub string, seqMin, seqMax int64) error
	RecordConsumerGroupInfo(namespace, eventhub, consumerGroup string, state string) error
	RecordConsumerGroupOwners(namespace, eventhub, consumerGroup string, ownerCount int) error
	RecordConsumerGroupEvents(namespace, eventhub, consumerGroup string, eventCount int64) error
	RecordConsumerGroupPartitionLag(namespace, eventhub, consumerGroup, partitionID string, lag int64) error
	RecordConsumerGroupLag(namespace, eventhub, consumerGroup string, lag int64) error
}

type RecordService interface {
	RecordMetric(metric *Metric, labels map[string]string, value float64) error
}

type service struct {
	recorder RecordService
}

func (s *service) RecordNamespaceInfo(namespace, endpoint string) error {
	return s.recorder.RecordMetric(NamespaceInfo, map[string]string{
		"eh_namespace": namespace,
		"eh_endpoint":  endpoint},
		1.0)
}

func (s *service) RecordEventhubInfo(namespace, eventhub string, partitionCount int, messageRetentionInDays int) error {
	return s.recorder.RecordMetric(EventhubInfo, map[string]string{
		"eh_namespace":      namespace,
		"eventhub":          eventhub,
		"partition_count":   fmt.Sprintf("%d", partitionCount),
		"retention_in_days": fmt.Sprintf("%d", messageRetentionInDays)},
		1.0)
}

func (s *service) RecordEventhubPartitionSequenceNumber(namespace, eventhub, partitionID string, seqMin,
	seqMax int64) error {
	err := s.recorder.RecordMetric(EventhubPartitionSequenceNumberMin, map[string]string{
		"eh_namespace": namespace,
		"eventhub":     eventhub,
		"partition_id": partitionID},
		float64(seqMin))
	if err != nil {
		return err
	}
	return s.recorder.RecordMetric(EventhubPartitionSequenceNumberMax, map[string]string{
		"eh_namespace": namespace,
		"eventhub":     eventhub,
		"partition_id": partitionID},
		float64(seqMax))
}

func (s *service) RecordEventhubSequenceNumberSum(namespace, eventhub string, seqMin, seqMax int64) error {
	err := s.recorder.RecordMetric(EventhubSequenceNumberMinSum, map[string]string{
		"eh_namespace": namespace,
		"eventhub":     eventhub},
		float64(seqMin))
	if err != nil {
		return err
	}
	return s.recorder.RecordMetric(EventhubSequenceNumberMaxSum, map[string]string{
		"eh_namespace": namespace,
		"eventhub":     eventhub},
		float64(seqMax))
}

func (s *service) RecordConsumerGroupInfo(namespace, eventhub string, consumerGroup string, state string) error {

	value := 1.0
	if state != "stable" {
		value = 0.0
	}

	return s.recorder.RecordMetric(ConsumerGroupInfo, map[string]string{
		"eh_namespace":   namespace,
		"eventhub":       eventhub,
		"consumer_group": consumerGroup},
		value)
}

func (s *service) RecordConsumerGroupOwners(namespace, eventhub, consumerGroup string, ownerCount int) error {
	return s.recorder.RecordMetric(ConsumerGroupOwners, map[string]string{
		"eh_namespace":   namespace,
		"eventhub":       eventhub,
		"consumer_group": consumerGroup},
		float64(ownerCount))
}

func (s *service) RecordConsumerGroupEvents(namespace, eventhub, consumerGroup string, eventCount int64) error {
	return s.recorder.RecordMetric(ConsumerGroupEventsSum, map[string]string{
		"eh_namespace":   namespace,
		"eventhub":       eventhub,
		"consumer_group": consumerGroup},
		float64(eventCount))
}

func (s *service) RecordConsumerGroupPartitionLag(namespace, eventhub, consumerGroup, partitionID string,
	lag int64) error {
	return s.recorder.RecordMetric(ConsumerGroupPartitionLag, map[string]string{
		"eh_namespace":   namespace,
		"eventhub":       eventhub,
		"consumer_group": consumerGroup,
		"partition_id":   partitionID},
		float64(lag))
}

func (s *service) RecordConsumerGroupLag(namespace, eventhub, consumerGroup string, lag int64) error {
	return s.recorder.RecordMetric(ConsumerGroupLag, map[string]string{
		"eh_namespace":   namespace,
		"eventhub":       eventhub,
		"consumer_group": consumerGroup},
		float64(lag))
}
