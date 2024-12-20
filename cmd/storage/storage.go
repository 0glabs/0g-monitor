package storage

import (
	"sync"

	"github.com/0glabs/0g-monitor/storage"
	"github.com/0glabs/0g-monitor/utils"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func NewStorageCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "storage",
		Short: "run storage monitor",
		Run: func(cmd *cobra.Command, args []string) {
			var wg sync.WaitGroup
			utils.StartAction(storage.MustMonitorFromViper, &wg)
			logrus.Warn("Storage monitoring service started")
			wg.Wait()
		},
	}

	return cmd
}
