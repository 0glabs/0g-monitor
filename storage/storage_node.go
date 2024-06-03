package storage

import (
	"fmt"
	"time"

	"github.com/0glabs/0g-storage-client/node"
	"github.com/Conflux-Chain/go-conflux-util/health"
	"github.com/sirupsen/logrus"
)

type StorageNode struct {
	client  node.Client
	name    string
	address string
	health  health.TimedCounter
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

func (node *StorageNode) CheckStatus(config health.TimedCounterConfig, privateKey string) {
	_, err := node.client.ZeroGStorage().GetStatus()
	if logrus.IsLevelEnabled(logrus.DebugLevel) {
		logrus.WithFields(logrus.Fields{
			"storage node": node.name,
			"address":      node.address,
		}).Debug("Storage node status report")
	}

	if err != nil {
		unhealthy, unrecovered, elapsed := node.health.OnFailure(config)

		if unhealthy {
			logrus.WithFields(logrus.Fields{
				"elapsed":      prettyElapsed(elapsed),
				"storage node": node.String(),
			}).Error("Storage node disconnected")
		}

		if unrecovered {
			logrus.WithFields(logrus.Fields{
				"elapsed":      prettyElapsed(elapsed),
				"storage node": node.String(),
			}).Error("Storage node disconnected and not recovered yet")
		}
	} else if recovered, elapsed := node.health.OnSuccess(config); recovered {
		logrus.WithFields(logrus.Fields{
			"elapsed":      prettyElapsed(elapsed),
			"storage node": node.String(),
		}).Warn("Storage node recovered now")
	}
}

func prettyElapsed(elapsed time.Duration) string {
	return fmt.Sprint(elapsed.Truncate(time.Second))
}
