package blockchain

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/Conflux-Chain/go-conflux-util/health"
	"github.com/Conflux-Chain/go-conflux-util/metrics"
	"github.com/go-resty/resty/v2"
	"github.com/sirupsen/logrus"
)

type HeightReportConfig struct {
	health.TimedCounterConfig

	MaxGap uint64 `default:"30"`
}

type Node struct {
	name string
	url  string

	height uint64 // fullnode block number

	rpcHealth health.TimedCounter
	rpcError  string // last rpc error message

	heightHealth health.TimedCounter
}

func MustNewNode(name, urlstr string) *Node {
	url, _ := url.Parse(urlstr)
	// url.Path = "status"

	metrics.GetOrRegisterGauge("monitor/blockchain/rpc/height/unhealth/%v", name).Update(0)
	metrics.GetOrRegisterGauge("monitor/blockchain/height/behind/%v", name).Update(0)

	return &Node{
		name: name,
		url:  url.String(),
	}
}

func getBlockNumber(url string) (uint64, error) {
	reqBody := map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  "eth_blockNumber",
		"params":  []interface{}{},
		"id":      1,
	}

	// Encode the request body to JSON
	reqBytes, err := json.Marshal(reqBody)
	if err != nil {
		return 0, err
	}

	// Send the HTTP POST request
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(reqBytes))
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	// Decode the JSON response
	var respBody map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&respBody); err != nil {
		return 0, err
	}

	// Get the block height from the response
	blockNumber := respBody["result"].(string)
	return strconv.ParseUint(blockNumber, 0, 64)
}

func fetchHeight(url string) (uint64, error) {

	var result map[string]interface{}
	client := resty.New()
	resp, err := client.R().SetResult(&result).Get(url)
	if err != nil || resp.IsError() {
		return 0, err
	}
	height, err := strconv.Atoi(result["result"].(map[string]interface{})["sync_info"].(map[string]interface{})["latest_block_height"].(string))
	if err != nil {
		return 0, err
	}
	return uint64(height), nil
}

func (node *Node) UpdateHeight(config health.TimedCounterConfig) {
	start := time.Now()
	height, err := getBlockNumber(node.url)
	elapsed := time.Since(start).Nanoseconds()
	metrics.GetOrRegisterHistogram("monitor/blockchain/rpc/height/latency/%v", node.name).Update(elapsed)
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

			metrics.GetOrRegisterGauge("monitor/blockchain/rpc/height/unhealth/%v", node.name).Update(1)
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
		metrics.GetOrRegisterHistogram("monitor/blockchain/rpc/height/latency/success/%v", node.name).Update(elapsed)

		// check reorg
		if height < node.height {
			logrus.WithFields(logrus.Fields{
				"node":     node.name,
				"old":      node.height,
				"new":      height,
				"reverted": node.height - height,
			}).Warn("Block reorg detected")
		}

		node.height = height
		node.rpcError = ""

		// report on recovered
		if recovered, elapsed := node.rpcHealth.OnSuccess(config); recovered {
			logrus.WithFields(logrus.Fields{
				"node":    node.name,
				"elapsed": prettyElapsed(elapsed),
			}).Warn("Node RPC is healthy now")

			metrics.GetOrRegisterGauge("monitor/blockchain/rpc/height/unhealth/%v", node.name).Update(0)
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

			metrics.GetOrRegisterGauge("monitor/blockchain/height/behind/%v", node.name).Update(0)
		}
	} else {
		unhealthy, unrecovered, elapsed := node.heightHealth.OnFailure(config.TimedCounterConfig)

		if unhealthy {
			logrus.WithFields(logrus.Fields{
				"node":    node.name,
				"elapsed": prettyElapsed(elapsed),
				"behind":  behind,
			}).Error("Node block height became unhealthy")

			metrics.GetOrRegisterGauge("monitor/blockchain/height/behind/%v", node.name).Update(1)
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
