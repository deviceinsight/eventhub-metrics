package blobstorage

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"
	"path"
	"regexp"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/container"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azeventhubs/v2/checkpoints"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
)

// MinDirectoryDepth container should have directories namespace/eventHub/consumerGroup.
const MinDirectoryDepth = 3

type StorageContainer struct {
	Endpoint  string
	Container string
}

type StoredConsumerGroup struct {
	Namespace     string
	Eventhub      string
	ConsumerGroup string
}

type StoredGroupsMap = map[StorageContainer][]StoredConsumerGroup

func GetContainerInfos(ctx context.Context, credential *azidentity.DefaultAzureCredential, endpoint string,
	includedContainersRegex, excludedContainersRegex *regexp.Regexp) (StoredGroupsMap, error) {

	blobClient, err := getBlobClient(credential, endpoint)
	if err != nil {
		return nil, err
	}

	containers, err := getContainers(ctx, blobClient, includedContainersRegex, excludedContainersRegex)
	if err != nil {
		return nil, err
	}

	infos := make(StoredGroupsMap)

	for _, containerName := range containers {
		containerClient := blobClient.ServiceClient().NewContainerClient(containerName)
		directoryTrees, err := listDirectories(ctx, containerClient, "", MinDirectoryDepth)
		if err != nil {
			return nil, err
		}

		storedConsumerGroups := make([]StoredConsumerGroup, 0)

		for _, directoryTree := range directoryTrees {
			parts := strings.Split(directoryTree, "/")
			if len(parts) != (MinDirectoryDepth + 1) {
				slog.Warn("ignoring invalid storage container", "endpoint", endpoint, "container",
					containerName, "tree", directoryTrees)
				continue
			}

			storedConsumerGroups = append(storedConsumerGroups, StoredConsumerGroup{
				Namespace:     parts[0],
				Eventhub:      parts[1],
				ConsumerGroup: parts[2],
			})
		}

		if len(storedConsumerGroups) > 0 {
			infos[StorageContainer{
				Endpoint:  endpoint,
				Container: containerName,
			}] = storedConsumerGroups
		}

	}

	return infos, nil
}

func listDirectories(ctx context.Context, client *container.Client, prefix string, maxDepth int) ([]string, error) {

	pager := client.NewListBlobsHierarchyPager("/", &container.ListBlobsHierarchyOptions{
		Prefix: to.Ptr(prefix),
	})

	if maxDepth == 0 {
		return []string{prefix}, nil
	}

	directories := make([]string, 0)

	for pager.More() {
		resp, err := pager.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to list blobs: %w", err)
		}

		if resp.Segment.BlobPrefixes != nil {
			for _, blobPrefix := range resp.Segment.BlobPrefixes {
				subDirectories, err := listDirectories(ctx, client, *blobPrefix.Name, maxDepth-1)
				if err != nil {
					return nil, err
				}

				for _, subDir := range subDirectories {
					directories = append(directories, subDir)
				}
			}
		}
	}

	return directories, nil
}

func getContainers(ctx context.Context, client *azblob.Client,
	includedContainersRegex, excludedContainersRegex *regexp.Regexp) ([]string, error) {

	// List the containers in the storage account and include metadata
	pager := client.NewListContainersPager(nil)
	containers := make([]string, 0)

	for pager.More() {
		resp, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, containerItem := range resp.ContainerItems {

			containerName := *containerItem.Name

			if includedContainersRegex != nil && !includedContainersRegex.MatchString(containerName) {
				slog.Debug("skipping non-included container", "container", containerName,
					"regex", includedContainersRegex.String())
				continue
			}

			if excludedContainersRegex != nil && excludedContainersRegex.MatchString(containerName) {
				slog.Debug("skipping excluded container", "container", containerName,
					"regex", excludedContainersRegex.String())
				continue
			}

			containers = append(containers, containerName)
		}
	}

	return containers, nil
}

func GetBlobStores(credential *azidentity.DefaultAzureCredential, storedGroupsMap StoredGroupsMap,
	namespace, eventHub string) (map[string]*checkpoints.BlobStore, error) {

	consumerGroupBlobStores := make(map[string]*checkpoints.BlobStore)

	for storageContainer, storedConsumerGroups := range storedGroupsMap {

		consumerGroups := make([]string, 0)

		for _, storedConsumerGroup := range storedConsumerGroups {
			if storedConsumerGroup.Namespace == namespace && storedConsumerGroup.Eventhub == eventHub {
				consumerGroups = append(consumerGroups, storedConsumerGroup.ConsumerGroup)
			}
		}

		if len(consumerGroups) > 0 {
			blobStore, err := getBlobStore(credential, storageContainer)
			if err != nil {
				return nil, fmt.Errorf("unable to create blob store for endpoint=%s, error=%w",
					storageContainer.Endpoint, err)
			}

			for _, consumerGroup := range consumerGroups {
				consumerGroupBlobStores[consumerGroup] = blobStore
			}
		}
	}

	return consumerGroupBlobStores, nil
}

func getBlobStore(credential *azidentity.DefaultAzureCredential,
	storageContainer StorageContainer) (*checkpoints.BlobStore, error) {

	storageServiceURL, err := getStorageServiceURL(storageContainer.Endpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to parse storage service url: %w", err)
	}

	blobClient, err := azblob.NewClient(storageServiceURL.String(), credential, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating blob client: %w", err)
	}

	azBlobContainerClient := blobClient.ServiceClient().NewContainerClient(storageContainer.Container)

	return checkpoints.NewBlobStore(azBlobContainerClient, nil)
}

func getBlobClient(credential *azidentity.DefaultAzureCredential, endpoint string) (*azblob.Client, error) {
	storageServiceURL, err := getStorageServiceURL(endpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to parse storage service url: %w", err)
	}

	blobClient, err := azblob.NewClient(storageServiceURL.String(), credential, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating blob client: %w", err)
	}
	return blobClient, nil
}

func getStorageServiceURL(endpoint string, paths ...string) (*url.URL, error) {
	u, err := url.Parse(fmt.Sprintf("https://%s", endpoint))
	if err != nil {
		return nil, err
	}
	for _, p := range paths {
		u.Path = path.Join(u.Path, p)
	}
	return u, nil
}
