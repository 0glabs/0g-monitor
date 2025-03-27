package blockchain

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/sirupsen/logrus"
)

func EthGetLatestBlockInfo(url string) (*BlockInfo, error) {
	reqBody := map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  "eth_getBlockByNumber",
		"params":  []interface{}{"latest", false},
		"id":      1,
	}

	// Encode the request body to JSON
	reqBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	// Send the HTTP POST request
	client, err := createClient("eth", url)
	if err != nil {
		return nil, err
	}
	var respBody map[string]interface{}
	resp, err := client.R().SetResult(&respBody).SetHeader("Content-Type", "application/json").SetBody(string(reqBytes)).Post(url)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode() != 200 {
		logrus.WithError(err).WithField("url", url).WithField("status_code", resp.StatusCode()).Info("get latest block info via ethereum rpc failed")
		return nil, fmt.Errorf("get latest block info via ethereum rpc failed, status code: %d", resp.StatusCode())
	}

	if logrus.IsLevelEnabled(logrus.DebugLevel) {
		jsonStr, _ := json.Marshal(respBody)
		logrus.WithFields(logrus.Fields{
			"url":      url,
			"response": fmt.Sprintf("%+v", string(jsonStr)),
		}).Debug("response of ethereum rpc eth_getBlockByNumber: ")
	}

	if v, exists := respBody["result"]; !exists || v == nil || len(v.(map[string]interface{})) == 0 {
		jsonStr, _ := json.Marshal(respBody)
		logrus.WithError(err).WithField("url", url).WithField("response", fmt.Sprintf("%+v", string(jsonStr))).Info("get latest block info via ethereum rpc failed")
		return nil, fmt.Errorf("get latest block info via ethereum rpc failed, data invalid ")
	}

	// Get the block height from the response
	blockNumber := respBody["result"].(map[string]interface{})["number"].(string)
	timestamp := respBody["result"].(map[string]interface{})["timestamp"].(string)
	hsah := respBody["result"].(map[string]interface{})["hash"].(string)

	resBlockNumber, err := strconv.ParseUint(blockNumber, 0, 64)
	if err != nil {
		return nil, err
	}

	resTimestamp, err := strconv.ParseUint(timestamp, 0, 64)
	if err != nil {
		return nil, err
	}

	txs := respBody["result"].(map[string]interface{})["transactions"].([]interface{})
	hashList := make([]string, 0, len(txs))
	for _, tx := range txs {
		hashList = append(hashList, tx.(string))
	}

	return &BlockInfo{
		Height:    resBlockNumber,
		Timestamp: resTimestamp,
		Hash:      hsah,
		TxHashes:  hashList,
	}, nil
}

func EthFetchTxReceiptStatus(url string, txHash string) (uint64, error) {
	reqBody := map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  "eth_getTransactionReceipt",
		"params":  []interface{}{txHash},
		"id":      1,
	}

	// Encode the request body to JSON
	reqBytes, err := json.Marshal(reqBody)
	if err != nil {
		return 0, err
	}

	// Send the HTTP POST request
	client, err := createClient("eth", url)
	if err != nil {
		return 0, err
	}
	var respBody map[string]interface{}
	resp, err := client.R().SetResult(&respBody).SetHeader("Content-Type", "application/json").SetBody(string(reqBytes)).Post(url)
	if err != nil {
		return 0, err
	}

	if resp.StatusCode() != 200 {
		logrus.WithError(err).WithField("url", url).WithField("status_code", resp.StatusCode()).Info("fetch tx receipt status via ethereum rpc failed")
		return 0, fmt.Errorf("fetch tx receipt status via ethereum rpc failed, status code: %d", resp.StatusCode())
	}

	// Get the block height from the response
	statusStr := respBody["result"].(map[string]interface{})["status"].(string)

	statusCode, err := strconv.ParseUint(statusStr, 0, 64)
	if err != nil {
		return 0, err
	}

	return statusCode, nil
}

func EthFetchBlockReceiptStatus(url string, height uint64) (map[string]bool, error) {
	reqBody := map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  "eth_getBlockReceipts",
		"params":  []interface{}{fmt.Sprintf("0x%x", height)},
		"id":      1,
	}

	// Encode the request body to JSON
	reqBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	// Send the HTTP POST request
	client, err := createClient("eth", url)
	if err != nil {
		return nil, err
	}
	var respBody map[string]interface{}
	resp, err := client.R().SetResult(&respBody).SetHeader("Content-Type", "application/json").SetBody(string(reqBytes)).Post(url)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode() != 200 {
		logrus.WithError(err).WithField("url", url).WithField("status_code", resp.StatusCode()).Info("fetch block receipt status via ethereum rpc failed")
		return nil, fmt.Errorf("fetch block receipt status via ethereum rpc failed, status code: %d", resp.StatusCode())
	}

	if logrus.IsLevelEnabled(logrus.DebugLevel) {
		jsonStr, _ := json.Marshal(respBody)
		logrus.WithFields(logrus.Fields{
			"url":      url,
			"response": fmt.Sprintf("%+v", string(jsonStr)),
		}).Debug("response of ethereum rpc eth_getBlockReceipts: ")
	}

	result, ok := respBody["result"].([]interface{})
	if !ok {
		println(fmt.Sprintf("%v", respBody))
		return nil, fmt.Errorf("invalid response of ethereum rpc eth_getBlockReceipts")
	}

	statusMap := make(map[string]bool, len(result))

	for _, txr := range result {
		r := txr.(map[string]interface{})
		statusStr := r["status"].(string)
		txHash := r["transactionHash"].(string)
		statusMap[txHash] = statusStr == "0x1"
	}

	return statusMap, nil
}
