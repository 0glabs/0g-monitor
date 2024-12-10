package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"
)

func PrettyElapsed(elapsed time.Duration) string {
	return fmt.Sprint(elapsed.Truncate(time.Second))
}


func GetBlockNumber(url string) (uint64, error) {
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