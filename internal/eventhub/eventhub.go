package eventhub

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/deviceinsight/eventhub-metrics/internal/rest"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azeventhubs"
)

type Details struct {
	Name                   string
	PartitionCount         int
	PartitionIDs           []string
	MessageRetentionInDays int
}

type SequenceNumbers struct {
	Min int64
	Max int64
}

func GetEventHubs(ctx context.Context, credential *azidentity.DefaultAzureCredential,
	endpoint string) ([]Details, error) {

	token, err := rest.GetToken(ctx, credential, endpoint, "/.default")
	if err != nil {
		return nil, fmt.Errorf("failed to get token: %w", err)
	}

	requestURL, err := rest.GetURL(endpoint, "/$Resources/Eventhubs?api-version=2014-01")
	if err != nil {
		return nil, fmt.Errorf("failed to parse event hubs request url: %w", err)
	}

	feed, err := rest.PerformRequest(ctx, token, requestURL, parseXMLResponse)
	if err != nil {
		return nil, fmt.Errorf("failed to request event hubs: %w", err)
	}

	eventHubs := make([]Details, len(feed.Entry))
	for i, entry := range feed.Entry {
		eventHubs[i] = Details{
			Name:                   entry.Title,
			PartitionCount:         entry.Content.EventHubDescription.PartitionCount,
			PartitionIDs:           entry.Content.EventHubDescription.PartitionIDs,
			MessageRetentionInDays: entry.Content.EventHubDescription.MessageRetentionInDays,
		}
	}

	return eventHubs, nil
}

func GetConsumerGroups(ctx context.Context, credential *azidentity.DefaultAzureCredential, endpoint string,
	eventHub string) ([]string, error) {

	token, err := rest.GetToken(ctx, credential, endpoint, "/.default")
	if err != nil {
		return nil, fmt.Errorf("failed to get token: %w", err)
	}

	requestURL, err := rest.GetURL(endpoint, eventHub, "/consumergroups?api-version=2014-01")
	if err != nil {
		return nil, fmt.Errorf("failed to parse consumer groups request url: %w", err)
	}

	feed, err := rest.PerformRequest(ctx, token, requestURL, parseXMLResponse)
	if err != nil {
		return nil, fmt.Errorf("failed to request event hubs: %w", err)
	}

	consumerGroups := make([]string, len(feed.Entry))
	for i, entry := range feed.Entry {
		consumerGroups[i] = entry.Title
	}

	return consumerGroups, nil
}

func GetSequenceNumbers(ctx context.Context, credential *azidentity.DefaultAzureCredential,
	endpoint string, eventhubDetails *Details) (map[string]SequenceNumbers, error) {

	eventhubURL, err := rest.GetURL(endpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to parse eventhub url: %w", err)
	}

	consumerClient, err := azeventhubs.NewConsumerClient(eventhubURL.Hostname(), eventhubDetails.Name,
		azeventhubs.DefaultConsumerGroup, credential, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create consumer client: %w", err)
	}

	lastEnqueuedSequenceNumbers := make(map[string]SequenceNumbers)

	for _, partitionID := range eventhubDetails.PartitionIDs {
		partitionProps, err := consumerClient.GetPartitionProperties(ctx, partitionID, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to get partition properties: %w", err)
		}

		lastEnqueuedSequenceNumbers[partitionID] = SequenceNumbers{
			Min: partitionProps.BeginningSequenceNumber,
			Max: partitionProps.LastEnqueuedSequenceNumber,
		}
	}

	return lastEnqueuedSequenceNumbers, nil
}

func IsOwnershipExpired(ownership azeventhubs.Ownership, expirationDuration time.Duration) bool {

	// https://github.com/Azure/azure-sdk-for-go/blob/main/sdk/messaging/azeventhubs/processor_load_balancer.go#L142
	if time.Since(ownership.LastModifiedTime.UTC()) > expirationDuration {
		return true
	}

	return ownership.OwnerID == ""
}

func GetNamespaceName(endpoint string) (string, error) {
	u, err := rest.GetURL(endpoint)
	if err != nil {
		return "", fmt.Errorf("failed to parse u: %w", err)
	}
	return strings.SplitN(u.Hostname(), ".", 2)[0], nil //nolint:mnd // we just need to split the subdomain
}
