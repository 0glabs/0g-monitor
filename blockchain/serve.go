package blockchain

import (
	"net/url"
	"time"

	"github.com/Conflux-Chain/go-conflux-util/health"
	"github.com/Conflux-Chain/go-conflux-util/viper"
	"github.com/sirupsen/logrus"
)

type Config struct {
	Nodes                  map[string]string
	Interval               time.Duration `default:"60s"`
	AvailabilityReport     health.TimedCounterConfig
	NodeHeightReport       HeightReportConfig
	BlockchainHeightReport health.TimedCounterConfig
	Validators             map[string]string
	ValidatorReport        health.TimedCounterConfig
	PrivateKey             string
	CosmosRPC              string `default:"https://cosmosrpc-test.0g.ai/"`
}

const ValidatorFile = "data/validator_rpcs.csv"

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

	// Connect to all fullnodes
	var nodes []*Node
	for name, url := range config.Nodes {
		logrus.WithField("name", name).WithField("url", url).Debug("Start to monitor fullnode")
		nodes = append(nodes, MustNewNode(name, url))
	}

	var validators []*Validator
	url, _ := url.Parse(config.CosmosRPC)
	for name, address := range config.Validators {
		logrus.WithField("name", name).WithField("address", address).Debug("Start to monitor validator")
		validators = append(validators, MustNewValidator(url, name, address))
	}

	// Monitor once immediately
	monitorOnce(&config, nodes, validators)

	// Monitor node status periodically
	ticker := time.NewTicker(config.Interval)
	defer ticker.Stop()

	for range ticker.C {
		monitorOnce(&config, nodes, validators)
	}
}

func monitorOnce(config *Config, nodes []*Node, validators []*Validator) {
	for _, v := range nodes {
		v.UpdateHeight(config.AvailabilityReport)
	}

	max := FindMaxBlockHeight(nodes)
	if max == 0 {
		return
	}

	logrus.WithField("height", max).Debug("Fullnode status report")

	defaultBlockchainHeightHealth.Update(config.BlockchainHeightReport, max)

	for _, v := range nodes {
		v.CheckHeight(&config.NodeHeightReport, max)
	}

	for _, v := range validators {
		v.Update(config.ValidatorReport)
	}
}
