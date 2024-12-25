package blockchain

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strconv"
)

func EthGetLatestBlockInfo(url string) (BlockInfo, error) {
	reqBody := map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  "eth_getBlockByNumber",
		"params":  []interface{}{"latest", false},
		"id":      1,
	}

	// Encode the request body to JSON
	reqBytes, err := json.Marshal(reqBody)
	if err != nil {
		return BlockInfo{}, err
	}

	// Send the HTTP POST request
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(reqBytes))
	if err != nil {
		return BlockInfo{}, err
	}
	defer resp.Body.Close()

	// Decode the JSON response
	var respBody map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&respBody); err != nil {
		return BlockInfo{}, err
	}

	// Get the block height from the response
	blockNumber := respBody["result"].(map[string]interface{})["number"].(string)
	timestamp := respBody["result"].(map[string]interface{})["timestamp"].(string)
	hsah := respBody["result"].(map[string]interface{})["hash"].(string)

	resBlockNumber, err := strconv.ParseUint(blockNumber, 0, 64)
	if err != nil {
		return BlockInfo{}, err
	}

	resTimestamp, err := strconv.ParseUint(timestamp, 0, 64)
	if err != nil {
		return BlockInfo{}, err
	}

	txs := respBody["result"].(map[string]interface{})["transactions"].([]interface{})
	hashList := make([]string, 0, len(txs))
	for _, tx := range txs {
		hashList = append(hashList, tx.(string))
	}

	return BlockInfo{
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
	statusStr := respBody["result"].(map[string]interface{})["status"].(string)

	statusCode, err := strconv.ParseUint(statusStr, 0, 64)
	if err != nil {
		return 0, err
	}

	return statusCode, nil
}
