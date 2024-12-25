package blockchain

import (
	"github.com/0glabs/0g-monitor/blockchain"
	"github.com/0glabs/0g-monitor/utils"
	"github.com/spf13/cobra"
)

// const (
// 	FlagConfig = "config"
// )

func NewBlockchainCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "blockchain",
		Short: "run blockchain monitor",
		Run: func(cmd *cobra.Command, args []string) {
			// load config
			// create monitor
			utils.StartDeamon(func() {
				blockchain.MustMonitorFromViper()
			})
		},
	}

	// cmd.Flags().String(FlagConfig, "", "path to config file")

	// _ = cmd.MarkFlagRequired(FlagConfig)
	return cmd
}
