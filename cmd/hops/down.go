package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
)

type DownCmd struct {
	Dir string `arg:"positional" default:"." help:"path to Hiphops dir - defaults to current directory"`
}

func (d *DownCmd) Run() error {
	ctx := context.Background()

	cmd := exec.CommandContext(
		ctx,
		"docker", "compose",
		"-p", "hiphops",
		"down",
	)

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("unable to run docker compose down: %w", err)
	}
	return nil
}
