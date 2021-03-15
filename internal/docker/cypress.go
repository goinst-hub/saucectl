package docker

import (
	"context"

	"github.com/rs/zerolog/log"

	"github.com/saucelabs/saucectl/internal/config"
	"github.com/saucelabs/saucectl/internal/cypress"
	"github.com/saucelabs/saucectl/internal/framework"
)

// CypressRunner represents the docker implementation of a test runner.
type CypressRunner struct {
	ContainerRunner
	Project cypress.Project
}

// NewCypress creates a new CypressRunner instance.
func NewCypress(c cypress.Project, ms framework.MetadataService) (*CypressRunner, error) {
	r := CypressRunner{
		Project: c,
		ContainerRunner: ContainerRunner{
			Ctx:             context.Background(),
			docker:          nil,
			containerConfig: &containerConfig{},
			Framework: framework.Framework{
				Name:    c.Kind,
				Version: c.Cypress.Version,
			},
			FrameworkMeta:  ms,
			ShowConsoleLog: c.ShowConsoleLog,
		},
	}

	var err error
	r.docker, err = Create()
	if err != nil {
		return nil, err
	}

	return &r, nil
}

// RunProject runs the tests defined in config.Project.
func (r *CypressRunner) RunProject() (int, error) {
	files := []string{
		r.Project.Cypress.ConfigFile,
		r.Project.Cypress.ProjectPath,
	}

	if r.Project.Cypress.EnvFile != "" {
		files = append(files, r.Project.Cypress.EnvFile)
	}

	if r.Project.RootDir != "" {
		files = append(files, r.Project.RootDir)
	}

	if r.Project.Sauce.Concurrency > 1 {
		log.Info().Msg("concurrency > 1: forcing file transfer mode to use 'copy'.")
		r.Project.Docker.FileTransfer = config.DockerFileCopy
	}

	if err := r.fetchImage(&r.Project.Docker); err != nil {
		return 1, err
	}

	containerOpts, results := r.createWorkerPool(r.Project.Sauce.Concurrency)
	defer close(results)

	go func() {
		for _, suite := range r.Project.Suites {
			containerOpts <- containerStartOptions{
				Docker:      r.Project.Docker,
				BeforeExec:  r.Project.BeforeExec,
				Project:     r.Project,
				SuiteName:   suite.Name,
				Environment: suite.Config.Env,
				Files:       files,
				Sauceignore: r.Project.Sauce.Sauceignore,
			}
		}
		close(containerOpts)
	}()

	hasPassed := r.collectResults(results, len(r.Project.Suites))
	if !hasPassed {
		return 1, nil
	}
	return 0, nil
}
