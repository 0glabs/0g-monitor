package storage

import (
	"net/http"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/ybbus/jsonrpc/v2"
)

type StorageNode struct {
	client  jsonrpc.RPCClient
	name    string
	address string
}

func MustNewStorageNode(name, address string) *StorageNode {

	storageNode, err := NewStorageNode(name, address)
	if err != nil {
		logrus.WithError(err).Fatal("Failed to create storage node")
	}

	return storageNode
}

func NewStorageNode(name, address string) (*StorageNode, error) {
	httpClient := &http.Client{
		Timeout: 5 * time.Second, // Set timeout to 5 seconds
	}

	client := jsonrpc.NewClientWithOpts(address, &jsonrpc.RPCClientOpts{
		HTTPClient: httpClient,
	})

	return &StorageNode{
		client:  client,
		name:    name,
		address: address,
	}, nil
}

func (node StorageNode) String() string {
	if len(node.name) == 0 {
		return node.address
	}

	return node.name
}

func (node *StorageNode) CheckStatus(privateKey string) {
	_, err := node.client.Call("zgs_getStatus")
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"storage node": node.name,
		}).Error("Storage node became unhealthy")
		return
	}
}
