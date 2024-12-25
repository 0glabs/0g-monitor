package blockchain

import (
	"github.com/go-resty/resty/v2"
)

func IsValidatorJailed(url string) (bool, error) {
	client := resty.New()
	var result map[string]interface{}
	resp, err := client.R().SetResult(&result).Get(url)
	if err != nil {
		return false, err
	}

	if resp.StatusCode() != 200 {
		return false, ErrorNotSuccess
	}

	return result["validator"].(map[string]interface{})["jailed"].(bool), nil
}
