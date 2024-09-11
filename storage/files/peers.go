package files

import (
	"context"
	"sync"

	"github.com/0glabs/0g-storage-client/common/parallel"
	"github.com/0glabs/0g-storage-client/common/shard"
	"github.com/0glabs/0g-storage-client/indexer"
	"github.com/0glabs/0g-storage-client/node"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type Discovery struct {
	config Config

	client *indexer.Client

	peers  []string
	shards map[string]shard.ShardConfig

	mu sync.RWMutex
}

func NewDiscovery(client *indexer.Client, config Config) (*Discovery, error) {
	d := Discovery{
		config: config,
		client: client,
	}

	if err := d.Discover(); err != nil {
		return nil, errors.WithMessage(err, "Failed to initialize discovery")
	}

	return &d, nil
}

func (d *Discovery) Discover() error {
	// retrieve discovered peers from indexer
	shardedNodes, err := d.client.GetShardedNodes(context.Background())
	if err != nil {
		return errors.WithMessage(err, "Failed to retrieve discovered nodes from indexer")
	}

	logger.WithFields(logrus.Fields{
		"trusted":    len(shardedNodes.Trusted),
		"discovered": len(shardedNodes.Discovered),
	}).Info("Discovered nodes retrieved from indexer")

	nodes := make([]string, 0, len(shardedNodes.Discovered))
	for _, v := range shardedNodes.Discovered {
		nodes = append(nodes, v.URL)
	}

	// query the latest shard config and remove failed nodes
	getShardConfigFunc := func(client *node.ZgsClient, ctx context.Context) (shard.ShardConfig, error) {
		return client.GetShardConfig(ctx)
	}
	option := parallel.RpcOption{
		Parallel: parallel.SerialOption{
			Routines: d.config.Routines,
		},
		Provider: defaultProviderOption,
	}
	result := parallel.QueryZgsRpc(context.Background(), nodes, getShardConfigFunc, option)

	peers := make([]string, 0, len(result))
	shards := make(map[string]shard.ShardConfig)
	for url, rpcResult := range result {
		if rpcResult.Err == nil && rpcResult.Data.IsValid() {
			peers = append(peers, url)
			shards[url] = rpcResult.Data
		}
	}

	logger.WithField("nodes", len(peers)).Info("Discovered nodes to statistic file status")

	d.mu.Lock()
	defer d.mu.Unlock()

	d.peers = peers
	d.shards = shards

	return nil
}

func (d *Discovery) GetPeers() ([]string, map[string]shard.ShardConfig) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	return d.peers, d.shards
}
