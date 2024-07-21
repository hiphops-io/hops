package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"

	"github.com/hiphops-io/hops/config"
)

type UpCmd struct {
	Dir string `arg:"positional" default:"." help:"path to Hiphops dir - defaults to current directory"`
}

func (u *UpCmd) Run() error {
	ctx := context.Background()
	cfg := config.NewConfig(u.Dir, "")

	cmd := exec.CommandContext(
		ctx,
		"docker", "compose",
		"-p", "hiphops",
		"--project-directory", u.Dir,
		"-f", cfg.DockerComposePath(),
		"up",
		"-d", "--wait",
		"--remove-orphans",
	)
	// TODO: In detached mode, we don't get to see errors within the container
	// if any (e.g. CMD failed).
	// We should find a way to detect failures and automatically show the logs
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("unable to run docker compose up: %w", err)
	}
	return nil
}
