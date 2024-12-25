package blockchain

import (
	"net/url"
	"time"

	"github.com/0glabs/0g-monitor/utils"
	"github.com/Conflux-Chain/go-conflux-util/health"
	"github.com/Conflux-Chain/go-conflux-util/metrics"
	"github.com/sirupsen/logrus"
)

type Mempool struct {
	url string

	cometbftRpcHealth health.TimedCounter
	cometbftRpcError  string // last rpc error message
}

func MustNewMempool(urlstr string) *Mempool {
	url, _ := url.Parse(urlstr)

	metrics.GetOrRegisterHistogram(mempoolUncommitTxCntPattern).Update(0)
	metrics.GetOrRegisterGauge(mempoolHighLoadPattern).Update(0)

	return &Mempool{
		url: url.String(),
	}
}

func (m *Mempool) UpdateUncommitTxCnt(config health.TimedCounterConfig) int {
	var unconfirmedTxCnt int
	executeRequest(
		func() error {
			var err error
			unconfirmedTxCnt, err = rpcGetUncommitTxCnt(m.url)
			if err != nil {
				return err
			} else {
				return nil
			}
		},
		func(err error, unhealthy, unrecovered bool, elapsed time.Duration) {
			m.cometbftRpcError = err.Error()
			// report unhealthy
			if unhealthy {
				logrus.WithFields(logrus.Fields{
					"node":    "mempool",
					"elapsed": utils.PrettyElapsed(elapsed),
					"error":   err,
				}).Error("Node cometbft RPC became unhealthy")
			}

			// remind unhealthy
			if unrecovered {
				logrus.WithFields(logrus.Fields{
					"node":    "mempool",
					"elapsed": utils.PrettyElapsed(elapsed),
				}).Error("Node cometbft RPC not recovered yet")
			}
		},
		func(recovered bool, elapsed time.Duration) {
			m.cometbftRpcError = ""
			if recovered {
				logrus.WithFields(logrus.Fields{
					"node":    "mempool",
					"elapsed": utils.PrettyElapsed(elapsed),
				}).Warn("Node cometbft RPC is healthy now")
			}
		},
		nodeCometbftRpcLatencyPattern, nodeCometbftRpcUnhealthPattern, "mempool",
		&m.cometbftRpcHealth,
		config,
	)

	return unconfirmedTxCnt
}
