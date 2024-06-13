package cmd

import (
	"sync"

	"github.com/0glabs/0g-monitor/blockchain"
	"github.com/0glabs/0g-monitor/storage"
	"github.com/Conflux-Chain/go-conflux-util/config"
	"github.com/Conflux-Chain/go-conflux-util/log"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "0g-monitor",
	Short: "Daemon to monitor all 0G service status",
	Run:   start,
}

func init() {
	cobra.OnInitialize(initConfig)

	log.BindFlags(rootCmd)
}

func initConfig() {
	config.MustInit("ZG_MONITOR")
}

func start(*cobra.Command, []string) {
	var wg sync.WaitGroup

	startAction(blockchain.MustMonitorFromViper, &wg)
	startAction(storage.MustMonitorFromViper, &wg)

	logrus.Warn("Monitoring service started")

	wg.Wait()
}

func startAction(action func(), wg *sync.WaitGroup) {
	wg.Add(1)

	go func() {
		defer wg.Done()
		action()
	}()
}

// Execute is the command line entrypoint.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		logrus.WithError(err).Fatal("Failed to execute command")
	}
}
