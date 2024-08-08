package da

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/Conflux-Chain/go-conflux-util/health"
	"github.com/Conflux-Chain/go-conflux-util/viper"
	_ "github.com/go-sql-driver/mysql"
	"github.com/sirupsen/logrus"
)

type DBConfig struct {
	Host     string
	Port     string
	Username string
	Password string
	DbName   string
}

type Config struct {
	Nodes    map[string]string
	Interval time.Duration `default:"600s"`
	// DaNodeReport   health.TimedCounterConfig
	DaClientReport health.TimedCounterConfig
	// DbConfig       DBConfig
}

const (
	NodeDisconnected string = "DISCONNECTED"
	NodeConnected    string = "CONNECTED"
)

const ValidatorFile = "data/validator_rpcs.csv"
const operatorSysLogFile = "log/monitor.log"

func MustMonitorFromViper() {
	var config Config
	viper.MustUnmarshalKey("da", &config)
	Monitor(config)
}

func Monitor(config Config) {
	// lumberjackLogger := &lumberjack.Logger{
	// 	Filename: operatorSysLogFile,
	// 	MaxSize:  500, // MB
	// 	MaxAge:   300, // days
	// 	Compress: false,
	// }
	// logrus.SetOutput(io.MultiWriter(os.Stdout, lumberjackLogger))

	logrus.WithFields(logrus.Fields{
		"nodes": len(config.Nodes),
		// "validators": len(config.Validators),
	}).Info("Start to monitor da client")

	// f, err := os.Open(ValidatorFile)
	// if err != nil {
	// 	fmt.Println("Error opening csv:", err)
	// 	return
	// }
	// defer f.Close()

	// db, err := CreateDBClients(config.DbConfig)
	// if err != nil {
	// 	fmt.Println("Error opening db:", err)
	// 	return
	// }
	// defer db.Close()

	// Read the file into a dataframe
	// df := dataframe.ReadCSV(f)

	// var userDaNodes []*DaNode
	var userDaClients []*DaClient
	for name, url := range config.Nodes {
		logrus.WithField("name", name).WithField("url", url).Debug("Start to monitor da client")
		userDaClients = append(userDaClients, MustNewDaClient(name, url))
	}

	defer func() {
		for _, v := range userDaClients {
			defer v.Close()
		}
	}()

	// for i := 0; i < df.Nrow(); i++ {
	// 	discordId := df.Subset(i).Col("discord_id").Elem(0).String()
	// 	validatorAddress := df.Subset(i).Col("validator_address").Elem(0).String()
	// 	da_node_grpc := df.Subset(i).Col("da_node_grpc").Elem(0).String()
	// 	ips := strings.Split(da_node_grpc, ",")
	// 	for _, ip := range ips {
	// 		ip = strings.TrimSpace(ip)
	// 		logrus.WithField("discord_id", discordId).WithField("ip", ip).Debug("Start to monitor user da node")
	// 		currNode := MustNewDaNode(discordId, validatorAddress, ip)
	// 		if currNode != nil {
	// 			userDaNodes = append(userDaNodes, currNode)
	// 		}
	// 	}
	// 	// da_client_grpc := df.Subset(i).Col("da_client_grpc").Elem(0).String()
	// 	// ips = strings.Split(da_client_grpc, ",")
	// 	// for _, ip := range ips {
	// 	// 	ip = strings.TrimSpace(ip)
	// 	// 	logrus.WithField("discord_id", discordId).WithField("ip", ip).Debug("Start to monitor user da client")
	// 	// 	currNode := MustNewDaClient(discordId, validatorAddress, ip)
	// 	// 	if currNode != nil {
	// 	// 		userDaClients = append(userDaClients, currNode)
	// 	// 	}
	// 	// }
	// }

	// Monitor node status periodically
	ticker := time.NewTicker(config.Interval)
	defer ticker.Stop()

	for range ticker.C {
		monitorOnce(&config, userDaClients)
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

func monitorOnce(config *Config, userClients []*DaClient) {
	// for _, v := range userNodes {
	// 	v.CheckStatusSilence(config.DaNodeReport, db)
	// }

	for _, v := range userClients {
		v.CheckStatusSilence(config.DaClientReport)
	}
}
