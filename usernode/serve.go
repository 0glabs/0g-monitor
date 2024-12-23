package usernode

import (
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/Conflux-Chain/go-conflux-util/viper"
	"github.com/go-gota/gota/dataframe"
	"github.com/sirupsen/logrus"
)

type Config struct {
	Nodes      map[string]string
	Interval   time.Duration `default:"60s"`
	PrivateKey string
	CosmosRPC  string `default:"https://cosmosrpc-test.0g.ai/"`
}

const ValidatorFile = "data/validator_rpcs.csv"

func MustMonitorFromViper() {
	var config Config
	viper.MustUnmarshalKey("usernode", &config)
	Monitor(config)
}

func Monitor(config Config) {
	f, err := os.Open(ValidatorFile)
	if err != nil {
		fmt.Println("Error opening csv:", err)
		return
	}
	defer f.Close()

	url, _ := url.Parse(config.CosmosRPC)

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

			currNode := MustNewValidator(url, validatorAddress, ip)
			if currNode != nil {
				userNodes = append(userNodes, currNode)
			}
		}
	}

	logrus.WithFields(logrus.Fields{
		"userNodes": len(userNodes),
	}).Info("Start to monitor user node")

	// Monitor once immediately
	monitorOnce(userNodes)

	// Monitor node status periodically
	ticker := time.NewTicker(config.Interval)
	defer ticker.Stop()

	for range ticker.C {
		monitorOnce(userNodes)
	}
}

func monitorOnce(userNodes []*Validator) {
	for _, v := range userNodes {
		v.CheckStatusSilence()
	}
}
