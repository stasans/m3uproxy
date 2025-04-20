package diags

import (
	"github.com/a13labs/m3uproxy/cli/cmd"

	"github.com/spf13/cobra"
)

var diagsCmd = &cobra.Command{
	Use:   "diags",
	Short: "Run diagnostics on M3U proxy server",
	Long:  ``,
}

func init() {
	cmd.RootCmd.AddCommand(diagsCmd)
}
