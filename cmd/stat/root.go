package stat

import (
	"context"
	"time"

	"github.com/0glabs/0g-monitor/storage"
	"github.com/0glabs/0g-storage-client/indexer"
	"github.com/0glabs/0g-storage-client/node"
	"github.com/Conflux-Chain/go-conflux-util/parallel"
	providers "github.com/openweb3/go-rpc-provider/provider_wrapper"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var (
	indexerURL string
	serialOpt  parallel.SerialOption

	Cmd = &cobra.Command{
		Use:   "stat",
		Short: "Statistics subcommands",
	}
)

func init() {
	Cmd.PersistentFlags().StringVar(&indexerURL, "indexer", "https://rpc-storage-testnet-turbo.0g.ai", "Indexer URL to discover storage nodes")
	// Cmd.MarkPersistentFlagRequired("indexer")
	Cmd.PersistentFlags().IntVar(&serialOpt.Routines, "threads", 0, "Number of threads to query RPC")
}

func mustNewIndexerClient() *indexer.Client {
	option := indexer.IndexerClientOption{
		ProviderOption: providers.Option{
			RequestTimeout: 3 * time.Second,
		},
	}

	client, err := indexer.NewClient(indexerURL, option)
	if err != nil {
		logrus.WithError(err).WithField("url", indexerURL).Fatal("Failed to connect to indexer")
	}

	return client
}

func mustStatRpc[T any](statRpcFunc func(*node.ZgsClient, context.Context) (T, error)) map[string]*storage.QueryRpcResult[T] {
	// dail to indexer
	client := mustNewIndexerClient()
	defer client.Close()
	logrus.Info("Dailed to indexer")

	// retrieve discovered nodes from indexer
	shardedNodes, err := client.GetShardedNodes(context.Background())
	if err != nil {
		logrus.WithError(err).Fatal("Failed to retrieve sharded nodes")
	}
	logrus.WithFields(logrus.Fields{
		"trusted":    len(shardedNodes.Trusted),
		"discovered": len(shardedNodes.Discovered),
	}).Info("Succeeded to retrieve sharded nodes")

	// call rpc in parallel
	nodes := make([]string, 0, len(shardedNodes.Discovered))
	for _, v := range shardedNodes.Discovered {
		nodes = append(nodes, v.URL)
	}
	logrus.Info("Begin to query RPC in parallel")
	start := time.Now()
	result, err := storage.ParallelQueryRpc(context.Background(), nodes, statRpcFunc, serialOpt)
	if err != nil {
		logrus.WithError(err).Fatal("Failed to query RPC in parallel")
	}
	logrus.WithField("elapsed", time.Since(start)).Info("Completed to query RPC in parallel")

	return result
}
