package eventhub

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
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

	token, err := getToken(ctx, credential, endpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to get token: %w", err)
	}

	requestURL, err := GetNamespaceURL(endpoint, "/$Resources/Eventhubs?api-version=2014-01")
	if err != nil {
		return nil, fmt.Errorf("failed to parse event hubs request url: %w", err)
	}

	response, err := performRequest(ctx, token, requestURL)
	if err != nil {
		return nil, fmt.Errorf("failed to peform request: %w", err)
	}
	defer closeBody(response)

	feed, err := parseXMLResponse(response)
	if err != nil {
		return nil, fmt.Errorf("failed to parse xml response: %w", err)
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

	token, err := getToken(ctx, credential, endpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to get token: %w", err)
	}

	requestURL, err := GetNamespaceURL(endpoint, eventHub, "/consumergroups?api-version=2014-01")
	if err != nil {
		return nil, fmt.Errorf("failed to parse consumer groups request url: %w", err)
	}

	response, err := performRequest(ctx, token, requestURL)
	if err != nil {
		return nil, fmt.Errorf("failed to peform request: %w", err)
	}
	defer closeBody(response)

	feed, err := parseXMLResponse(response)
	if err != nil {
		return nil, fmt.Errorf("failed to parse xml response: %w", err)
	}
	consumerGroups := make([]string, len(feed.Entry))
	for i, entry := range feed.Entry {
		consumerGroups[i] = entry.Title
	}

	return consumerGroups, nil
}

func getToken(ctx context.Context, credential *azidentity.DefaultAzureCredential, endpoint string) (string, error) {

	scope, err := GetNamespaceURL(endpoint, "/.default")
	if err != nil {
		return "", fmt.Errorf("failed to parse scope: %w", err)
	}

	options := policy.TokenRequestOptions{Scopes: []string{scope.String()}}
	token, err := credential.GetToken(ctx, options)
	if err != nil {
		return "", fmt.Errorf("failed to get token: %w", err)
	}
	return token.Token, nil
}

func performRequest(ctx context.Context, token string, url *url.URL) (*http.Response, error) {

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", token))

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make http request: %w", err)
	}

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return nil, fmt.Errorf("request failed with status=%d", res.StatusCode)
	}
	return res, nil
}

func closeBody(response *http.Response) {
	if response == nil {
		return
	}
	_ = response.Body.Close()
}

func GetSequenceNumbers(ctx context.Context, credential *azidentity.DefaultAzureCredential,
	endpoint string, eventhubDetails *Details) (map[string]SequenceNumbers, error) {

	eventhubURL, err := GetNamespaceURL(endpoint)
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

func GetNamespaceURL(endpoint string, paths ...string) (*url.URL, error) {
	u, err := url.Parse(fmt.Sprintf("https://%s", endpoint))
	if err != nil {
		return nil, err
	}
	for _, p := range paths {
		u.Path = path.Join(u.Path, p)
	}
	return u, nil
}

func GetNamespaceName(endpoint string) (string, error) {
	u, err := GetNamespaceURL(endpoint)
	if err != nil {
		return "", fmt.Errorf("failed to parse u: %w", err)
	}
	return strings.SplitN(u.Hostname(), ".", 2)[0], nil //nolint:mnd // we just need to split the subdomain
}
