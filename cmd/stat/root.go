package stat

import (
	"context"
	"fmt"
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
	Cmd.PersistentFlags().StringVar(&indexerURL, "indexer", "", "Indexer URL to discover storage nodes")
	Cmd.MarkPersistentFlagRequired("indexer")
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
	logrus.Info("Begin to query shard configs in parallel")
	result, err := storage.ParallelQueryRpc(context.Background(), nodes, statRpcFunc, serialOpt)
	if err != nil {
		logrus.WithError(err).Fatal("Failed to query shard configs in parallel")
	}

	return result
}
