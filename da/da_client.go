package da

import (
	"context"
	"crypto/rand"
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

	requestId []byte
	counter   uint
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
		conn:      conn,
		client:    c,
		name:      name,
		ip:        ip,
		requestId: nil,
		counter:   0,
	}
}

func (daClient *DaClient) Close() {
	daClient.conn.Close()
}

func (daClient *DaClient) CheckStatusSilence(config health.TimedCounterConfig) {
	var err error
	var statusReply *pb.BlobStatusReply

	if daClient.requestId == nil || daClient.counter >= 20 {
		statusReply, err = daClient.DisperseNewBlob()
		if err == nil {
			daClient.counter = 0
		}
	} else {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
		defer cancel()

		statusReply, err = daClient.client.GetBlobStatus(
			ctx,
			&pb.BlobStatusRequest{
				RequestId: daClient.requestId,
			},
		)

		daClient.counter += 1
	}

	if err != nil {
		logrus.WithError(err).WithField("da client", daClient.name).Error("Failed to query da client")

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
		if statusReply.Status != pb.BlobStatus_FINALIZED {
			logrus.WithFields(logrus.Fields{
				"node":        daClient.name,
				"statusError": statusReply.Status,
			}).Warn("DA client blob status is not expected")
		} else {
			daClient.rpcError = ""
			if recovered, elapsed := daClient.health.OnSuccess(config); recovered {
				logrus.WithFields(logrus.Fields{
					"node":    daClient.name,
					"elapsed": prettyElapsed(elapsed),
				}).Warn("DA client RPC is healthy now")
			}

			logrus.WithField("request id", daClient.requestId).Debug("Blob status is expected")
		}
	}
}

func (daClient *DaClient) DisperseNewBlob() (*pb.BlobStatusReply, error) {
	logrus.Debug("Disperse new blob")
	byteArray := make([]byte, 100)

	_, err := rand.Read(byteArray)
	if err != nil {
		return nil, fmt.Errorf("an error occurred when rand bytes: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*180)
	defer cancel()

	blobReply, err := daClient.client.DisperseBlob(ctx, &pb.DisperseBlobRequest{Data: byteArray})
	if err != nil {
		return nil, err
	}

	daClient.requestId = blobReply.GetRequestId()
	retryCount := 0
	for {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
		defer cancel()

		statusReply, err := daClient.client.GetBlobStatus(ctx, &pb.BlobStatusRequest{
			RequestId: daClient.requestId,
		})

		if err != nil {
			return nil, err
		}

		status := statusReply.GetStatus()
		if status == pb.BlobStatus_FINALIZED {
			return statusReply, nil
		} else if status == pb.BlobStatus_FAILED {
			return nil, fmt.Errorf("the status of the new disperse blob is unexpected")
		}

		retryCount++
		if retryCount > 60 {
			return nil, fmt.Errorf("failed to get the status of the new disperse blob; retry limit reached")
		}

		time.Sleep(10 * time.Second)
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

func prettyElapsed(elapsed time.Duration) string {
	return fmt.Sprint(elapsed.Truncate(time.Second))
}
