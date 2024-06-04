package storage

import (
	"database/sql"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/Conflux-Chain/go-conflux-util/health"
	"github.com/Conflux-Chain/go-conflux-util/viper"
	"github.com/go-gota/gota/dataframe"
	"github.com/sirupsen/logrus"
	"gopkg.in/natefinch/lumberjack.v2"
)

type DBConfig struct {
	Host     string
	Port     string
	Username string
	Password string
	DbName   string
}

type Config struct {
	Interval          time.Duration `default:"1800s"`
	StorageNodes      map[string]string
	StorageNodeReport health.TimedCounterConfig
	DbConfig          DBConfig
}

const ValidatorFile = "../liveness/user-data/validator_rpcs.csv"
const operatorSysLogFile = "/var/log/0g-monitor/monitor.log"

func MustMonitorFromViper() {
	var config Config
	viper.MustUnmarshalKey("storage", &config)
	Monitor(config)
}

func Monitor(config Config) {
	lumberjackLogger := &lumberjack.Logger{
		Filename: operatorSysLogFile,
		MaxSize:  20, // MB
		MaxAge:   30, // days
		Compress: false,
	}
	logrus.SetOutput(io.MultiWriter(os.Stdout, lumberjackLogger))

	logrus.WithFields(logrus.Fields{
		"storage_nodes": len(config.StorageNodes),
	}).Info("Start to monitor storage services")

	if len(config.StorageNodes) == 0 {
		return
	}

	var storageNodes []*StorageNode
	for name, ip := range config.StorageNodes {
		logrus.WithField("name", name).WithField("ip", ip).Debug("Start to monitor storage node")
		storageNodes = append(storageNodes, MustNewStorageNode(name, name, ip))
	}

	f, err := os.Open(ValidatorFile)
	if err != nil {
		fmt.Println("Error opening csv:", err)
		return
	}
	defer f.Close()

	db, err := CreateDBClients(config.DbConfig)
	if err != nil {
		fmt.Println("Error opening db:", err)
		return
	}
	defer db.Close()

	// Read the file into a dataframe
	df := dataframe.ReadCSV(f)
	var userNodes []*StorageNode
	for i := 0; i < df.Nrow(); i++ {
		discordId := df.Subset(1).Col("discord_id").Elem(0).String()
		validatorAddress := df.Subset(1).Col("validator_address").Elem(0).String()
		rpc := df.Subset(1).Col("storage_node_rpc").Elem(0).String()
		ips := strings.Split(rpc, ",")
		for _, ip := range ips {
			ip = strings.TrimSpace(ip)
			logrus.WithField("discord_id", discordId).WithField("ip", ip).Debug("Start to monitor user node")
			userNodes = append(userNodes, MustNewStorageNode(discordId, validatorAddress, ip))
		}
	}

	// Monitor node status periodically
	ticker := time.NewTicker(config.Interval)
	defer ticker.Stop()

	for range ticker.C {
		monitorOnce(&config, db, storageNodes, userNodes)
	}
}

func CreateDBClients(config DBConfig) (*sql.DB, error) {
	// Define the MySQL connection string
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s", config.Username, config.Password, config.Host, config.Port, config.DbName)

	// Open a connection to the database
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}

	// Test the connection
	err = db.Ping()
	if err != nil {
		return nil, err
	}

	return db, nil
}

func monitorOnce(config *Config, db *sql.DB, storageNodes []*StorageNode, userNodes []*StorageNode) {
	for _, v := range storageNodes {
		v.CheckStatus(config.StorageNodeReport)
	}

	for _, v := range userNodes {
		v.CheckStatusSilence(config.StorageNodeReport, db)
	}
}
