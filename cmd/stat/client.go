package stat

import (
	"context"
	"fmt"

	"github.com/0glabs/0g-storage-client/node"
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

	statPeers("Seen IP Count Stat:", peers, func(pi *node.PeerInfo) any { return len(pi.SeenIps) })
	statPeers("OS Stat:", peers, func(pi *node.PeerInfo) any { return pi.Client.OS })
	statPeers("Protocol Stat:", peers, func(pi *node.PeerInfo) any { return pi.Client.Protocol })
	statPeers("Version Stat:", peers, func(pi *node.PeerInfo) any { return pi.Client.Version })
}

func statPeers(label string, peers map[string]*node.PeerInfo, statFunc func(*node.PeerInfo) any) {
	result := make(map[any]int)

	for _, v := range peers {
		val := statFunc(v)
		result[val]++
	}

	fmt.Println(label)
	for val, count := range result {
		fmt.Printf("\t%v: %v\n", val, count)
	}
}
