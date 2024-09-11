package files

import (
	"context"
	"time"

	"github.com/0glabs/0g-storage-client/common/parallel"
	"github.com/0glabs/0g-storage-client/common/util"
	"github.com/0glabs/0g-storage-client/indexer"
	"github.com/0glabs/0g-storage-client/node"
	"github.com/Conflux-Chain/go-conflux-util/store/mysql"
	"github.com/Conflux-Chain/go-conflux-util/viper"
	providers "github.com/openweb3/go-rpc-provider/provider_wrapper"
	"github.com/openweb3/web3go"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

var (
	defaultProviderOption = providers.Option{
		RequestTimeout: 3 * time.Second,
	}

	defaultIndexerProviderOption = indexer.IndexerClientOption{
		ProviderOption: defaultProviderOption,
	}

	logger *logrus.Entry
)

type Config struct {
	Indexer                string
	Fullnode               string
	DiscoveryPeersInterval time.Duration `default:"10m"`
	Routines               int           `default:"500"`
	Mysql                  mysql.Config
}

func MustCollectFromViper() {
	var config Config
	viper.MustUnmarshalKey("storage.files", &config)
	MustCollect(config)
}

func MustCollect(config Config) {
	if err := Collect(config); err != nil {
		logrus.WithError(err).Fatal("Failed to statistics file status")
	}
}

func Collect(config Config) error {
	logger = logrus.WithField("module", "zgs.stat.files")

	// create indexer client
	indexerClient, err := indexer.NewClient(config.Indexer, defaultIndexerProviderOption)
	if err != nil {
		return errors.WithMessage(err, "Failed to create indexer client")
	}
	defer indexerClient.Close()
	logger.WithField("url", config.Indexer).Info("Dailed to indexer")

	// create W3 client
	w3Client, err := web3go.NewClient(config.Fullnode)
	if err != nil {
		return errors.WithMessage(err, "Failed to create W3 client")
	}
	defer w3Client.Close()
	logger.WithField("url", config.Fullnode).Info("Dailed to 0gchain")

	// discover peers
	logger.Debug("Begin to initialize discovery")
	discovery, err := NewDiscovery(indexerClient, config)
	if err != nil {
		return errors.WithMessage(err, "Failed to create discovery")
	}
	go util.Schedule(discovery.Discover, config.DiscoveryPeersInterval, "Failed to discovery peers")
	logger.WithField("interval", config.DiscoveryPeersInterval).Info("Scheduled to discover peers")

	// sample txSeq to statistic
	logger.Debug("Begin to initialize sampler")
	sampler, err := NewSampler(indexerClient, w3Client)
	if err != nil {
		return errors.WithMessage(err, "Failed to create sampler")
	}
	go util.Schedule(sampler.Update, 5*time.Second, "Failed to update max tx seq")
	logger.WithField("max", sampler.maxTxSeq.Load()).Info("Begin to update max tx seq from flow contract")

	// create store
	store := MustNewStore(config.Mysql)
	logger.Info("Database initialized")

	collect(config, discovery, sampler, store)

	return nil
}

func collect(config Config, discovery *Discovery, sampler *Sampler, store *Store) {
	for {
		peers, shards := discovery.GetPeers()

		txSeq := sampler.Random()

		rpcFunc := func(client *node.ZgsClient, ctx context.Context) (*node.FileInfo, error) {
			return client.GetFileInfoByTxSeq(ctx, txSeq)
		}

		logger.WithField("txSeq", txSeq).Debug("Begin to statistic file status")
		result := parallel.QueryZgsRpc(context.Background(), peers, rpcFunc, parallel.RpcOption{
			Parallel: parallel.SerialOption{
				Routines: config.Routines,
			},
			Provider: defaultProviderOption,
		})

		counter := NewShardCounter()
		var aaa int

		for peer, rpcResult := range result {
			if rpcResult.Err == nil && rpcResult.Data != nil && rpcResult.Data.Finalized {
				counter.Insert(shards[peer])
				aaa++
			}
		}

		replica := counter.Replica()
		logger.WithFields(logrus.Fields{
			"txSeq":   txSeq,
			"replica": replica,
			"count":   aaa,
		}).Debug("Completed to statistic file status")

		model := Replica{
			TxSeq:   txSeq,
			Replica: replica,
		}

		if err := store.Upsert(&model); err != nil {
			logger.WithError(err).Warn("Failed to upsert replica in db")
		}
	}
}
