package blobstorage

import (
	"fmt"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azeventhubs/checkpoints"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
)

func GetBlobStore(credential *azidentity.DefaultAzureCredential, endpoint, container string) (*checkpoints.BlobStore, error) {

	storageServiceURL := fmt.Sprintf("https://%s/", endpoint)

	blobClient, err := azblob.NewClient(storageServiceURL, credential, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating blob client: %w", err)
	}

	azBlobContainerClient := blobClient.ServiceClient().NewContainerClient(container)

	return checkpoints.NewBlobStore(azBlobContainerClient, nil)
}
