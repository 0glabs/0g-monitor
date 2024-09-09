package stat

import (
	"context"
	"fmt"

	"github.com/0glabs/0g-monitor/storage"
	"github.com/0glabs/0g-storage-client/common/shard"
	"github.com/0glabs/0g-storage-client/node"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var shardCmd = &cobra.Command{
	Use:   "shard",
	Short: "Statistic shard config of storage node network",
	Run:   statShard,
}

func init() {
	Cmd.AddCommand(shardCmd)
}

func statShard(*cobra.Command, []string) {
	client := mustNewIndexerClient()
	defer client.Close()
	logrus.Info("Dailed to indexer")

	// retrieve discovered nodes from indexer
	ips, err := client.GetNodeLocations(context.Background())
	if err != nil {
		logrus.WithError(err).Fatal("Failed to retrieve node locations")
	}
	logrus.WithField("ips", len(ips)).Info("Succeeded to retrieve node IP locations")

	// retrieve shard configs in parallel
	nodes := make([]string, 0, len(ips))
	for ip := range ips {
		nodes = append(nodes, fmt.Sprintf("http://%v:5678", ip))
	}
	shardConfigFunc := func(client *node.ZgsClient, ctx context.Context) (shard.ShardConfig, error) {
		return client.GetShardConfig(ctx)
	}
	logrus.Info("Begin to query shard configs in parallel")
	result, err := storage.ParallelQueryRpc(context.Background(), nodes, shardConfigFunc, serialOpt)
	if err != nil {
		logrus.WithError(err).Fatal("Failed to query shard configs in parallel")
	}

	// stat shard configs
	var rpcFailures int
	invalidShardNodes := make(map[string]shard.ShardConfig)
	shardDistribution := make(map[uint64]map[uint64]int)

	for node, rpcResult := range result {
		if rpcResult.Err != nil {
			rpcFailures++
		} else if rpcResult.Data.NumShard > 1024 {
			invalidShardNodes[node] = rpcResult.Data
		} else if v, ok := shardDistribution[rpcResult.Data.NumShard]; ok {
			v[rpcResult.Data.ShardId]++
		} else {
			shardDistribution[rpcResult.Data.NumShard] = map[uint64]int{
				rpcResult.Data.ShardId: 1,
			}
		}
	}

	fmt.Println()
	fmt.Println("RPC Failures:", rpcFailures)
	if len(invalidShardNodes) > 0 {
		fmt.Println("Invalid shard config nodes:")
		for node, config := range invalidShardNodes {
			fmt.Printf("\t%v: %v / %v\n", node, config.ShardId, config.NumShard)
		}
	}
	fmt.Println("Shard distribution:")
	for numShard, shardIds := range shardDistribution {
		fmt.Println("\tNum shards:", numShard)
		for id, count := range shardIds {
			fmt.Printf("\t\t%v: %v\n", id, count)
		}
	}
}
