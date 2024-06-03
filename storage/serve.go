package storage

import (
	"time"

	"github.com/Conflux-Chain/go-conflux-util/health"
	"github.com/Conflux-Chain/go-conflux-util/viper"
	"github.com/sirupsen/logrus"
)

type Config struct {
	Interval          time.Duration `default:"5s"`
	StorageNodes      map[string]string
	StorageNodeReport health.TimedCounterConfig
	PrivateKey        string
}

func MustMonitorFromViper() {
	var config Config
	viper.MustUnmarshalKey("storage", &config)
	Monitor(config)
}

func Monitor(config Config) {
	logrus.WithFields(logrus.Fields{
		"storage_nodes": len(config.StorageNodes),
	}).Info("Start to monitor storage services")

	if len(config.StorageNodes) == 0 {
		return
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
		monitorOnce(&config, storageNodes)
	}
}

func monitorOnce(config *Config, storageNodes []*StorageNode) {
	for _, v := range storageNodes {
		v.CheckStatus(config.StorageNodeReport, config.PrivateKey)
	}
}
