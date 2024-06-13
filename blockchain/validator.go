package blockchain

import (
	"fmt"
	"strings"

	"github.com/0glabs/0g-monitor/blockchain/contracts"
	"github.com/Conflux-Chain/go-conflux-util/health"
	"github.com/ethereum/go-ethereum/common"
	"github.com/openweb3/web3go"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

var precompileStaking = common.HexToAddress("0x0000000000000000000000000000000000000800")

type Validator struct {
	staking *contracts.StakingCaller
	name    string
	address string
	health  health.TimedCounter
}

func MustNewValidator(client *web3go.Client, name, address string, isCommunity bool) *Validator {
	validator, err := NewValidator(client, name, address, isCommunity)
	if err != nil {
		logrus.WithError(err).Fatal("Failed to create validator")
	}

	return validator
}

func NewValidator(client *web3go.Client, name, address string, isCommunity bool) (*Validator, error) {
	address = strings.TrimSpace(address)
	if len(address) == 0 {
		return nil, errors.New("empty address")
	}
	if isCommunity {
		return &Validator{
			staking: nil,
			name:    name,
			address: address,
		}, nil
	}

	caller, _ := client.ToClientForContract()
	staking, err := contracts.NewStakingCaller(precompileStaking, caller)
	if err != nil {
		return nil, errors.WithMessage(err, "Failed to new staking caller")
	}

	return &Validator{
		staking: staking,
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
	info, err := validator.staking.Validator(nil, validator.address)
	if err != nil {
		logrus.WithError(err).WithField("validator", validator.String()).Info("Failed to query validator info")
		return
	}

	if logrus.IsLevelEnabled(logrus.DebugLevel) {
		logrus.WithFields(logrus.Fields{
			"validator": validator.String(),
			"info":      fmt.Sprintf("%+v", info),
		}).Debug("Validator status report")
	}

	if len(info.OperatorAddress) == 0 || info.Jailed {
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
		}).Error("Validator connection failed")
	}
}
