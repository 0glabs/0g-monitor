package blockchain

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/Conflux-Chain/go-conflux-util/health"
	"github.com/go-resty/resty/v2"
	"github.com/openweb3/web3go"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type Validator struct {
	url     string
	name    string
	address string
	health  health.TimedCounter
}

func MustNewValidator(url *url.URL, name, address string, isCommunity bool) *Validator {
	validator, err := NewValidator(url, name, address, isCommunity)
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"name":    name,
			"address": address,
		}).WithError(err).Info("Failed to create validator node")
		return nil
	}

	return validator
}

func NewValidator(url *url.URL, name, address string, isCommunity bool) (*Validator, error) {
	address = strings.TrimSpace(address)
	if len(address) == 0 {
		return nil, errors.New("empty address")
	}
	if isCommunity {
		url.Path = "/cosmos/staking/v1beta1/validators/" + address
		return &Validator{
			url:     url.String(),
			name:    name,
			address: address,
		}, nil
	}

	url.Path = "/cosmos/staking/v1beta1/validators/" + address
	return &Validator{
		url:     url.String(),
		name:    name,
		address: address,
	}, nil
}

func (validator Validator) String() string {
	if len(validator.name) == 0 {
		return validator.address
	}

	return validator.name
}

func (validator *Validator) Update(config health.TimedCounterConfig) {
	client := resty.New()
	var result map[string]interface{}
	resp, err := client.R().SetResult(&result).Get(validator.url)
	if err != nil || resp.StatusCode() != 200 {
		logrus.WithError(err).WithField("validator", validator.String()).Info("Failed to query validator info")
		return
	}
	jailed := result["validator"].(map[string]interface{})["jailed"].(bool)

	if logrus.IsLevelEnabled(logrus.DebugLevel) {
		jsonStr, _ := json.Marshal(result)
		logrus.WithFields(logrus.Fields{
			"validator": validator.String(),
			"info":      fmt.Sprintf("%+v", jsonStr),
		}).Debug("Validator status report")
	}

	if jailed {
		unhealthy, unrecovered, elapsed := validator.health.OnFailure(config)

		if unhealthy {
			logrus.WithFields(logrus.Fields{
				"elapsed":   prettyElapsed(elapsed),
				"validator": validator.String(),
			}).Error("Validator jailed")
		}

		if unrecovered {
			logrus.WithFields(logrus.Fields{
				"elapsed":   prettyElapsed(elapsed),
				"validator": validator.String(),
			}).Error("Validator jailed and not recovered yet")
		}
	} else if recovered, elapsed := validator.health.OnSuccess(config); recovered {
		logrus.WithFields(logrus.Fields{
			"elapsed":   prettyElapsed(elapsed),
			"validator": validator.String(),
		}).Warn("Validator unfailed now")
	}
}

func (validator *Validator) CheckStatusSilence() {
	isConnected := false

	if strings.HasPrefix(validator.address, "http") {
		// Connect to the RPC endpoint
		_, err := web3go.NewClient(validator.address)
		if err == nil {
			isConnected = true
		}
	} else {
		// Connect to the IPC endpoint
		_, err := web3go.NewClient(fmt.Sprintf("http://%s", validator.address))
		if err != nil {
			_, err = web3go.NewClient(fmt.Sprintf("https://%s", validator.address))
			if err == nil {
				isConnected = true
			}
		} else {
			isConnected = true
		}
	}

	if isConnected {
		logrus.WithFields(logrus.Fields{
			"address": validator.name,
			"ip":      validator.address,
		}).Info("Validator connection succeeded")
	} else {
		logrus.WithFields(logrus.Fields{
			"address": validator.name,
			"ip":      validator.address,
		}).Info("Validator connection failed")
	}
}
