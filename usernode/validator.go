package usernode

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/Conflux-Chain/go-conflux-util/metrics"
	"github.com/openweb3/web3go"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type Validator struct {
	url     string
	name    string
	address string
}

func MustNewValidator(url *url.URL, name, address string) *Validator {
	validator, err := NewValidator(url, name, address)
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"name":    name,
			"address": address,
		}).WithError(err).Info("Failed to create validator node")
		return nil
	}

	return validator
}

func NewValidator(url *url.URL, name, address string) (*Validator, error) {
	address = strings.TrimSpace(address)
	if len(address) == 0 {
		return nil, errors.New("empty address")
	}

	url.Path = "/cosmos/staking/v1beta1/validators/" + address

	metrics.GetOrRegisterGauge("monitor/blockchain/validator/jailed/%v", name).Update(0)

	return &Validator{
		url:     url.String(),
		name:    name,
		address: address,
	}, nil
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
