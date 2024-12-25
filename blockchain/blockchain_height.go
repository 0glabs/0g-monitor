package blockchain

import (
	"github.com/0glabs/0g-monitor/utils"
	"github.com/Conflux-Chain/go-conflux-util/health"
	"github.com/Conflux-Chain/go-conflux-util/metrics"
	"github.com/sirupsen/logrus"
)

var defaultBlockchainHeightHealth BlockchainHeightHealth

type BlockchainHeightHealth struct {
	height uint64
	health health.TimedCounter
}

func (bhh *BlockchainHeightHealth) Update(config health.TimedCounterConfig, height uint64) {
	if bhh.height == 0 {
		metrics.GetOrRegisterGauge(chainHeightHaltPattern).Update(0)
	}

	if height > bhh.height {
		if recovered, elapsed := bhh.health.OnSuccess(config); recovered {
			logrus.WithFields(logrus.Fields{
				"elapsed": utils.PrettyElapsed(elapsed),
				"old":     bhh.height,
				"new":     height,
			}).Warn("Blockchain height is growing again")

			metrics.GetOrRegisterGauge(chainHeightHaltPattern).Update(0)
		}

		bhh.height = height
	} else {
		unhealthy, unrecovered, elapsed := bhh.health.OnFailure(config)

		newHeight := height
		if newHeight == 0 {
			newHeight = bhh.height
		}

		if unhealthy {
			logrus.WithFields(logrus.Fields{
				"elapsed": utils.PrettyElapsed(elapsed),
				"height":  newHeight,
			}).Error("Blockchain height stops growing")

			metrics.GetOrRegisterGauge(chainHeightHaltPattern).Update(1)
		}

		if unrecovered {
			logrus.WithFields(logrus.Fields{
				"elapsed": utils.PrettyElapsed(elapsed),
				"height":  newHeight,
			}).Error("Blockchain height stops growing for a long time and not recovered yet")
		}
	}
}
