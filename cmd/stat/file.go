package stat

import (
	"context"
	"fmt"

	"github.com/0glabs/0g-monitor/storage/files"
	"github.com/0glabs/0g-storage-client/common/shard"
	"github.com/0glabs/0g-storage-client/node"
	"github.com/ethereum/go-ethereum/common"
	"github.com/spf13/cobra"
)

var (
	txSeq uint64
	root  string

	fileCmd = &cobra.Command{
		Use:   "file",
		Short: "Statistic file distribution of storage node network",
		Run:   statFileDistribution,
	}
)

func init() {
	fileCmd.Flags().Uint64Var(&txSeq, "seq", 0, "Tx seq to stat")
	fileCmd.Flags().StringVar(&root, "root", "", "File root to stat")
	fileCmd.MarkFlagsOneRequired("seq", "root")
	fileCmd.MarkFlagsMutuallyExclusive("seq", "root")

	Cmd.AddCommand(fileCmd)
}

func statFileDistribution(*cobra.Command, []string) {
	type ShardedFileInfo struct {
		File  *node.FileInfo
		Shard shard.ShardConfig
	}

	infos := mustStatRpc(func(client *node.ZgsClient, ctx context.Context) (*ShardedFileInfo, error) {
		var info ShardedFileInfo
		var err error

		if info.Shard, err = client.GetShardConfig(ctx); err != nil {
			return nil, err
		}

		if len(root) == 0 {
			info.File, err = client.GetFileInfoByTxSeq(ctx, txSeq)
		} else {
			info.File, err = client.GetFileInfo(ctx, common.HexToHash(root))
		}

		return &info, err
	})

	var rpcFailures int
	fileDistribution := make(map[string]int)
	uploadedShardCounter := files.NewShardCounter()

	for _, v := range infos {
		if v.Err != nil {
			rpcFailures++
		} else if v.Data.File == nil {
			fileDistribution["Unsynced"]++
		} else if v.Data.File.Finalized {
			fileDistribution["Uploaded"]++
			uploadedShardCounter.Insert(v.Data.Shard)
		} else {
			fileDistribution["Synced"]++
		}
	}

	fmt.Println("\nRPC Failures:", rpcFailures)

	fmt.Println("\nStatus distribution:")
	for status, count := range fileDistribution {
		fmt.Printf("\t%v: %v\n", status, count)
	}

	prettyPrint(uploadedShardCounter, "\nUploaded shard distribution")
}
