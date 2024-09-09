package stat

import (
	"time"

	"github.com/0glabs/0g-storage-client/indexer"
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
