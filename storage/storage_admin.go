package storage

import (
	"context"
	"fmt"
	"strings"

	"github.com/0glabs/0g-storage-client/node"
	"github.com/Conflux-Chain/go-conflux-util/metrics"
	providers "github.com/openweb3/go-rpc-provider/provider_wrapper"

	"github.com/sirupsen/logrus"
)

type Statistics struct {
	movingAvg   []int64
	peak        int64
	decreaseCnt int
}

type AdminNode struct {
	client     *node.AdminClient
	ip         string
	statistics Statistics
}

func MustNewAdminNode(ip string) *AdminNode {
	adminNode, err := NewAdminNode(ip)
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"ip": ip,
		}).WithError(err).Info("Failed to create admin node")
		return nil
	}

	return adminNode
}

func NewAdminNode(ip string) (*AdminNode, error) {
	ip = strings.TrimSpace(ip)
	if len(ip) == 0 {
		return nil, fmt.Errorf("empty ip")
	}

	client, err := node.NewAdminClient(ip, providers.Option{
		RequestTimeout: DefaultTimeout,
	})
	if err != nil {
		return nil, err
	}
	return &AdminNode{
		client: client,
		ip:     ip,
	}, nil
}

func (adminNode *AdminNode) CheckDiscoveredPeers() {
	peers, err := adminNode.client.GetPeers(context.Background())
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"ip": adminNode.ip,
		}).WithError(err).Warn("Failed to get discovered peers")
		return
	}

	// Calculate moving average
	numPeers := int64(len(peers))
	statistics := &adminNode.statistics

	statistics.movingAvg = append(statistics.movingAvg, numPeers)
	if len(statistics.movingAvg) > 5 {
		statistics.movingAvg = statistics.movingAvg[1:] // Keep window size to 5
	}

	// Calculate the current moving average
	var sum int64
	for _, val := range statistics.movingAvg {
		sum += val
	}
	currentAvg := sum / int64(len(statistics.movingAvg))

	// Track the peak value
	if currentAvg > statistics.peak {
		statistics.peak = currentAvg
		statistics.decreaseCnt = 0 // Reset the decrease counter on new peak
	} else {
		statistics.decreaseCnt++
	}

	// Check for warnings
	if statistics.decreaseCnt >= 10 && currentAvg < int64(0.6*float64(statistics.peak)) {
		logrus.WithFields(logrus.Fields{
			"ip":          adminNode.ip,
			"currentAvg":  currentAvg,
			"lastPeak":    statistics.peak,
			"decreaseCnt": statistics.decreaseCnt,
		}).Warn("Discovered peers dropped sharply below 60% of the last peak")
	}

	// Update the metric
	metrics.GetOrRegisterGauge("storage_node/storage_layer/discovered_peers").Update(numPeers)

}
