package blockchain

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/0glabs/0g-monitor/utils"
	"github.com/Conflux-Chain/go-conflux-util/health"
	"github.com/Conflux-Chain/go-conflux-util/metrics"
	"github.com/openweb3/web3go"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type Validator struct {
	url      string
	name     string
	address  string
	health   health.TimedCounter
	rpcError string
	jailed   bool
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

	metrics.GetOrRegisterGauge(validatorJailedPattern, name).Update(0)

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
	err := executeRequest(
		func() error {
			jailed, err := IsValidatorJailed(validator.url)
			if err != nil {
				logrus.WithError(err).WithField("validator", validator.String()).Info("Failed to query validator info")
				return err
			}

			validator.jailed = jailed

			return nil
		},
		func(err error, unhealthy, unrecovered bool, elapsed time.Duration) {
			validator.rpcError = err.Error()
			if unhealthy {
				logrus.WithFields(logrus.Fields{
					"validator": validator.name,
					"elapsed":   utils.PrettyElapsed(elapsed),
					"error":     validator.rpcError,
				}).Error("Node cosmos RPC became unhealthy")
			}

			// remind unhealthy
			if unrecovered {
				logrus.WithFields(logrus.Fields{
					"validator": validator.name,
					"elapsed":   utils.PrettyElapsed(elapsed),
					"rpcError":  validator.rpcError,
				}).Error("Node cosmos RPC not recovered yet")
			}
		},
		func(recovered bool, elapsed time.Duration) {
			validator.rpcError = ""
			// report on recovered
			if recovered {
				logrus.WithFields(logrus.Fields{
					"validator": validator.name,
					"elapsed":   utils.PrettyElapsed(elapsed),
				}).Warn("Node cosmos RPC is healthy now")
			}
		},
		nodeCosmosRpcLatencyPattern, nodeCosmosRpcUnhealthPattern, validator.name,
		&validator.health,
		config,
	)

	if err == nil {
		if validator.jailed {
			unhealthy, unrecovered, elapsed := validator.health.OnFailure(config)

			if unhealthy {
				logrus.WithFields(logrus.Fields{
					"elapsed":   utils.PrettyElapsed(elapsed),
					"validator": validator.String(),
				}).Error("Validator jailed")

				metrics.GetOrRegisterGauge(validatorJailedPattern, validator.name).Update(1)
			}

			if unrecovered {
				logrus.WithFields(logrus.Fields{
					"elapsed":   utils.PrettyElapsed(elapsed),
					"validator": validator.String(),
				}).Error("Validator jailed and not recovered yet")
			}
		} else if recovered, elapsed := validator.health.OnSuccess(config); recovered {
			logrus.WithFields(logrus.Fields{
				"elapsed":   utils.PrettyElapsed(elapsed),
				"validator": validator.String(),
			}).Warn("Validator unfailed now")

			metrics.GetOrRegisterGauge(validatorJailedPattern, validator.name).Update(0)
		}
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
