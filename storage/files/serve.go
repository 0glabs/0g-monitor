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
	"github.com/ethereum/go-ethereum/common"
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
	RpcBatch               uint64        `default:"200"`
	Mysql                  mysql.Config

	// be be removed once storage node return flow contract address
	Flow string
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

	// create store
	store := MustNewStore(config.Mysql)
	logger.Info("Database initialized")

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
	sampler, err := NewSamplerWithFlow(common.HexToAddress(config.Flow), w3Client)
	if err != nil {
		return errors.WithMessage(err, "Failed to create sampler")
	}
	go util.Schedule(sampler.Update, 5*time.Second, "Failed to update max tx seq")
	logger.WithField("max", sampler.maxTxSeq.Load()).Info("Begin to update max tx seq from flow contract")

	collect(config, discovery, sampler, store)

	return nil
}

func collect(config Config, discovery *Discovery, sampler *Sampler, store *Store) {
	var next uint64

	// continue from break point in db
	max, err := store.MaxTxSeq()
	if err != nil {
		logger.WithError(err).Warn("Failed to load max tx seq from store")
	} else if max.Valid {
		next = uint64(max.Int64 + 1)
	}
	logger.WithField("next", next).Info("Begin to collect file status")

	txSeqBuf := make([]uint64, config.RpcBatch)

	for {
		maxTxSeq := sampler.maxTxSeq.Load()

		// start from 0 again
		if next > maxTxSeq {
			next = 0
			logrus.WithField("max", maxTxSeq).Warn("Statistic file status from seq 0 again")
		}

		batchSize := min(config.RpcBatch, maxTxSeq+1-next)
		for i := uint64(0); i < batchSize; i++ {
			txSeqBuf[i] = next + i
		}

		rpcFunc := func(client *node.ZgsClient, ctx context.Context) (*batchGetFileInfoResult, error) {
			return batchGetFileInfo(client.Provider, ctx, txSeqBuf[:batchSize]...)
		}

		peers, shards := discovery.GetPeers()

		logger.WithFields(logrus.Fields{
			"start": next,
			"end":   next + batchSize - 1,
			"peers": len(peers),
		}).Debug("Begin to statistic file status")

		start := time.Now()
		result := parallel.QueryZgsRpc(context.Background(), peers, rpcFunc, parallel.RpcOption{
			Parallel: parallel.SerialOption{
				Routines: config.Routines,
			},
			Provider: defaultProviderOption,
		})

		files := make([]*File, batchSize)

		for i := 0; i < int(batchSize); i++ {
			counter := NewShardCounter()
			file := File{
				TxSeq: txSeqBuf[i],
			}

			for peer, rpcResult := range result {
				if rpcResult.Err != nil || rpcResult.Data.errors[i] != nil {
					file.NumErrors++
				} else if rpcResult.Data.files[i] == nil {
					file.NumNotSync++
				} else if rpcResult.Data.files[i].Finalized {
					file.NumUploaded++
					counter.Insert(shards[peer])
				} else {
					file.NumSynced++
				}
			}

			file.NumReplica = counter.Replica()
			files[i] = &file
		}

		if err := store.Upsert(files...); err != nil {
			logger.WithError(err).Warn("Failed to upsert file status in db")
		}

		logger.WithFields(logrus.Fields{
			"start":   next,
			"end":     next + batchSize - 1,
			"peers":   len(peers),
			"elapsed": time.Since(start),
		}).Debug("Completed to statistic file status")

		next += batchSize
	}
}
