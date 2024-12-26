package blockchain

import (
	"fmt"
	"net/url"
	"sync"
	"time"

	"github.com/Conflux-Chain/go-conflux-util/metrics"
	"github.com/Conflux-Chain/go-conflux-util/viper"
	"github.com/sirupsen/logrus"
)

const ValidatorFile = "data/validator_rpcs.csv"

var (
	blockTxCntRecord       map[uint64]int
	blockFailedTxCntRecord map[uint64]int
	blockFailedTxCntLock   sync.RWMutex
)

func MustMonitorFromViper() {
	var config Config
	viper.MustUnmarshalKey("blockchain", &config)
	Monitor(config)
}

func Monitor(config Config) {
	logrus.WithFields(logrus.Fields{
		"nodes":      len(config.Nodes),
		"validators": len(config.Validators),
	}).Info("Start to monitor blockchain")

	createMetricsForChain()

	// Connect to all fullnodes
	var nodes []*Node
	for name, url := range config.Nodes {
		logrus.WithField("name", name).WithField("url", url).Debug("Start to monitor fullnode")
		nodes = append(nodes, MustNewNode(name, url))
	}

	var validators []*Validator
	url, _ := url.Parse(config.CosmosRest)
	for name, address := range config.Validators {
		logrus.WithField("name", name).WithField("address", address).Debug("Start to monitor validator")
		validators = append(validators, MustNewValidator(url, name, address))
	}

	consensus := MustNewConsensus(config.CometbftRPC)

	blockTxCntRecord = make(map[uint64]int, config.BlockTxCntLimit)
	blockFailedTxCntRecord = make(map[uint64]int, config.BlockTxCntLimit)

	// Monitor once immediately
	monitorOnce(&config, nodes, validators, consensus)

	// Monitor node status periodically
	ticker := time.NewTicker(config.Interval)
	defer ticker.Stop()

	for range ticker.C {
		monitorOnce(&config, nodes, validators, consensus)
	}
}

func createMetricsForChain() {
	metrics.GetOrRegisterHistogram(validatorActiveCountPattern).Update(0)
	metrics.GetOrRegisterGauge(validatorActiveCountUnhealthPattern).Update(0)

	metrics.GetOrRegisterGauge(failedTxCountUnhealthPattern).Update(0)
	metrics.GetOrRegisterHistogram(failedTxCountPattern).Update(0)

	metrics.GetOrRegisterHistogram(blockTxCountPattern).Update(0)
}

func monitorOnce(config *Config, nodes []*Node, validators []*Validator, consensus *Consensus) {
	blockSwitched := false
	var blockTxInfo *BlockTxInfo
	for _, v := range nodes {
		v.UpdateHeight(config.AvailabilityReport)
		// generate block tx info for new block
		if _, existed := blockTxCntRecord[v.currentBlockInfo.Height]; !existed {
			blockTxCntRecord[v.currentBlockInfo.Height] = len(v.currentBlockInfo.TxHashes)
			blockTxInfo = &BlockTxInfo{
				Height:   v.currentBlockInfo.Height,
				TxHashes: v.currentBlockInfo.TxHashes,
			}

			if !blockSwitched {
				blockSwitched = true
			}
		}
	}

	max := FindMaxBlockHeight(nodes)
	if max == 0 {
		return
	}
	defaultBlockchainHeightHealth.Update(config.BlockchainHeightReport, max)

	logrus.WithField("height", max).Debug("Fullnode status report")

	for _, v := range nodes {
		v.CheckHeight(&config.NodeHeightReport, max)
	}

	// detect tx failures and detect fork
	if blockSwitched {
		monitorTxFailures(config, nodes, blockTxInfo)

		// detect chain fork
		recordor := make(map[uint64]string, 20)
		for _, v := range nodes {
			v.CheckFork(recordor)
		}

		monitorBlockValidator(config, consensus, blockTxInfo.Height)
	}

	// update validator status
	monitorValidator(config, validators)

	monitorMempool(config, consensus)
}

func countFailedTx(statusMap map[string]bool) int {
	failedCnt := 0
	for _, status := range statusMap {
		if !status {
			failedCnt++
		}
	}
	return failedCnt
}

func monitorTxFailures(config *Config, nodes []*Node, txInfo *BlockTxInfo) {
	if txInfo != nil {
		blockTxCnt := len(txInfo.TxHashes)
		metrics.GetOrRegisterHistogram(blockTxCountPattern).Update(int64(blockTxCnt))

		logrus.Debug(fmt.Sprintf("Block (%d) tx count: %d", txInfo.Height, blockTxCnt))

		if blockTxCnt > 0 {
			index := int(time.Now().UnixNano() % int64(len(nodes)))
			statusMap, err := nodes[index].FetchBlockReceiptStatus(config.NodeHeightReport.TimedCounterConfig, txInfo.Height)
			if err != nil {
				return
			}

			blockFailedTxCntRecord[txInfo.Height] = countFailedTx(statusMap)
		}

		totalTxCnt, failedTxCnt := 0, 0
		for i := 0; i < config.BlockTxCntLimit; i++ {
			if uint64(i) > txInfo.Height {
				break
			}
			targetHeight := txInfo.Height - uint64(i)
			if cnt, existed := blockTxCntRecord[targetHeight]; existed {
				totalTxCnt += cnt
				failedTxCnt += blockFailedTxCntRecord[targetHeight]
			} else {
				break
			}
		}

		metrics.GetOrRegisterHistogram(failedTxCountPattern).Update(int64(failedTxCnt))
		percentage := float64(failedTxCnt*100) / float64(totalTxCnt)
		if failedTxCnt > 0 && percentage-float64(config.FailedTxCntAlarmThreshold) > 0 {
			metrics.GetOrRegisterGauge(failedTxCountUnhealthPattern).Update(1)
		} else {
			metrics.GetOrRegisterGauge(failedTxCountUnhealthPattern).Update(0)
		}

		if len(blockTxCntRecord) > config.BlockTxCntLimit {
			if uint64(config.BlockTxCntLimit) <= nodes[0].currentBlockInfo.Height {
				startHeight := nodes[0].currentBlockInfo.Height - uint64(config.BlockTxCntLimit)
				for k := range blockTxCntRecord {
					if k < startHeight {
						delete(blockTxCntRecord, k)
						delete(blockFailedTxCntRecord, k)
					}
				}
			}
		}
	} else {
		metrics.GetOrRegisterHistogram(blockTxCountPattern).Update(0)
	}
}

func monitorValidator(config *Config, validators []*Validator) {
	jailedCnt := 0
	for _, v := range validators {
		v.Update(config.ValidatorReport)
		if v.jailed {
			jailedCnt++
		}
	}

	activeValidatorCount := len(validators) - jailedCnt
	metrics.GetOrRegisterHistogram(validatorActiveCountPattern).Update(int64(activeValidatorCount))
	percentage := 100 * float64(activeValidatorCount) / float64(len(validators))
	if percentage-float64(67) >= 0 {
		metrics.GetOrRegisterGauge(validatorActiveCountUnhealthPattern).Update(0)
	} else {
		metrics.GetOrRegisterGauge(validatorActiveCountUnhealthPattern).Update(1)
	}

	logrus.WithField("active", activeValidatorCount).WithField("jailed", jailedCnt).Debug("Validators status report")
}

func monitorMempool(config *Config, consensus *Consensus) {
	unconfirmedTxCnt := consensus.UpdateUncommitTxCnt(config.MempoolReport.TimedCounterConfig)

	metrics.GetOrRegisterHistogram(mempoolUncommitTxCntPattern).Update(int64(unconfirmedTxCnt))
	percentage := float64(unconfirmedTxCnt*100) / float64(config.MempoolReport.PoolSize)
	metrics.GetOrRegisterGauge(mempoolLoadPattern).Update(int64(percentage))
	logrus.Debug("Mempool status report: unconfirmed tx count = ", unconfirmedTxCnt, ", percentage = ", percentage)
	if percentage-float64(config.MempoolReport.AlarmThreshold) > 0 {
		metrics.GetOrRegisterGauge(mempoolHighLoadPattern).Update(1)
	} else {
		metrics.GetOrRegisterGauge(mempoolHighLoadPattern).Update(0)
	}
}

func monitorBlockValidator(config *Config, consensus *Consensus, blockHeight uint64) {
	blkValidatorCnt := consensus.GetBlockValidatorCnt(config.MempoolReport.TimedCounterConfig, blockHeight)
	logrus.Debug(fmt.Sprintf("count of validator who signed block %d = %d", blockHeight, blkValidatorCnt))
	metrics.GetOrRegisterHistogram(blockValidatorCountPattern).Update(int64(blkValidatorCnt))
}

func FindMaxBlockHeight(nodes []*Node) uint64 {
	max := uint64(0)

	for _, v := range nodes {
		if v.rpcHealth.IsSuccess() && max < v.currentBlockInfo.Height {
			max = v.currentBlockInfo.Height
		}
	}

	return max
}
