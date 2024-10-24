package collector

import (
	"context"
	"fmt"
	"log/slog"
	"regexp"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azeventhubs"
	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azeventhubs/checkpoints"
	"github.com/deviceinsight/eventhub-metrics/internal/config"
	"github.com/deviceinsight/eventhub-metrics/internal/eventhub"
	"github.com/deviceinsight/eventhub-metrics/internal/metrics"
)

type Service interface {
	ProcessNamespace(ctx context.Context, credential *azidentity.DefaultAzureCredential,
		endpoint string) (string, []eventhub.Details, error)
	ProcessEventHub(ctx context.Context, credential *azidentity.DefaultAzureCredential,
		blobStore *checkpoints.BlobStore, namespace, endpoint string, eventHubDetails *eventhub.Details,
		excludeConsumerGroupsRegex *regexp.Regexp) error
}

type service struct {
	metrics metrics.Service
	cfg     config.CollectorConfig
}

func NewService(metrics metrics.Service, cfg config.CollectorConfig) Service {
	return &service{
		metrics: metrics,
		cfg:     cfg,
	}
}

func (s *service) ProcessNamespace(ctx context.Context, credential *azidentity.DefaultAzureCredential,
	endpoint string) (string, []eventhub.Details, error) {

	namespace, err := eventhub.GetNamespaceName(endpoint)
	if err != nil {
		return "", nil, fmt.Errorf("failed get namespace name: %w", err)
	}

	if err := s.metrics.RecordNamespaceInfo(namespace, endpoint); err != nil {
		return "", nil, fmt.Errorf("failed to record namespace info metric: %w", err)
	}
	eventHubs, err := eventhub.GetEventHubs(ctx, credential, endpoint)
	return namespace, eventHubs, err
}

func (s *service) ProcessEventHub(ctx context.Context, credential *azidentity.DefaultAzureCredential,
	blobStore *checkpoints.BlobStore, namespace, endpoint string, eventHubDetails *eventhub.Details,
	excludeConsumerGroupsRegex *regexp.Regexp) error {

	consumerGroups, err := eventhub.GetConsumerGroups(ctx, credential, endpoint, eventHubDetails.Name)
	if err != nil {
		return fmt.Errorf("failed to get consumer groups: %w", err)
	}
	sequenceNumbers, err := eventhub.GetSequenceNumbers(ctx, credential, endpoint, eventHubDetails)
	if err != nil {
		return fmt.Errorf("failed to get sequence numbers: %w", err)
	}

	if err := s.metrics.RecordEventhubInfo(namespace, eventHubDetails.Name, eventHubDetails.PartitionCount,
		eventHubDetails.MessageRetentionInDays); err != nil {
		return fmt.Errorf("failed to record eventhub info metric: %w", err)
	}

	seqSum := eventhub.SequenceNumbers{}

	for partitionID, seq := range sequenceNumbers {
		if err := s.metrics.RecordEventhubPartitionSequenceNumber(namespace, eventHubDetails.Name, partitionID,
			seq.Min, seq.Max); err != nil {
			return fmt.Errorf("failed to record eventhub partition sequence number metric: %w", err)
		}
		seqSum.Min += seq.Min
		seqSum.Max += seq.Max
	}

	if err := s.metrics.RecordEventhubSequenceNumberSum(namespace, eventHubDetails.Name,
		seqSum.Min, seqSum.Max); err != nil {
		return fmt.Errorf("failed to record eventhub sequence number sum metric: %w", err)
	}

	for _, consumerGroup := range consumerGroups {

		if excludeConsumerGroupsRegex != nil && excludeConsumerGroupsRegex.MatchString(consumerGroup) {
			slog.Debug("skipping excluded consumerGroup", "consumerGroup", consumerGroup)
			continue
		}

		if err := s.processConsumerGroup(ctx, blobStore, endpoint, eventHubDetails.Name, consumerGroup, sequenceNumbers,
			namespace, eventHubDetails.PartitionCount); err != nil {
			return err
		}
	}
	return nil
}

//nolint:gocognit
func (s *service) processConsumerGroup(ctx context.Context, blobStore *checkpoints.BlobStore, endpoint string,
	eventHub string, consumerGroup string, sequenceNumbers map[string]eventhub.SequenceNumbers, namespace string,
	partitionCount int) error {

	checkpointList, err := blobStore.ListCheckpoints(ctx, endpoint, eventHub, consumerGroup, nil)
	if err != nil {
		return fmt.Errorf("failed to list checkpoints: %w", err)
	}

	lagSum := int64(0)
	sequenceSum := int64(0)

	for _, checkpoint := range checkpointList {
		lag := sequenceNumbers[checkpoint.PartitionID].Max
		if checkpoint.SequenceNumber != nil {
			lag = sequenceNumbers[checkpoint.PartitionID].Max - *checkpoint.SequenceNumber
			sequenceSum += *checkpoint.SequenceNumber
		}
		if lag < 0 {
			slog.Warn("negative lag", "namespace", endpoint, "eventHub", eventHub,
				"consumerGroup", consumerGroup, "partition", checkpoint.PartitionID)
			lag = 0
		}

		lagSum += lag

		if err := s.metrics.RecordConsumerGroupPartitionLag(namespace, eventHub, consumerGroup, checkpoint.PartitionID,
			lag); err != nil {
			return fmt.Errorf("failed to record consumer group partition lag metric: %w", err)
		}
	}

	if err := s.metrics.RecordConsumerGroupLag(namespace, eventHub, consumerGroup, lagSum); err != nil {
		return fmt.Errorf("failed to record consumer group lag metric: %w", err)
	}

	if err := s.metrics.RecordConsumerGroupEvents(namespace, eventHub, consumerGroup, sequenceSum); err != nil {
		return fmt.Errorf("failed to record consumer group events metric: %w", err)
	}

	ownerships, err := blobStore.ListOwnership(ctx, endpoint, eventHub, consumerGroup, nil)
	if err != nil {
		return fmt.Errorf("failed to list ownership: %w", err)
	}

	var expiredOwnerships []azeventhubs.Ownership
	var activeOwnerships []azeventhubs.Ownership

	for _, ownership := range ownerships {
		expired := false
		if eventhub.IsOwnershipExpired(ownership, s.cfg.OwnershipExpirationDuration) {
			expiredOwnerships = append(expiredOwnerships, ownership)
			expired = true
		} else {
			activeOwnerships = append(activeOwnerships, ownership)
		}
		if err := s.metrics.RecordConsumerGroupPartitionOwner(namespace, eventHub, consumerGroup, ownership.PartitionID,
			ownership.OwnerID, expired); err != nil {
			return fmt.Errorf("failed to record consumer group partition owner metric: %w", err)
		}
	}

	if err := s.metrics.RecordConsumerGroupOwners(namespace, eventHub, consumerGroup, len(activeOwnerships)); err != nil {
		return fmt.Errorf("failed to record consumer group owners metric: %w", err)
	}

	state := "unstable"
	if len(activeOwnerships) == partitionCount {
		state = "stable"
	} else if len(activeOwnerships) == 0 {
		state = "empty"
	}

	if err := s.metrics.RecordConsumerGroupInfo(namespace, eventHub, consumerGroup, state); err != nil {
		return fmt.Errorf("failed to record consumer group info metric: %w", err)
	}
	return nil
}
