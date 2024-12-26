package blockchain

import (
	"time"

	"github.com/Conflux-Chain/go-conflux-util/health"
	"github.com/Conflux-Chain/go-conflux-util/metrics"
)

func executeRequest(
	doWhat func() error,
	errHdlr func(err error, unhealthy, unrecovered bool, elapsed time.Duration),
	successHdlr func(recovered bool, elapsed time.Duration),
	latencyPattern, unhealtyPattern, nodeName string,
	health *health.TimedCounter,
	heathCfg health.TimedCounterConfig,
) error {
	start := time.Now()
	err := doWhat()
	elapsed := time.Since(start).Nanoseconds()
	if len(nodeName) > 0 {
		metrics.GetOrRegisterHistogram(latencyPattern, nodeName).Update(elapsed)
	} else {
		metrics.GetOrRegisterHistogram("%s", latencyPattern).Update(elapsed)
	}
	if err != nil {
		unhealthy, unrecovered, elapsed := health.OnFailure(heathCfg)
		if unhealthy {
			if len(nodeName) > 0 {
				metrics.GetOrRegisterGauge(unhealtyPattern, nodeName).Update(1)
			} else {
				metrics.GetOrRegisterGauge("%s", unhealtyPattern).Update(1)
			}
		}

		if errHdlr != nil {
			errHdlr(err, unhealthy, unrecovered, time.Duration(elapsed))
		}
	} else {
		recovered, elapsed := health.OnSuccess(heathCfg)
		if recovered {
			if len(nodeName) > 0 {
				metrics.GetOrRegisterGauge(unhealtyPattern, nodeName).Update(0)
			} else {
				metrics.GetOrRegisterGauge("%s", unhealtyPattern).Update(0)
			}
		}

		if successHdlr != nil {
			successHdlr(recovered, time.Duration(elapsed))
		}
	}

	return err
}
