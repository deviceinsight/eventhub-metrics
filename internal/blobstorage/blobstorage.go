package blobstorage

import (
	"fmt"
	"net/url"
	"path"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azeventhubs/checkpoints"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
)

func GetBlobStore(credential *azidentity.DefaultAzureCredential, endpoint,
	container string) (*checkpoints.BlobStore, error) {

	storageServiceURL, err := getStorageServiceURL(endpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to parse storage service url: %w", err)
	}

	blobClient, err := azblob.NewClient(storageServiceURL.String(), credential, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating blob client: %w", err)
	}

	azBlobContainerClient := blobClient.ServiceClient().NewContainerClient(container)

	return checkpoints.NewBlobStore(azBlobContainerClient, nil)
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
