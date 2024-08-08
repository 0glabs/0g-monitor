package da

import (
	"context"
	"fmt"
	"time"

	pb "github.com/0glabs/0g-monitor/proto/da-client"
	"github.com/Conflux-Chain/go-conflux-util/health"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type DaClient struct {
	// discordId        string
	// validatorAddress string
	conn   *grpc.ClientConn
	client pb.DisperserClient
	name   string

	ip       string
	health   health.TimedCounter
	rpcError string // last rpc error message
}

func MustNewDaClient(name, ip string) *DaClient {
	conn, err := grpc.NewClient(ip, []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}...)
	if err != nil {
		return nil
	}

	c := pb.NewDisperserClient(conn)
	// 	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	// 	defer cancel()
	// 	_, err = c.GetStatus(ctx, &pb.Empty{})
	// }

	return &DaClient{
		// discordId:        discordId,
		// validatorAddress: validatorAddress,
		conn:   conn,
		client: c,
		name:   name,
		ip:     ip,
	}
}

func (daClient *DaClient) Close() {
	daClient.conn.Close()
}

func (daClient *DaClient) CheckStatusSilence(config health.TimedCounterConfig) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()
	reply, err := daClient.client.GetBlobStatus(
		ctx,
		&pb.BlobStatusRequest{RequestId: []byte("a1e97acdaf863a0be73ada65895145a6ef6b5a9d332c2ee40aca2018b34d40dd-313732333130303737373830343536323933322fe3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855")},
	)
	if err != nil {
		logrus.WithError(err).WithField("da client", daClient.name).Debug("Failed to query blob status")

		daClient.rpcError = err.Error()
		unhealthy, unrecovered, elapsed := daClient.health.OnFailure(config)

		// report unhealthy
		if unhealthy {
			logrus.WithFields(logrus.Fields{
				"node":    daClient.name,
				"elapsed": prettyElapsed(elapsed),
				"error":   daClient.rpcError,
			}).Error("DA client RPC became unhealthy")
		}

		// remind unhealthy
		if unrecovered {
			logrus.WithFields(logrus.Fields{
				"node":     daClient.name,
				"elapsed":  prettyElapsed(elapsed),
				"rpcError": daClient.rpcError,
			}).Error("DA client RPC not recovered yet")
		}

	} else {
		if reply.Status != pb.BlobStatus_FINALIZED {
			logrus.WithFields(logrus.Fields{
				"node":        daClient.name,
				"statusError": reply.Status,
			}).Warn("DA client blob status is not expected")
		} else {
			daClient.rpcError = ""
			if recovered, elapsed := daClient.health.OnSuccess(config); recovered {
				logrus.WithFields(logrus.Fields{
					"node":    daClient.name,
					"elapsed": prettyElapsed(elapsed),
				}).Warn("DA client RPC is healthy now")
			}
		}
	}

	// upsertQuery := `
	//     INSERT INTO user_da_client_status (ip, discord_id, address, status)
	//     VALUES (?, ?, ?, ?)
	//     ON DUPLICATE KEY UPDATE
	//     status = VALUES(status)
	// `

	// conn, err := grpc.NewClient(daClient.ip, []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}...)
	// if err == nil {
	// 	c := pb.NewDisperserClient(conn)
	// 	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	// 	defer cancel()
	// 	_, err = c.GetStatus(ctx, &pb.Empty{})
	// }
	// defer conn.Close()

	// if err != nil {
	// 	logrus.WithFields(logrus.Fields{
	// 		"address": daClient.validatorAddress,
	// 		"ip":      daClient.ip,
	// 	}).WithError(err).Info("Da client connection failed")

	// 	daClient.health.OnFailure(config)
	// 	_, err = db.Exec(upsertQuery, daClient.ip, daClient.discordId, daClient.validatorAddress, NodeDisconnected)
	// 	if err != nil {
	// 		logrus.WithFields(logrus.Fields{
	// 			"ip": daClient.ip,
	// 		}).Warn("Failed to update da client status in db")
	// 	}
	// } else {
	// 	logrus.WithFields(logrus.Fields{
	// 		"address": daClient.validatorAddress,
	// 		"ip":      daClient.ip,
	// 	}).Info("Da client connection succeeded")

	// 	_, err = db.Exec(upsertQuery, daClient.ip, daClient.discordId, daClient.validatorAddress, NodeConnected)
	// 	if err != nil {
	// 		logrus.WithFields(logrus.Fields{
	// 			"ip": daClient.ip,
	// 		}).Warn("Failed to update da client status in db")
	// 	}
	// }
}

func prettyElapsed(elapsed time.Duration) string {
	return fmt.Sprint(elapsed.Truncate(time.Second))
}
