package blockchain

import (
	"fmt"
	"time"

	"github.com/Conflux-Chain/go-conflux-util/health"
	"github.com/openweb3/web3go"
	"github.com/sirupsen/logrus"
)

type HeightReportConfig struct {
	health.TimedCounterConfig

	MaxGap uint64 `default:"30"`
}

type Node struct {
	*web3go.Client

	name string
	url  string

	height         uint64 // fullnode block number
	updateTime     time.Time
	livenessHealth health.TimedCounter

	rpcHealth health.TimedCounter
	rpcError  string // last rpc error message

	heightHealth health.TimedCounter
}

func MustNewNode(name, url string) *Node {
	return &Node{
		Client:     web3go.MustNewClient(url),
		name:       name,
		url:        url,
		updateTime: time.Now(),
	}
}

func (node *Node) UpdateHeight(config health.TimedCounterConfig) {
	bn, err := node.Eth.BlockNumber()
	if err != nil {
		logrus.WithError(err).WithField("node", node.name).Debug("Failed to query block number")

		node.rpcError = err.Error()
		unhealthy, unrecovered, elapsed := node.rpcHealth.OnFailure(config)

		// report unhealthy
		if unhealthy {
			logrus.WithFields(logrus.Fields{
				"node":    node.name,
				"elapsed": prettyElapsed(elapsed),
				"error":   node.rpcError,
			}).Error("Node RPC became unhealthy")
		}

		// remind unhealthy
		if unrecovered {
			logrus.WithFields(logrus.Fields{
				"node":     node.name,
				"elapsed":  prettyElapsed(elapsed),
				"rpcError": node.rpcError,
			}).Error("Node RPC not recovered yet")
		}
	} else {
		// check reorg
		if bn.Uint64() < node.height {
			logrus.WithFields(logrus.Fields{
				"node":     node.name,
				"old":      node.height,
				"new":      bn.Uint64(),
				"reverted": node.height - bn.Uint64(),
			}).Warn("Block reorg detected")
		}

		now := time.Now()
		if node.height != bn.Uint64() {
			node.height = bn.Uint64()
			node.updateTime = now

			if recovered, elapsed := node.livenessHealth.OnSuccess(config); recovered {
				logrus.WithFields(logrus.Fields{
					"node":    node.name,
					"elapsed": prettyElapsed(elapsed),
				}).Warn("Node is making progress now")
			}
		} else {
			elapsed := now.Sub(node.updateTime)
			if elapsed > 180*time.Second {
				unhealthy, unrecovered, elapsed := node.livenessHealth.OnFailure(config)

				// report unhealthy
				if unhealthy {
					logrus.WithFields(logrus.Fields{
						"node":    node.name,
						"elapsed": prettyElapsed(elapsed),
					}).Error("Node is not making progress")
				}

				// remind unhealthy
				if unrecovered {
					logrus.WithFields(logrus.Fields{
						"node":    node.name,
						"elapsed": prettyElapsed(elapsed),
					}).Error("Node is not making progress")
				}
			}
		}

		node.rpcError = ""

		// report on recovered
		if recovered, elapsed := node.rpcHealth.OnSuccess(config); recovered {
			logrus.WithFields(logrus.Fields{
				"node":    node.name,
				"elapsed": prettyElapsed(elapsed),
			}).Warn("Node RPC is healthy now")
		}
	}
}

func FindMaxBlockHeight(nodes []*Node) uint64 {
	max := uint64(0)

	for _, v := range nodes {
		if v.rpcHealth.IsSuccess() && max < v.height {
			max = v.height
		}
	}

	return max
}

func (node *Node) CheckHeight(config *HeightReportConfig, target uint64) {
	// ignore on rpc error
	if !node.rpcHealth.IsSuccess() {
		return
	}

	// number of blocks fall behind
	var behind uint64
	if node.height < target {
		behind = target - node.height
	}

	if behind <= config.MaxGap {
		if recovered, elapsed := node.heightHealth.OnSuccess(config.TimedCounterConfig); recovered {
			logrus.WithFields(logrus.Fields{
				"node":    node.name,
				"elapsed": prettyElapsed(elapsed),
				"behind":  behind,
			}).Warn("Node block height is healthy now")
		}
	} else {
		unhealthy, unrecovered, elapsed := node.heightHealth.OnFailure(config.TimedCounterConfig)

		if unhealthy {
			logrus.WithFields(logrus.Fields{
				"node":    node.name,
				"elapsed": prettyElapsed(elapsed),
				"behind":  behind,
			}).Error("Node block height became unhealthy")
		}

		if unrecovered {
			logrus.WithFields(logrus.Fields{
				"node":    node.name,
				"elapsed": prettyElapsed(elapsed),
				"behind":  behind,
			}).Error("Node block height not recovered yet")
		}
	}
}

func prettyElapsed(elapsed time.Duration) string {
	return fmt.Sprint(elapsed.Truncate(time.Second))
}
