package stat

import (
	"context"
	"fmt"

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
	files := mustStatRpc(func(client *node.ZgsClient, ctx context.Context) (*node.FileInfo, error) {
		if len(root) == 0 {
			return client.GetFileInfoByTxSeq(ctx, txSeq)
		}

		return client.GetFileInfo(ctx, common.HexToHash(root))
	})

	var rpcFailures int
	fileDistribution := make(map[string]int)

	for _, v := range files {
		if v.Err != nil {
			rpcFailures++
		} else if v.Data == nil {
			fileDistribution["Unsynced"]++
		} else if v.Data.Finalized {
			fileDistribution["Uploaded"]++
		} else {
			fileDistribution["Synced"]++
		}
	}

	fmt.Println()
	fmt.Println("RPC Failures:", rpcFailures)
	fmt.Println("File distribution:")
	for status, count := range fileDistribution {
		fmt.Printf("\t%v: %v\n", status, count)
	}
}
