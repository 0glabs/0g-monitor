package stat

import (
	"context"
	"fmt"

	"github.com/0glabs/0g-storage-client/node"
	"github.com/emirpasic/gods/maps/treemap"
	"github.com/emirpasic/gods/utils"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var (
	zgsNodeUrl string

	clientCmd = &cobra.Command{
		Use:   "client",
		Short: "Statistic client versions of storage node network",
		Run:   statClient,
	}
)

func init() {
	clientCmd.Flags().StringVar(&zgsNodeUrl, "url", "", "Admin RPC URL of storage node. If empty, use trusted node from indexer")

	Cmd.AddCommand(clientCmd)
}

func statClient(*cobra.Command, []string) {
	if len(zgsNodeUrl) == 0 {
		indexer := mustNewIndexerClient()
		defer indexer.Close()

		nodes, err := indexer.GetShardedNodes(context.Background())
		if err != nil {
			logrus.WithError(err).Fatal("Failed to retrieve sharded nodes from indexer")
		}

		if len(nodes.Trusted) == 0 {
			logrus.Fatal("No trusted nodes found from indexer")
		}

		zgsNodeUrl = nodes.Trusted[0].URL
		logrus.WithField("url", zgsNodeUrl).Info("Use trusted node from indexer")
	}

	client, err := node.NewAdminClient(zgsNodeUrl)
	if err != nil {
		logrus.WithError(err).Fatal("Failed to dail to storage node")
	}
	defer client.Close()

	peers, err := client.GetPeers(context.Background())
	if err != nil {
		logrus.WithError(err).Fatal("Failed to retrieve peers from storage node")
	}

	fmt.Println("Total peers:", len(peers))

	statPeers("Seen IP Count Stat:", peers, func(pi *node.PeerInfo) any { return len(pi.SeenIps) }, utils.IntComparator)
	statPeers("OS Stat:", peers, func(pi *node.PeerInfo) any { return pi.Client.OS }, utils.StringComparator)
	statPeers("Protocol Stat:", peers, func(pi *node.PeerInfo) any { return pi.Client.Protocol }, utils.StringComparator)
	statPeers("Version Stat:", peers, func(pi *node.PeerInfo) any { return pi.Client.Version }, CompareGitVersionString)
}

func statPeers(label string, peers map[string]*node.PeerInfo, statFunc func(*node.PeerInfo) any, comparator utils.Comparator) {
	result := treemap.NewWith(comparator)

	for _, v := range peers {
		key := statFunc(v)

		if val, ok := result.Get(key); ok {
			result.Put(key, val.(int)+1)
		} else {
			result.Put(key, 1)
		}
	}

	fmt.Println(label)

	result.Each(func(key, value interface{}) {
		fmt.Printf("\t%v: %v\n", key, value)
	})
}
