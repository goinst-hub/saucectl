package saucecloud

import (
	"context"
	"errors"
	"fmt"

	"github.com/rs/zerolog/log"
	"github.com/saucelabs/saucectl/internal/cypress"
	"github.com/saucelabs/saucectl/internal/cypress/suite"
	"github.com/saucelabs/saucectl/internal/framework"
	"github.com/saucelabs/saucectl/internal/job"
	"github.com/saucelabs/saucectl/internal/msg"
)

// CypressRunner represents the Sauce Labs cloud implementation for cypress.
type CypressRunner struct {
	CloudRunner
	Project cypress.Project
}

// RunProject runs the tests defined in cypress.Project.
func (r *CypressRunner) RunProject() (int, error) {
	var deprecationMessage string
	exitCode := 1

	cyVersion := r.Project.GetVersion()
	m, err := r.MetadataSearchStrategy.Find(context.Background(), r.MetadataService, cypress.Kind, cyVersion)
	if err != nil {
		r.logFrameworkError(err)
		return exitCode, err
	}
	r.Project.SetVersion(m.FrameworkVersion)
	if r.Project.GetRunnerVersion() == "" {
		r.Project.SetRunnerVersion(m.CloudRunnerVersion)
	}

	if m.IsDeprecated() && !m.IsFlaggedForRemoval() {
		deprecationMessage = r.deprecationMessage(cypress.Kind, cyVersion, m.RemovalDate)
		fmt.Print(deprecationMessage)
	}
	if m.IsFlaggedForRemoval() {
		deprecationMessage = r.flaggedForRemovalMessage(cypress.Kind, cyVersion)
		fmt.Print(deprecationMessage)
	}

	for _, s := range r.Project.GetSuites() {
		if s.PlatformName != "" && !framework.HasPlatform(m, s.PlatformName) {
			msg.LogUnsupportedPlatform(s.PlatformName, framework.PlatformNames(m.Platforms))
			return 1, errors.New("unsupported platform")
		}
	}

	if err := r.validateTunnel(r.Project.GetSauceCfg().Tunnel.Name, r.Project.GetSauceCfg().Tunnel.Owner, r.Project.IsDryRun()); err != nil {
		return 1, err
	}

	fileURIs, err := r.remoteArchiveProject(r.Project, r.Project.GetRootDir(), r.Project.GetSauceCfg().Sauceignore, r.Project.IsDryRun())
	if err != nil {
		return exitCode, err
	}

	if r.Project.IsDryRun() {
		log.Info().Msgf("The following test suites would have run: [%s].", r.Project.GetSuiteNames())
		return 0, nil
	}

	passed := r.runSuites(fileURIs)
	if passed {
		exitCode = 0
	}

	if deprecationMessage != "" {
		fmt.Print(deprecationMessage)
	}

	return exitCode, nil
}

// checkCypressVersion do several checks before running Cypress tests.
func (r *CypressRunner) checkCypressVersion() error {
	if r.Project.GetVersion() == "" {
		return fmt.Errorf("missing cypress version. Check available versions here: https://docs.saucelabs.com/dev/cli/saucectl/#supported-frameworks-and-browsers")
	}
	return nil
}

func (r *CypressRunner) runSuites(fileURIs []string) bool {
	sigChan := r.registerSkipSuitesOnSignal()
	defer unregisterSignalCapture(sigChan)
	jobOpts, results, err := r.createWorkerPool(r.Project.GetSauceCfg().Concurrency, r.Project.GetSauceCfg().Retries)
	if err != nil {
		return false
	}
	defer close(results)

	suites := r.Project.GetSuites()
	if r.Project.GetSauceCfg().LaunchOrder != "" {
		history, err := r.getHistory(r.Project.GetSauceCfg().LaunchOrder)
		if err != nil {
			log.Warn().Err(err).Msg(msg.RetrieveJobHistoryError)
		} else {
			suites = suite.SortByHistory(suites, history)
		}
	}
	// Submit suites to work on.
	go func() {
		for _, s := range suites {
			jobOpts <- job.StartOptions{
				ConfigFilePath:   r.Project.GetCfgPath(),
				CLIFlags:         r.Project.GetCLIFlags(),
				DisplayName:      s.Name,
				Timeout:          s.Timeout,
				App:              fileURIs[0],
				OtherApps:        fileURIs[1:],
				Suite:            s.Name,
				Framework:        "cypress",
				FrameworkVersion: r.Project.GetVersion(),
				BrowserName:      s.Browser,
				BrowserVersion:   s.BrowserVersion,
				PlatformName:     s.PlatformName,
				Name:             s.Name,
				Build:            r.Project.GetSauceCfg().Metadata.Build,
				Tags:             r.Project.GetSauceCfg().Metadata.Tags,
				Tunnel: job.TunnelOptions{
					ID:     r.Project.GetSauceCfg().Tunnel.Name,
					Parent: r.Project.GetSauceCfg().Tunnel.Owner,
				},
				ScreenResolution: s.ScreenResolution,
				RunnerVersion:    r.Project.GetRunnerVersion(),
				Experiments:      r.Project.GetSauceCfg().Experiments,
				Attempt:          0,
				Retries:          r.Project.GetSauceCfg().Retries,
				TimeZone:         s.TimeZone,
				Visibility:       r.Project.GetSauceCfg().Visibility,
				PassThreshold:    s.PassThreshold,
			}
		}
	}()

	return r.collectResults(r.Project.GetArtifactsCfg().Download, results, r.Project.GetSuiteCount())
}
