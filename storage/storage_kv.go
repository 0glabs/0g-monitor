package storage

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/0glabs/0g-storage-client/node"
	"github.com/Conflux-Chain/go-conflux-util/health"
	providers "github.com/openweb3/go-rpc-provider/provider_wrapper"

	"github.com/sirupsen/logrus"
)

type KvNode struct {
	client           *node.KvClient
	backupClient     *node.KvClient
	discordId        string
	validatorAddress string
	ip               string
	health           health.TimedCounter
}

func MustNewKvNode(discordId, validatorAddress, ip string) *KvNode {
	storageNode, err := NewKvNode(discordId, validatorAddress, ip)
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"address": validatorAddress,
			"ip":      ip,
		}).WithError(err).Info("Failed to create kv node")
		return nil
	}

	return storageNode
}

func NewKvNode(discordId, validatorAddress, ip string) (*KvNode, error) {
	ip = strings.TrimSpace(ip)
	if len(ip) == 0 {
		return nil, fmt.Errorf("empty ip")
	}

	if strings.HasPrefix(ip, "http") {
		client, err := node.NewKvClient(ip)
		if err != nil {
			return nil, err
		}
		return &KvNode{
			client:           client,
			discordId:        discordId,
			validatorAddress: validatorAddress,
			ip:               ip,
		}, nil
	}

	client, err := node.NewKvClient("http://"+ip, providers.Option{
		RequestTimeout: DefaultTimeout,
	})
	if err != nil {
		return nil, err
	}
	backupClient, err := node.NewKvClient("https://"+ip, providers.Option{
		RequestTimeout: DefaultTimeout,
	})
	if err != nil {
		backupClient = nil
	}

	return &KvNode{
		client:           client,
		backupClient:     backupClient,
		discordId:        discordId,
		validatorAddress: validatorAddress,
		ip:               ip,
	}, nil
}

func (kvNode *KvNode) CheckStatus(config health.TimedCounterConfig) {
	_, err := kvNode.client.GetHoldingStreamIds(context.Background())
	if logrus.IsLevelEnabled(logrus.DebugLevel) {
		logrus.WithFields(logrus.Fields{
			"ip": kvNode.ip,
		}).Debug("Kv node status report")
	}

	if err != nil {
		unhealthy, unrecovered, elapsed := kvNode.health.OnFailure(config)

		if unhealthy {
			logrus.WithFields(logrus.Fields{
				"elapsed": PrettyElapsed(elapsed),
				"ip":      kvNode.ip,
			}).Error("Kv node disconnected")
		}

		if unrecovered {
			logrus.WithFields(logrus.Fields{
				"elapsed": PrettyElapsed(elapsed),
				"ip":      kvNode.ip,
			}).Error("Kv node disconnected and not recovered yet")
		}
	} else if recovered, elapsed := kvNode.health.OnSuccess(config); recovered {
		logrus.WithFields(logrus.Fields{
			"elapsed": PrettyElapsed(elapsed),
			"ip":      kvNode.ip,
		}).Warn("Kv node recovered now")
	}
}

func (kvNode *KvNode) CheckStatusSilence(config health.TimedCounterConfig, db *sql.DB) {
	upsertQuery := `
        INSERT INTO user_kv_status (ip, discord_id, address, status)
        VALUES (?, ?, ?, ?)
        ON DUPLICATE KEY UPDATE
        status = VALUES(status)
	`

	_, err := kvNode.client.GetHoldingStreamIds(context.Background())
	if err != nil && kvNode.backupClient != nil {
		_, err = kvNode.client.GetHoldingStreamIds(context.Background())
	}

	if err != nil {
		logrus.WithFields(logrus.Fields{
			"address": kvNode.validatorAddress,
			"ip":      kvNode.ip,
		}).WithError(err).Info("Kv node connection failed")

		kvNode.health.OnFailure(config)
		_, err = db.Exec(upsertQuery, kvNode.ip, kvNode.discordId, kvNode.validatorAddress, NodeDisconnected)
		if err != nil {
			logrus.WithFields(logrus.Fields{
				"ip": kvNode.ip,
			}).Warn("Failed to update kv node status in db")
		}
	} else {
		logrus.WithFields(logrus.Fields{
			"address": kvNode.validatorAddress,
			"ip":      kvNode.ip,
		}).Info("Kv node connection succeeded")

		_, err = db.Exec(upsertQuery, kvNode.ip, kvNode.discordId, kvNode.validatorAddress, NodeConnected)
		if err != nil {
			logrus.WithFields(logrus.Fields{
				"ip": kvNode.ip,
			}).Warn("Failed to update kv node status in db")
		}
	}
}
