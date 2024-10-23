package da

import (
	"database/sql"

	"context"
	"time"

	pb "github.com/0glabs/0g-monitor/proto/da-node"
	"github.com/0glabs/0g-monitor/utils"
	"github.com/Conflux-Chain/go-conflux-util/health"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/sirupsen/logrus"
)

type DaNode struct {
	discordId        string
	validatorAddress string
	ip               string
	health           health.TimedCounter
}

func MustNewDaNode(discordId, validatorAddress, ip string) *DaNode {
	return &DaNode{
		discordId:        discordId,
		validatorAddress: validatorAddress,
		ip:               ip,
	}

}

func (daNode *DaNode) CheckStatus(config health.TimedCounterConfig, db *sql.DB) {
	conn, err := grpc.NewClient(daNode.ip, []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}...)
	if err == nil {
		defer conn.Close()
		c := pb.NewSignerClient(conn)
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
		defer cancel()
		_, err = c.GetStatus(ctx, &pb.Empty{})
	}

	if logrus.IsLevelEnabled(logrus.DebugLevel) {
		logrus.WithFields(logrus.Fields{
			"ip": daNode.ip,
		}).Debug("Da node status report")
	}

	if err != nil {
		unhealthy, unrecovered, elapsed := daNode.health.OnFailure(config)

		if unhealthy {
			logrus.WithFields(logrus.Fields{
				"elapsed": utils.PrettyElapsed(elapsed),
				"ip":      daNode.ip,
			}).Error("Storage node disconnected")
		}

		if unrecovered {
			logrus.WithFields(logrus.Fields{
				"elapsed": utils.PrettyElapsed(elapsed),
				"ip":      daNode.ip,
			}).Error("Da node disconnected and not recovered yet")
		}
	} else if recovered, elapsed := daNode.health.OnSuccess(config); recovered {
		logrus.WithFields(logrus.Fields{
			"elapsed": utils.PrettyElapsed(elapsed),
			"ip":      daNode.ip,
		}).Warn("Da node recovered now")
	}
}

func (daNode *DaNode) CheckStatusSilence(config health.TimedCounterConfig, db *sql.DB) {
	upsertQuery := `
        INSERT INTO user_da_node_status (ip, discord_id, address, status)
        VALUES (?, ?, ?, ?)
        ON DUPLICATE KEY UPDATE
        status = VALUES(status)
	`

	conn, err := grpc.NewClient(daNode.ip, []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}...)

	if err == nil {
		defer conn.Close()
		c := pb.NewSignerClient(conn)
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
		defer cancel()
		_, err = c.GetStatus(ctx, &pb.Empty{})
	}

	if err != nil {
		logrus.WithFields(logrus.Fields{
			"address": daNode.validatorAddress,
			"ip":      daNode.ip,
		}).WithError(err).Info("Da node connection failed")

		daNode.health.OnFailure(config)
		_, err = db.Exec(upsertQuery, daNode.ip, daNode.discordId, daNode.validatorAddress, NodeDisconnected)
		if err != nil {
			logrus.WithFields(logrus.Fields{
				"ip": daNode.ip,
			}).Warn("Failed to update da node status in db")
		}
	} else {
		logrus.WithFields(logrus.Fields{
			"address": daNode.validatorAddress,
			"ip":      daNode.ip,
		}).Info("Da node connection succeeded")

		_, err = db.Exec(upsertQuery, daNode.ip, daNode.discordId, daNode.validatorAddress, NodeConnected)
		if err != nil {
			logrus.WithFields(logrus.Fields{
				"ip": daNode.ip,
			}).Warn("Failed to update da node status in db")
		}
	}
}
