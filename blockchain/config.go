package blockchain

import (
	"github.com/Conflux-Chain/go-conflux-util/health"
)

type AvailabilityReport struct {
	health.TimedCounterConfig

	MaxGap uint64 `default:"30"`
}

type MempoolMonitorReport struct {
	health.TimedCounterConfig

	AlarmThreshold uint64 `default:"90"`
	PoolSize       uint64 `default:"5000"`
}

type Config struct {
	Nodes                     map[string]string
	MempoolInterval           int `default:"1"`
	NodeInterval              int `default:"3"`
	ValidatorInterval         int `default:"15"`
	AvailabilityReport        AvailabilityReport
	NodeHeightReport          HeightReportConfig
	BlockchainHeightReport    health.TimedCounterConfig
	Validators                map[string]string
	ValidatorReport           health.TimedCounterConfig
	MempoolReport             MempoolMonitorReport
	PrivateKey                string
	CosmosRPC                 string `default:"https://127.0.0.1:26657"`
	CosmosRest                string `default:"http://127.0.0.1:1317"`
	CometbftRPC               string `default:"http://127.0.0.1:26657"`
	BlockTxCntLimit           int    `default:"100"`
	FailedTxCntAlarmThreshold int    `default:"2"`
	Mode                      string `default:"localtest"`
}
