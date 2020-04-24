package run

import (
	"os"
	"path/filepath"

	"github.com/saucelabs/saucectl/cli/command"
	"github.com/saucelabs/saucectl/cli/config"
	"github.com/saucelabs/saucectl/cli/runner"
	"github.com/spf13/cobra"
)

var (
	runUse     = "run ./.sauce/config.yaml"
	runShort   = "Run a test on Sauce Labs"
	runLong    = `Some long description`
	runExample = "saucectl run ./.sauce/config.yaml"

	defaultLogFir = "<cwd>/logs"

	cfgFilePath string
	cfgLogDir   string
)

// NewRunCommand creates the `run` command
func NewRunCommand(cli *command.SauceCtlCli) *cobra.Command {
	cmd := &cobra.Command{
		Use:     runUse,
		Short:   runShort,
		Long:    runLong,
		Example: runExample,
		Run: func(cmd *cobra.Command, args []string) {
			cli.Logger.Info().Msg("Start Run Command")
			checkErr(Run(cmd, cli, args))
			os.Exit(0)
		},
	}

	cmd.Flags().StringVarP(&cfgFilePath, "config", "c", "", "config file (e.g. ./.sauce/config.yaml")
	cmd.Flags().StringVarP(&cfgLogDir, "logDir", "l", defaultLogFir, "log path")

	return cmd
}

func checkErr(e error) {
	if e != nil {
		panic(e)
	}
}

// Run runs the command
func Run(cmd *cobra.Command, cli *command.SauceCtlCli, args []string) error {
	// Todo(Christian) write argument parser/validator
	if cfgLogDir == defaultLogFir {
		pwd, _ := os.Getwd()
		cfgLogDir = filepath.Join(pwd, "logs")
	}

	cli.Logger.Info().Msg("Read config file")
	configObject, err := config.NewJobConfiguration(cfgFilePath)
	if err != nil {
		return err
	}

	tr, err := runner.New(configObject, cli)
	if err != nil {
		return err
	}

	if err := tr.Setup(); err != nil {
		return err
	}

	exitCode, err := tr.Run()
	if err != nil {
		return err
	}

	if err != tr.Teardown(cfgLogDir) {
		return err
	}

	cli.Logger.Info().
		Int("ExitCode", exitCode).
		Msg("Command Finished")

	return nil
}