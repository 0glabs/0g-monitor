package blockchain

import (
	"sync"

	"github.com/0glabs/0g-monitor/usernode"
	"github.com/0glabs/0g-monitor/utils"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func NewUserNodeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "usernode",
		Short: "run user node monitor",
		Run: func(cmd *cobra.Command, args []string) {
			var wg sync.WaitGroup
			utils.StartAction(usernode.MustMonitorFromViper, &wg)
			logrus.Warn("User node monitoring service started")
			wg.Wait()
		},
	}

	return cmd
}
