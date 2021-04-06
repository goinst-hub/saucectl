package configure

import (
	"errors"
	"fmt"
	serrors "github.com/saucelabs/saucectl/internal/errors"
	"os"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/rs/zerolog/log"
	"github.com/saucelabs/saucectl/cli/command"
	"github.com/saucelabs/saucectl/internal/credentials"
	"github.com/spf13/cobra"
)

var (
	configureUse     = "configure"
	configureShort   = "Configure your Sauce Labs credentials"
	configureLong    = `Persist locally your Sauce Labs credentials`
	configureExample = "saucectl configure"
	cliUsername      = ""
	cliAccessKey     = ""
)

// Command creates the `configure` command
func Command(cli *command.SauceCtlCli) *cobra.Command {
	cmd := &cobra.Command{
		Use:     configureUse,
		Short:   configureShort,
		Long:    configureLong,
		Example: configureExample,
		Run: func(cmd *cobra.Command, args []string) {
			if err := Run(); err != nil {
				log.Err(err).Msg("failed to execute configure command")
				serrors.HandleAndFlush(err)
				os.Exit(1)
			}
		},
	}
	cmd.Flags().StringVarP(&cliUsername, "username", "u", "", "username, available on your sauce labs account")
	cmd.Flags().StringVarP(&cliAccessKey, "accessKey", "a", "", "accessKey, available on your sauce labs account")
	return cmd
}

// explainHowToObtainCredentials explains how to get credentials
func explainHowToObtainCredentials() {
	fmt.Println(`
Don't have an account? Signup here:
https://bit.ly/saucectl-signup

Already have an account? Get your username and access key here:
https://app.saucelabs.com/user-settings`)
}

// interactiveConfiguration expect user to manually type-in its credentials
func interactiveConfiguration() (*credentials.Credentials, error) {
	explainHowToObtainCredentials()
	creds := getDefaultCredentials()

	println("") // visual paragraph break
	qs := []*survey.Question{
		{
			Name: "username",
			Prompt: &survey.Input{
				Message: "SauceLabs username",
				Default: creds.Username,
			},
			Validate: func(val interface{}) error {
				str, ok := val.(string)
				if !ok {
					return errors.New("invalid username. Check it here: https://app.saucelabs.com/user-settings")
				}
				str = strings.TrimSpace(str)
				if str == "" {
					return errors.New("you need to type a username. Get yours here: https://app.saucelabs.com/user-settings")

				}
				return nil
			},
		},
		{
			Name: "accessKey",
			Prompt: &survey.Input{
				Message: "SauceLabs access key",
				Default: creds.AccessKey,
			},
			Validate: func(val interface{}) error {
				str, ok := val.(string)
				if !ok {
					return errors.New("invalid access key. Check it here: https://app.saucelabs.com/user-settings")
				}
				str = strings.TrimSpace(str)
				if str == "" {
					return errors.New("you need to type an access key. Get yours here: https://app.saucelabs.com/user-settings")

				}
				return nil
			},
		},
	}

	if err := survey.Ask(qs, creds); err != nil {
		return nil, err
	}

	fmt.Printf("\n\n")
	return creds, nil
}

// Run starts the configure command
func Run() error {
	var creds *credentials.Credentials
	var err error

	if cliUsername == "" && cliAccessKey == "" {
		creds, err = interactiveConfiguration()
	} else {
		creds = &credentials.Credentials{
			Username:  cliUsername,
			AccessKey: cliAccessKey,
		}
	}
	if err != nil {
		return err
	}

	if !creds.IsValid() {
		log.Error().Msg("The provided credentials appear to be invalid and will NOT be saved.")
		return fmt.Errorf("invalid credentials provided")
	}
	if err := creds.Store(); err != nil {
		return fmt.Errorf("unable to save credentials: %s", err)
	}
	println("You're all set!")
	return nil
}

// getDefaultCredentials returns first the file credentials, then the one founded in the env.
func getDefaultCredentials() *credentials.Credentials {
	fileCreds := credentials.FromFile()
	if fileCreds != nil {
		return fileCreds
	}

	envCreds := credentials.FromEnv()
	if envCreds != nil {
		return envCreds
	}
	return &credentials.Credentials{}
}
