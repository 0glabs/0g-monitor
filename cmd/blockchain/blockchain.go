package blockchain

import (
	"github.com/0glabs/0g-monitor/blockchain"
	"github.com/0glabs/0g-monitor/utils"
	"github.com/spf13/cobra"
)

func NewBlockchainCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "blockchain",
		Short: "run blockchain monitor",
		Run: func(cmd *cobra.Command, args []string) {

			utils.StartDeamon(func() {
				blockchain.MustMonitorFromViper()
			})
		},
	}

	return cmd
}
