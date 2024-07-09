package storage

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/0glabs/0g-storage-client/node"
	"github.com/Conflux-Chain/go-conflux-util/health"
	providers "github.com/openweb3/go-rpc-provider/provider_wrapper"

	"github.com/sirupsen/logrus"
)

type StorageNode struct {
	client           *node.Client
	backupClient     *node.Client
	discordId        string
	validatorAddress string
	ip               string
	health           health.TimedCounter
}

func MustNewStorageNode(discordId, validatorAddress, ip string) *StorageNode {
	storageNode, err := NewStorageNode(discordId, validatorAddress, ip)
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"address": validatorAddress,
			"ip":      ip,
		}).WithError(err).Info("Failed to create storage node")
		return nil
	}

	return storageNode
}

func NewStorageNode(discordId, validatorAddress, ip string) (*StorageNode, error) {
	ip = strings.TrimSpace(ip)
	if len(ip) == 0 {
		return nil, fmt.Errorf("empty ip")
	}

	if strings.HasPrefix(ip, "http") {
		client, err := node.NewClient(ip)
		if err != nil {
			return nil, err
		}
		return &StorageNode{
			client:           client,
			discordId:        discordId,
			validatorAddress: validatorAddress,
			ip:               ip,
		}, nil
	}

	client, err := node.NewClient("http://"+ip, providers.Option{
		RequestTimeout: DefaultTimeout,
	})
	if err != nil {
		return nil, err
	}
	backupClient, err := node.NewClient("https://"+ip, providers.Option{
		RequestTimeout: DefaultTimeout,
	})
	if err != nil {
		backupClient = nil
	}

	return &StorageNode{
		client:           client,
		backupClient:     backupClient,
		discordId:        discordId,
		validatorAddress: validatorAddress,
		ip:               ip,
	}, nil
}

func (storageNode *StorageNode) CheckStatus(config health.TimedCounterConfig) {
	_, err := storageNode.client.ZeroGStorage().GetStatus()
	if logrus.IsLevelEnabled(logrus.DebugLevel) {
		logrus.WithFields(logrus.Fields{
			"ip": storageNode.ip,
		}).Debug("Storage node status report")
	}

	if err != nil {
		unhealthy, unrecovered, elapsed := storageNode.health.OnFailure(config)

		if unhealthy {
			logrus.WithFields(logrus.Fields{
				"elapsed": PrettyElapsed(elapsed),
				"ip":      storageNode.ip,
			}).Error("Storage node disconnected")
		}

		if unrecovered {
			logrus.WithFields(logrus.Fields{
				"elapsed": PrettyElapsed(elapsed),
				"ip":      storageNode.ip,
			}).Error("Storage node disconnected and not recovered yet")
		}
	} else if recovered, elapsed := storageNode.health.OnSuccess(config); recovered {
		logrus.WithFields(logrus.Fields{
			"elapsed": PrettyElapsed(elapsed),
			"ip":      storageNode.ip,
		}).Warn("Storage node recovered now")
	}
}

func (storageNode *StorageNode) CheckStatusSilence(config health.TimedCounterConfig, db *sql.DB) {
	upsertQuery := `
        INSERT INTO user_storage_status (ip, discord_id, address, status)
        VALUES (?, ?, ?, ?)
        ON DUPLICATE KEY UPDATE
        status = VALUES(status)
	`

	_, err := storageNode.client.ZeroGStorage().GetStatus()
	if err != nil && storageNode.backupClient != nil {
		_, err = storageNode.backupClient.ZeroGStorage().GetStatus()
	}

	if err != nil {
		logrus.WithFields(logrus.Fields{
			"address": storageNode.validatorAddress,
			"ip":      storageNode.ip,
		}).WithError(err).Info("Storage node connection failed")

		storageNode.health.OnFailure(config)
		_, err = db.Exec(upsertQuery, storageNode.ip, storageNode.discordId, storageNode.validatorAddress, NodeDisconnected)
		if err != nil {
			logrus.WithFields(logrus.Fields{
				"ip": storageNode.ip,
			}).Warn("Failed to update storage node status in db")
		}
	} else {
		logrus.WithFields(logrus.Fields{
			"address": storageNode.validatorAddress,
			"ip":      storageNode.ip,
		}).Info("Storage node connection succeeded")

		_, err = db.Exec(upsertQuery, storageNode.ip, storageNode.discordId, storageNode.validatorAddress, NodeConnected)
		if err != nil {
			logrus.WithFields(logrus.Fields{
				"ip": storageNode.ip,
			}).Warn("Failed to update storage node status in db")
		}
	}
}
