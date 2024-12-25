package blockchain

import (
	"fmt"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/0glabs/0g-monitor/utils"
	"github.com/Conflux-Chain/go-conflux-util/metrics"
	"github.com/Conflux-Chain/go-conflux-util/viper"
	"github.com/go-gota/gota/dataframe"
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

	var userNodes []*Validator
	if config.Mode != "localtest" {
		userNodes = loadUserNodeInfo(url)
	}

	mempool := MustNewMempool(config.CometbftRPC)

	blockTxCntRecord = make(map[uint64]int, config.BlockTxCntLimit)
	blockFailedTxCntRecord = make(map[uint64]int, config.BlockTxCntLimit)

	// Monitor once immediately
	monitorOnce(&config, nodes, validators, userNodes, mempool)

	// Monitor node status periodically
	ticker := time.NewTicker(config.Interval)
	defer ticker.Stop()

	for range ticker.C {
		monitorOnce(&config, nodes, validators, userNodes, mempool)
	}
}

func createMetricsForChain() {
	metrics.GetOrRegisterHistogram(validatorActiveCountPattern).Update(0)
	metrics.GetOrRegisterGauge(validatorActiveCountUnhealthPattern).Update(0)

	metrics.GetOrRegisterGauge(failedTxCountUnhealthPattern).Update(0)
	metrics.GetOrRegisterHistogram(failedTxCountPattern).Update(0)
}

func monitorOnce(config *Config, nodes []*Node, validators []*Validator, userNodes []*Validator, mempool *Mempool) {
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
	}

	// update validator status
	monitorValidator(config, validators)

	// update user node status
	for _, v := range userNodes {
		v.CheckStatusSilence()
	}

	monitorMempool(config, mempool)
}

func monitorTxFailures(config *Config, nodes []*Node, txInfo *BlockTxInfo) {
	if txInfo != nil {
		if len(txInfo.TxHashes) > 0 {
			swg := utils.NewSizedWaitGroup(len(nodes))

			for i := range txInfo.TxHashes {
				swg.Add()
				go func(node *Node, hash string) {
					defer swg.Done()
					isSuccess, err := node.FetchTxReceiptStatus(config.NodeHeightReport.TimedCounterConfig, hash)
					if err == nil {
						if !isSuccess {
							defer blockFailedTxCntLock.Unlock()
							blockFailedTxCntLock.Lock()

							blockFailedTxCntRecord[node.currentBlockInfo.Height] += 1
						}
					}
				}(nodes[i%len(nodes)], txInfo.TxHashes[i])
			}
			swg.Wait()
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
		if failedTxCnt > 0 && failedTxCnt*100/totalTxCnt > config.FailedTxCntAlarmThreshold {
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

	if 100*float64(activeValidatorCount)/float64(len(validators)) <= 67 {
		metrics.GetOrRegisterGauge(validatorActiveCountUnhealthPattern).Update(1)
	} else {
		metrics.GetOrRegisterGauge(validatorActiveCountUnhealthPattern).Update(0)
	}

	logrus.WithField("active", activeValidatorCount).WithField("jailed", jailedCnt).Debug("Validators status report")
}

func monitorMempool(config *Config, mempool *Mempool) {
	unconfirmedTxCnt := mempool.UpdateUncommitTxCnt(config.MempoolReport.TimedCounterConfig)

	metrics.GetOrRegisterHistogram(mempoolUncommitTxCntPattern).Update(int64(unconfirmedTxCnt))
	if float64(unconfirmedTxCnt*100)/float64(config.MempoolReport.PoolSize)-float64(config.MempoolReport.AlarmThreshold) > 0 {
		metrics.GetOrRegisterGauge(mempoolHighLoadPattern).Update(1)
	} else {
		metrics.GetOrRegisterGauge(mempoolHighLoadPattern).Update(0)
	}
}

func loadUserNodeInfo(cosmosRpcUrl *url.URL) []*Validator {
	var userNodes []*Validator

	f, err := os.Open(ValidatorFile)
	if err != nil {
		fmt.Println("Error opening csv:", err)
		return userNodes
	}
	defer f.Close()
	df := dataframe.ReadCSV(f)

	for i := 0; i < df.Nrow(); i++ {
		discordId := df.Subset(i).Col("discord_id").Elem(0).String()

		validatorAddress := df.Subset(i).Col("validator_address").Elem(0).String()
		rpc := df.Subset(i).Col("validator_rpc").Elem(0).String()
		ips := strings.Split(rpc, ",")
		for _, ip := range ips {
			ip = strings.TrimSpace(ip)
			logrus.WithField("discord_id", discordId).WithField("ip", ip).Debug("Start to monitor user validator node")

			currNode := MustNewValidator(cosmosRpcUrl, validatorAddress, ip)
			if currNode != nil {
				userNodes = append(userNodes, currNode)
			}
		}
	}

	return userNodes
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
