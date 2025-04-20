package config

import (
	"github.com/a13labs/m3uproxy/cli/cmd"

	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Control the M3U proxy config",
	Long:  ``,
}

func init() {
	cmd.RootCmd.AddCommand(configCmd)
}
