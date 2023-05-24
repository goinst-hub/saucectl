package apit

import (
	"errors"
	"time"

	"github.com/saucelabs/saucectl/internal/credentials"
	"github.com/saucelabs/saucectl/internal/http"
	"github.com/saucelabs/saucectl/internal/region"
	"github.com/spf13/cobra"
)

var (
	apitesterClient http.APITester
)

func Command(preRun func(cmd *cobra.Command, args []string)) *cobra.Command {
	var regio string

	cmd := &cobra.Command{
		Use:          "apit",
		Short:        "Commands for interacting with API Testing",
		SilenceUsage: true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if preRun != nil {
				preRun(cmd, args)
			}

			if region.FromString(regio) == region.None {
				return errors.New("invalid region")
			}

			creds := credentials.Get()
			url := region.FromString(regio).APIBaseURL()

			apitesterClient = http.NewAPITester(url, creds.Username, creds.AccessKey, 15*time.Minute)

			return nil
		},
	}

	flags := cmd.PersistentFlags()
	flags.StringVarP(&regio, "region", "r", "us-west-1", "The Sauce Labs region. Options: us-west-1, eu-central-1.")

	cmd.AddCommand(
		VaultCommand(cmd.PersistentPreRunE),
	)
	return cmd
}