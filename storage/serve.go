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
	_ "github.com/go-sql-driver/mysql"
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
	Interval          time.Duration `default:"7200s"`
	Nodes             map[string]string
	KvNodes           map[string]string
	StorageNodeReport health.TimedCounterConfig
	DbConfig          DBConfig
}

const (
	NodeDisconnected string = "DISCONNECTED"
	NodeConnected    string = "CONNECTED"
)

const DefaultTimeout = 2
const ValidatorFile = "data/validator_rpcs.csv"
const operatorSysLogFile = "log/monitor.log"

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
		"storage_nodes": len(config.Nodes),
	}).Info("Start to monitor storage services")

	logrus.WithFields(logrus.Fields{
		"storage_kv": len(config.KvNodes),
	}).Info("Start to monitor kv services")

	var storageNodes []*StorageNode
	for name, ip := range config.Nodes {
		logrus.WithField("name", name).WithField("ip", ip).Debug("Start to monitor storage node")
		storageNodes = append(storageNodes, MustNewStorageNode(name, name, ip))
	}

	var kvNodes []*KvNode
	for name, ip := range config.KvNodes {
		logrus.WithField("name", name).WithField("ip", ip).Debug("Start to monitor kv node")
		kvNodes = append(kvNodes, MustNewKvNode(name, name, ip))
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
	var userStorageNodes []*StorageNode
	var userKvNodes []*KvNode
	for i := 0; i < df.Nrow(); i++ {
		discordId := df.Subset(i).Col("discord_id").Elem(0).String()
		validatorAddress := df.Subset(i).Col("validator_address").Elem(0).String()
		storage_rpc := df.Subset(i).Col("storage_node_rpc").Elem(0).String()
		ips := strings.Split(storage_rpc, ",")
		for _, ip := range ips {
			ip = strings.TrimSpace(ip)
			if len(ip) == 0 {
				continue
			}
			logrus.WithField("discord_id", discordId).WithField("ip", ip).Debug("Start to monitor user storage node")
			currNode := MustNewStorageNode(discordId, validatorAddress, ip)
			if currNode != nil {
				userStorageNodes = append(userStorageNodes, currNode)
			}
		}
		kv_rpc := df.Subset(i).Col("storage_kv_rpc").Elem(0).String()
		ips = strings.Split(kv_rpc, ",")
		for _, ip := range ips {
			ip = strings.TrimSpace(ip)
			if len(ip) == 0 {
				continue
			}
			logrus.WithField("discord_id", discordId).WithField("ip", ip).Debug("Start to monitor user kv node")
			currNode := MustNewKvNode(discordId, validatorAddress, ip)
			if currNode != nil {
				userKvNodes = append(userKvNodes, currNode)
			}
		}
	}

	// Monitor node status periodically
	ticker := time.NewTicker(config.Interval)
	defer ticker.Stop()

	for range ticker.C {
		monitorStorageNodeOnce(&config, db, storageNodes, userStorageNodes)
		monitorKvNodeOnce(&config, db, kvNodes, userKvNodes)
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

func monitorStorageNodeOnce(config *Config, db *sql.DB, storageNodes, userNodes []*StorageNode) {
	for _, v := range storageNodes {
		v.CheckStatus(config.StorageNodeReport)
	}

	for _, v := range userNodes {
		v.CheckStatusSilence(config.StorageNodeReport, db)
	}
}

func monitorKvNodeOnce(config *Config, db *sql.DB, kvNodes, userNodes []*KvNode) {
	for _, v := range kvNodes {
		v.CheckStatus(config.StorageNodeReport)
	}

	for _, v := range userNodes {
		v.CheckStatusSilence(config.StorageNodeReport, db)
	}
}

func PrettyElapsed(elapsed time.Duration) string {
	return fmt.Sprint(elapsed.Truncate(time.Second))
}
