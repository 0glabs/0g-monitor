package cmd

import (
	"github.com/0glabs/0g-monitor/cmd/blockchain"
	"github.com/0glabs/0g-monitor/cmd/da"
	"github.com/0glabs/0g-monitor/cmd/stat"
	"github.com/0glabs/0g-monitor/cmd/storage"
	"github.com/Conflux-Chain/go-conflux-util/config"
	"github.com/Conflux-Chain/go-conflux-util/log"
	"github.com/spf13/cobra"
)

func init() {
	cobra.OnInitialize(func() {
		config.MustInit("ZG_MONITOR")
	})
}

func NewRootCmd() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "0g-monitor",
		Short: "Daemon to monitor all 0G service status",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			return nil
		},
	}

	rootCmd.AddCommand(
		blockchain.NewBlockchainCmd(),
		storage.NewStorageCmd(),
		da.NewDaCmd(),
		stat.Cmd,
	)

	log.BindFlags(rootCmd)

	return rootCmd
}
