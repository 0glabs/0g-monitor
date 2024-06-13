package blockchain

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/Conflux-Chain/go-conflux-util/health"
	"github.com/Conflux-Chain/go-conflux-util/viper"
	"github.com/go-gota/gota/dataframe"
	"github.com/sirupsen/logrus"
)

type Config struct {
	Nodes                  map[string]string
	Interval               time.Duration `default:"600s"`
	AvailabilityReport     health.TimedCounterConfig
	NodeHeightReport       HeightReportConfig
	BlockchainHeightReport health.TimedCounterConfig
	Validators             map[string]string
	ValidatorReport        health.TimedCounterConfig
	PrivateKey             string
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

	if len(config.Nodes) == 0 {
		return
	}

	f, err := os.Open(ValidatorFile)
	if err != nil {
		fmt.Println("Error opening csv:", err)
		return
	}
	defer f.Close()

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
		validators = append(validators, MustNewValidator(nodes[0].Client, name, address, false))
	}

	// Read the file into a dataframe
	df := dataframe.ReadCSV(f)
	var userNodes []*Validator
	for i := 0; i < df.Nrow(); i++ {
		discordId := df.Subset(i).Col("discord_id").Elem(0).String()
		validatorAddress := df.Subset(i).Col("validator_address").Elem(0).String()
		rpc := df.Subset(i).Col("validator_rpc").Elem(0).String()
		ips := strings.Split(rpc, ",")
		for _, ip := range ips {
			ip = strings.TrimSpace(ip)
			logrus.WithField("discord_id", discordId).WithField("ip", ip).Debug("Start to monitor user validator node")
			currNode := MustNewValidator(nodes[0].Client, validatorAddress, ip, true)
			if currNode != nil {
				userNodes = append(userNodes, currNode)
			}
		}
	}

	// Monitor node status periodically
	ticker := time.NewTicker(config.Interval)
	defer ticker.Stop()

	for range ticker.C {
		monitorOnce(&config, nodes, validators, userNodes)
	}
}

func monitorOnce(config *Config, nodes []*Node, validators []*Validator, userNodes []*Validator) {
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

	for _, v := range userNodes {
		v.CheckStatusSilence()
	}
}
