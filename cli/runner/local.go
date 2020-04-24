package runner

import (
	"context"
	"errors"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"github.com/saucelabs/saucectl/cli/command"
	"github.com/saucelabs/saucectl/cli/config"
	"github.com/saucelabs/saucectl/cli/docker"
)

type localRunner struct {
	baseRunner
	containerID string
	docker      *docker.Handler
	tmpDir      string
}

func newLocalRunner(c config.JobConfiguration, cli *command.SauceCtlCli) (*localRunner, error) {
	runner := localRunner{}
	runner.cli = cli
	runner.context = context.Background()
	runner.jobConfig = c
	runner.startTime = makeTimestamp()

	var err error
	runner.docker, err = docker.Create()
	if err != nil {
		return nil, err
	}

	runner.tmpDir, err = ioutil.TempDir("", "saucectl")
	if err != nil {
		return nil, err
	}

	return &runner, nil
}

func (r *localRunner) Setup() error {
	err := r.docker.ValidateDependency()
	if err != nil {
		return errors.New("Docker is not installed")
	}

	// check if image is existing
	hasImage, err := r.docker.HasBaseImage(r.context, r.jobConfig.Image.Base)
	if err != nil {
		return err
	}

	// only pull base image if not already installed
	if !hasImage {
		if err := r.docker.PullBaseImage(r.context, "docker.io/"+r.jobConfig.Image.Base); err != nil {
			return err
		}
	}

	container, err := r.docker.StartContainer(r.context, r.jobConfig)
	if err != nil {
		return err
	}
	r.containerID = container.ID

	// wait until Xvfb started
	// ToDo(Christian): make this dynamic
	time.Sleep(1 * time.Second)

	// get runner config
	defer os.RemoveAll(r.tmpDir)
	hostDstPath := filepath.Join(r.tmpDir, filepath.Base(runnerConfigPath))
	if err := r.docker.CopyFromContainer(r.context, container.ID, runnerConfigPath, hostDstPath); err != nil {
		return err
	}

	r.runnerConfig, err = config.NewRunnerConfiguration(hostDstPath)
	if err != nil {
		return err
	}

	if err := r.docker.CopyTestFilesToContainer(r.context, r.containerID, r.jobConfig.Files, r.runnerConfig.TargetDir); err != nil {
		return err
	}
	return nil
}

func (r *localRunner) Run() (int, error) {
	var (
		out, stderr io.Writer
		in          io.ReadCloser
	)
	out = r.cli.Out()
	stderr = r.cli.Out()

	if err := r.cli.In().CheckTty(false, true); err != nil {
		return 1, err
	}

	createResp, attachResp, err := r.docker.ExecuteTest(r.context, r.containerID)
	if err != nil {
		return 1, err
	}

	defer attachResp.Close()

	errCh := make(chan error, 1)
	go func() {
		defer close(errCh)
		errCh <- func() error {
			streamer := ioStreamer{
				streams:      r.cli,
				inputStream:  in,
				outputStream: out,
				errorStream:  stderr,
				resp:         *attachResp,
				detachKeys:   "",
			}

			return streamer.stream(r.context)
		}()
	}()

	if err := <-errCh; err != nil {
		return 1, err
	}

	exitCode, err := r.docker.ExecuteInspect(r.context, createResp.ID)
	if err != nil {
		return 1, err
	}

	return exitCode, nil
}

func (r *localRunner) Teardown(logDir string) error {
	for _, containerSrcPath := range logFiles {
		file := filepath.Base(containerSrcPath)
		hostDstPath := filepath.Join(logDir, file)
		if err := r.docker.CopyFromContainer(r.context, r.containerID, containerSrcPath, hostDstPath); err != nil {
			continue
		}
	}

	if err := r.docker.ContainerStop(r.context, r.containerID); err != nil {
		return err
	}

	if err := r.docker.ContainerRemove(r.context, r.containerID); err != nil {
		return err
	}

	return nil
}