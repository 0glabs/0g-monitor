package stat

import (
	"context"
	"fmt"

	"github.com/0glabs/0g-monitor/storage/files"
	"github.com/0glabs/0g-storage-client/common/shard"
	"github.com/0glabs/0g-storage-client/node"
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
	shards := mustStatRpc(func(client *node.ZgsClient, ctx context.Context) (shard.ShardConfig, error) {
		return client.GetShardConfig(ctx)
	})

	// stat shard configs
	var rpcFailures int
	invalidShardNodes := make(map[string]shard.ShardConfig)
	shardCounter := files.NewShardCounter()

	for node, rpcResult := range shards {
		if rpcResult.Err != nil {
			rpcFailures++
		} else if rpcResult.Data.NumShard > 1024 {
			invalidShardNodes[node] = rpcResult.Data
		} else {
			shardCounter.Insert(rpcResult.Data)
		}
	}

	fmt.Println("\nRPC Failures:", rpcFailures)

	if len(invalidShardNodes) > 0 {
		fmt.Println("\nInvalid shard config nodes:")
		for node, config := range invalidShardNodes {
			fmt.Printf("\t%v: %v / %v\n", node, config.ShardId, config.NumShard)
		}
	}
	prettyPrint(shardCounter, "\nShard distribution")
}

func prettyPrint(counter *files.ShardCounter, label string) {
	fmt.Printf("%v, replica = %v:\n", label, counter.Replica())

	for numShard, id2Count := range counter.Items() {
		fmt.Println("\tNum shard:", numShard)

		for id, count := range id2Count {
			fmt.Printf("\t\t%v: %v\n", id, count)
		}
	}
}
