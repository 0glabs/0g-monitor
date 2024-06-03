package blockchain

import (
	"time"

	"github.com/Conflux-Chain/go-conflux-util/health"
	"github.com/Conflux-Chain/go-conflux-util/viper"
	"github.com/sirupsen/logrus"
)

type Config struct {
	Nodes                  map[string]string
	Interval               time.Duration `default:"5s"`
	AvailabilityReport     health.TimedCounterConfig
	NodeHeightReport       HeightReportConfig
	BlockchainHeightReport health.TimedCounterConfig
	Validators             map[string]string
	ValidatorReport        health.TimedCounterConfig
	StorageNodes           map[string]string
	PrivateKey             string
}

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

	if len(config.Nodes) == 0 {
		return
	}

	// Connect to all fullnodes
	var nodes []*Node
	for name, url := range config.Nodes {
		logrus.WithField("name", name).WithField("url", url).Debug("Start to monitor fullnode")
		nodes = append(nodes, MustNewNode(name, url))
	}
	defer func() {
		for _, v := range nodes {
			defer v.Close()
		}
	}()

	var validators []*Validator
	for name, address := range config.Validators {
		logrus.WithField("name", name).WithField("address", address).Debug("Start to monitor validator")
		validators = append(validators, MustNewValidator(nodes[0].Client, name, address))
	}

	var storageNodes []*StorageNode
	for name, address := range config.StorageNodes {
		logrus.WithField("name", name).WithField("address", address).Debug("Start to monitor storage node")
		storageNodes = append(storageNodes, MustNewStorageNode(name, address))
	}

	// Monitor node status periodically
	ticker := time.NewTicker(config.Interval)
	defer ticker.Stop()

	for range ticker.C {
		monitorOnce(&config, nodes, validators, storageNodes)
	}
}

func monitorOnce(config *Config, nodes []*Node, validators []*Validator, storageNodes []*StorageNode) {
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

	for _, v := range storageNodes {
		v.CheckStatus(config.PrivateKey, max)
	}
}
