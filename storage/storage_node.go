package storage

import (
	"github.com/0glabs/0g-storage-client/node"
	"github.com/sirupsen/logrus"
)

type StorageNode struct {
	client  node.Client
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
	client := node.MustNewClient(address)

	return &StorageNode{
		client:  *client,
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
	_, err := node.client.ZeroGStorage().GetStatus()
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"storage node": node.name,
		}).Error("Storage node became unhealthy")
		return
	}
}
