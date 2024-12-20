package main

import (
	"github.com/0glabs/0g-monitor/cmd"
	"github.com/sirupsen/logrus"
)

func main() {
	rootCmd := cmd.NewRootCmd()
	if err := rootCmd.Execute(); err != nil {
		logrus.WithError(err).Fatal("Failed to execute command")
	}
}
