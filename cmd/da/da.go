package da

import (
	"sync"

	"github.com/0glabs/0g-monitor/da"
	"github.com/0glabs/0g-monitor/utils"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func NewDaCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "da",
		Short: "run da monitor",
		RunE: func(cmd *cobra.Command, args []string) error {
			var wg sync.WaitGroup
			utils.StartAction(da.MustMonitorFromViper, &wg)
			logrus.Warn("DA monitoring service started")
			wg.Wait()
			return nil
		},
	}

	return cmd
}
