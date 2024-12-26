package blockchain

import (
	"net/url"
	"time"

	"github.com/0glabs/0g-monitor/utils"
	"github.com/Conflux-Chain/go-conflux-util/health"
	"github.com/Conflux-Chain/go-conflux-util/metrics"
	"github.com/sirupsen/logrus"
)

type HeightReportConfig struct {
	health.TimedCounterConfig

	MaxGap uint64 `default:"30"`
}

type Node struct {
	name string
	url  string

	currentBlockInfo BlockInfo

	rpcHealth health.TimedCounter
	rpcError  string // last rpc error message

	heightHealth health.TimedCounter

	lastBlockGap   uint64
	blockGapHealth health.TimedCounter

	ethRpcHealth health.TimedCounter
	ethRpcError  string // last rpc error message
}

func MustNewNode(name, urlstr string) *Node {
	url, _ := url.Parse(urlstr)

	createMetricsForNode(name)

	return &Node{
		name: name,
		url:  url.String(),
	}
}

func createMetricsForNode(name string) {
	metrics.GetOrRegisterHistogram(blockHeightBehindPattern, name).Update(0)
	metrics.GetOrRegisterGauge(blockHeightUnhealthPattern, name).Update(0)

	metrics.GetOrRegisterGauge(chainForkPattern, name).Update(0)

	metrics.GetOrRegisterHistogram(blockCollatedGapPattern, name).Update(0)
	metrics.GetOrRegisterGauge(blockCollatedGapUnhealthPattern, name).Update(0)
}

func (node *Node) UpdateHeight(config AvailabilityReport) {
	var info BlockInfo
	executeRequest(
		func() error {
			var err error
			info, err = EthGetLatestBlockInfo(node.url)
			if err != nil {
				logrus.WithError(err).WithField("node", node.name).Debug("Failed to query block number")
				return err
			}

			if info.Height < node.currentBlockInfo.Height {
				logrus.WithFields(logrus.Fields{
					"node":     node.name,
					"old":      node.currentBlockInfo.Height,
					"new":      info.Height,
					"reverted": node.currentBlockInfo.Height - info.Height,
				}).Warn("Block reorg detected")
			}

			if node.currentBlockInfo.Height != info.Height {
				latest := node.lastBlockGap
				node.lastBlockGap = info.Timestamp - node.currentBlockInfo.Timestamp

				if latest > 0 { // skip first report
					metrics.GetOrRegisterHistogram(blockCollatedGapPattern, node.name).Update(int64(node.lastBlockGap * uint64(time.Second)))

					if node.lastBlockGap > config.MaxGap {
						unhealthy, unrecovered, elapsed := node.blockGapHealth.OnFailure(config.TimedCounterConfig)

						if unhealthy {
							logrus.WithFields(logrus.Fields{
								"node":         node.name,
								"height":       node.currentBlockInfo.Height,
								"hash":         node.currentBlockInfo.Hash,
								"collated_gap": node.lastBlockGap,
								"elapsed":      utils.PrettyElapsed(elapsed),
							}).Error("Node block collated gap became unhealthy")
							metrics.GetOrRegisterGauge(blockCollatedGapUnhealthPattern, node.name).Update(1)
						}

						// remind unhealthy
						if unrecovered {
							logrus.WithFields(logrus.Fields{
								"node":         node.name,
								"elapsed":      utils.PrettyElapsed(elapsed),
								"height":       node.currentBlockInfo.Height,
								"hash":         node.currentBlockInfo.Hash,
								"collated_gap": node.lastBlockGap,
							}).Error("Node block collated gap not recovered yet")
						}
					} else {
						if recovered, elapsed := node.blockGapHealth.OnSuccess(config.TimedCounterConfig); recovered {
							logrus.WithFields(logrus.Fields{
								"node":    node.name,
								"elapsed": utils.PrettyElapsed(elapsed),
							}).Warn("Node block collated gap is healthy now")
							metrics.GetOrRegisterGauge(blockCollatedGapUnhealthPattern, node.name).Update(0)
						}
					}
				}
			}

			node.currentBlockInfo.Height = info.Height
			node.currentBlockInfo.Timestamp = info.Timestamp
			node.currentBlockInfo.Hash = info.Hash
			node.currentBlockInfo.TxHashes = info.TxHashes

			return nil
		},
		func(err error, unhealthy, unrecovered bool, elapsed time.Duration) {
			node.ethRpcError = err.Error()
			// report unhealthy
			if unhealthy {
				logrus.WithFields(logrus.Fields{
					"node":    node.name,
					"elapsed": utils.PrettyElapsed(elapsed),
					"error":   node.ethRpcError,
				}).Error("Node ethermint RPC became unhealthy")
			}

			// remind unhealthy
			if unrecovered {
				logrus.WithFields(logrus.Fields{
					"node":     node.name,
					"elapsed":  utils.PrettyElapsed(elapsed),
					"rpcError": node.ethRpcError,
				}).Error("Node ethermint RPC not recovered yet")
			}
		},
		func(recovered bool, elapsed time.Duration) {
			node.ethRpcError = ""

			// report on recovered
			if recovered {
				logrus.WithFields(logrus.Fields{
					"node":    node.name,
					"elapsed": utils.PrettyElapsed(elapsed),
				}).Warn("Node ethermint RPC is healthy now")
			}
		},
		nodeEthRpcLatencyPattern, nodeEthRpcUnhealthPattern, node.name,
		&node.ethRpcHealth,
		config.TimedCounterConfig,
	)
}

func (node *Node) CheckHeight(config *HeightReportConfig, target uint64) {
	// ignore on rpc error
	if !node.ethRpcHealth.IsSuccess() {
		return
	}

	// number of blocks fall behind
	var behind uint64
	if node.currentBlockInfo.Height < target {
		behind = target - node.currentBlockInfo.Height
	}

	metrics.GetOrRegisterHistogram(blockHeightBehindPattern, node.name).Update(int64(behind))
	if behind <= config.MaxGap {
		metrics.GetOrRegisterGauge(blockHeightUnhealthPattern, node.name).Update(0)

		if recovered, elapsed := node.heightHealth.OnSuccess(config.TimedCounterConfig); recovered {
			logrus.WithFields(logrus.Fields{
				"node":    node.name,
				"elapsed": utils.PrettyElapsed(elapsed),
				"behind":  behind,
			}).Warn("Node block height is healthy now")
			metrics.GetOrRegisterGauge(blockHeightUnhealthPattern, node.name).Update(0)
		}
	} else {
		unhealthy, unrecovered, elapsed := node.heightHealth.OnFailure(config.TimedCounterConfig)

		if unhealthy {
			logrus.WithFields(logrus.Fields{
				"node":    node.name,
				"elapsed": utils.PrettyElapsed(elapsed),
				"behind":  behind,
			}).Error("Node block height became unhealthy")
			metrics.GetOrRegisterGauge(blockHeightUnhealthPattern, node.name).Update(1)
		}

		if unrecovered {
			logrus.WithFields(logrus.Fields{
				"node":    node.name,
				"elapsed": utils.PrettyElapsed(elapsed),
				"behind":  behind,
			}).Error("Node block height not recovered yet")
		}
	}
}

func (node *Node) CheckFork(recordor map[uint64]string) {
	if existedHash, ok := recordor[node.currentBlockInfo.Height]; !ok {
		recordor[node.currentBlockInfo.Height] = node.currentBlockInfo.Hash
	} else {
		if node.currentBlockInfo.Hash != existedHash {
			// detected fork!
			logrus.WithFields(logrus.Fields{
				"node":         node.name,
				"height":       node.currentBlockInfo.Height,
				"hash":         node.currentBlockInfo.Hash,
				"existed_hash": existedHash,
			}).Error("Node block hash is different from existed one")
			metrics.GetOrRegisterGauge(chainForkPattern, node.name).Update(1)
		}
	}
}

func (node *Node) FetchTxReceiptStatus(config health.TimedCounterConfig, txHash string) (bool, error) {
	var statusCode uint64
	err := executeRequest(
		func() error {
			var err error
			statusCode, err = EthFetchTxReceiptStatus(node.url, txHash)
			if err != nil {
				return err
			}
			return nil
		},
		func(err error, unhealthy, unrecovered bool, elapsed time.Duration) {
			logrus.WithError(err).WithField("node", node.name).Debug("Failed to query tx receipt status")

			node.ethRpcError = err.Error()

			// log unhealthy
			if unhealthy {
				logrus.WithFields(logrus.Fields{
					"node":    node.name,
					"elapsed": utils.PrettyElapsed(elapsed),
					"error":   node.rpcError,
				}).Error("Node ethermint RPC became unhealthy")
			}

			// remind unhealthy
			if unrecovered {
				logrus.WithFields(logrus.Fields{
					"node":     node.name,
					"elapsed":  utils.PrettyElapsed(elapsed),
					"rpcError": node.rpcError,
				}).Error("Node ethermint RPC not recovered yet")
			}
		},
		func(recovered bool, elapsed time.Duration) {
			node.ethRpcError = ""

			// log on recovered
			if recovered {
				logrus.WithFields(logrus.Fields{
					"node":    node.name,
					"elapsed": utils.PrettyElapsed(elapsed),
				}).Warn("Node ethermint RPC is healthy now")
			}
		},
		nodeEthRpcLatencyPattern, nodeEthRpcUnhealthPattern, node.name,
		&node.ethRpcHealth,
		config,
	)

	if err != nil {
		return false, err
	}

	if statusCode == 1 {
		return true, nil
	}
	return false, nil
}

func (node *Node) FetchBlockReceiptStatus(config health.TimedCounterConfig, height uint64) (map[string]bool, error) {
	var statusMap map[string]bool
	err := executeRequest(
		func() error {
			var err error
			statusMap, err = EthFetchBlockReceiptStatus(node.url, height)
			if err != nil {
				return err
			}
			return nil
		},
		func(err error, unhealthy, unrecovered bool, elapsed time.Duration) {
			logrus.WithError(err).WithField("node", node.name).Debug("Failed to query tx receipt status")

			node.ethRpcError = err.Error()

			// log unhealthy
			if unhealthy {
				logrus.WithFields(logrus.Fields{
					"node":    node.name,
					"elapsed": utils.PrettyElapsed(elapsed),
					"error":   node.rpcError,
				}).Error("Node ethermint RPC became unhealthy")
			}

			// remind unhealthy
			if unrecovered {
				logrus.WithFields(logrus.Fields{
					"node":     node.name,
					"elapsed":  utils.PrettyElapsed(elapsed),
					"rpcError": node.rpcError,
				}).Error("Node ethermint RPC not recovered yet")
			}
		},
		func(recovered bool, elapsed time.Duration) {
			node.ethRpcError = ""

			// log on recovered
			if recovered {
				logrus.WithFields(logrus.Fields{
					"node":    node.name,
					"elapsed": utils.PrettyElapsed(elapsed),
				}).Warn("Node ethermint RPC is healthy now")
			}
		},
		nodeEthRpcLatencyPattern, nodeEthRpcUnhealthPattern, node.name,
		&node.ethRpcHealth,
		config,
	)

	if err != nil {
		return nil, err
	}

	return statusMap, nil
}
