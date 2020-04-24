package commands

import (
	"github.com/saucelabs/saucectl/cli/command"
	"github.com/saucelabs/saucectl/cli/command/run"
	"github.com/spf13/cobra"
)

// AddCommands attaches commands to cli
func AddCommands(cmd *cobra.Command, cli *command.SauceCtlCli) {
	cmd.AddCommand(
		run.NewRunCommand(cli),
		// logs.NewLogsCommand(cli),
	)
}