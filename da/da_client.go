package da

import (
	"database/sql"

	"context"
	"time"

	pb "github.com/0glabs/0g-monitor/proto/da-client"
	"github.com/Conflux-Chain/go-conflux-util/health"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/sirupsen/logrus"
)

type DaClient struct {
	discordId        string
	validatorAddress string
	ip               string
	health           health.TimedCounter
}

func MustNewDaClient(discordId, validatorAddress, ip string) *DaClient {
	return &DaClient{
		discordId:        discordId,
		validatorAddress: validatorAddress,
		ip:               ip,
	}

}

func (daClient *DaClient) CheckStatusSilence(config health.TimedCounterConfig, db *sql.DB) {
	upsertQuery := `
        INSERT INTO user_da_client_status (ip, discord_id, address, status)
        VALUES (?, ?, ?, ?) AS v
        ON DUPLICATE KEY UPDATE
        status = v.status
	`

	conn, err := grpc.NewClient(daClient.ip, []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}...)
	if err == nil {
		c := pb.NewDisperserClient(conn)
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
		defer cancel()
		_, err = c.GetStatus(ctx, &pb.Empty{})
	}
	defer conn.Close()

	if err != nil {
		logrus.WithFields(logrus.Fields{
			"address": daClient.validatorAddress,
			"ip":      daClient.ip,
		}).WithError(err).Info("Da client connection failed")

		daClient.health.OnFailure(config)
		_, err = db.Exec(upsertQuery, daClient.ip, daClient.discordId, daClient.validatorAddress, NodeDisconnected)
		if err != nil {
			logrus.WithFields(logrus.Fields{
				"ip": daClient.ip,
			}).Warn("Failed to update da client status in db")
		}
	} else {
		logrus.WithFields(logrus.Fields{
			"address": daClient.validatorAddress,
			"ip":      daClient.ip,
		}).Info("Da client connection succeeded")

		_, err = db.Exec(upsertQuery, daClient.ip, daClient.discordId, daClient.validatorAddress, NodeConnected)
		if err != nil {
			logrus.WithFields(logrus.Fields{
				"ip": daClient.ip,
			}).Warn("Failed to update da client status in db")
		}
	}
}
