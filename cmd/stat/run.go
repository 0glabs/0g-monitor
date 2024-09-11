package stat

import (
	"github.com/0glabs/0g-monitor/storage/files"
	"github.com/spf13/cobra"
)

var (
	runCmd = &cobra.Command{
		Use:   "run",
		Short: "Start service to statistic file status in storage node network",
		Run:   run,
	}
)

func init() {
	Cmd.AddCommand(runCmd)
}

func run(*cobra.Command, []string) {
	files.MustCollectFromViper()
}
