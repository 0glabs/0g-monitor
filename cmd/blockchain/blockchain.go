package blockchain

import (
	"sync"

	"github.com/0glabs/0g-monitor/blockchain"
	"github.com/0glabs/0g-monitor/utils"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func NewBlockchainCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "blockchain",
		Short: "run blockchain monitor",
		Run: func(cmd *cobra.Command, args []string) {
			var wg sync.WaitGroup
			utils.StartAction(blockchain.MustMonitorFromViper, &wg)
			logrus.Warn("Blockchain monitoring service started")
			wg.Wait()
		},
	}

	return cmd
}
