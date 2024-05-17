package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/alexflint/go-arg"
	"github.com/hiphops-io/hops/config"
	"github.com/mitchellh/go-homedir"
)

type Cmd struct {
	Directory   string          `arg:"-d,--dir,env:HIPHOPS_DIR" default:"." help:"path to Hiphops dir" placeholder:"HIPHOPS_DIR"`
	Tag         string          `arg:"-t,--tag,env:HIPHOPS_TAG" help:"config overlay to apply e.g. 'dev'"`
	Healthcheck *HealthcheckCMD `arg:"subcommand:health" help:"CLI based healthcheck for docker-compose"`
}

type HealthcheckCMD struct{}

func (Cmd) Description() string {
	return "Hops provides core orchestration, event handling, and messaging for Hiphops"
}

func Run() error {
	cmd := Cmd{}
	arg.MustParse(&cmd)

	if cmd.Healthcheck != nil {
		return healthCheck()
	}

	if expanded, err := homedir.Expand(cmd.Directory); err != nil {
		return err
	} else {
		cmd.Directory = expanded
	}

	config, err := config.LoadConfig(cmd.Directory, cmd.Tag)
	if err != nil {
		return err
	}

	return Start(config)
}

func healthCheck() error {
	fmt.Println("Running CLI healthcheck")
	// TODO: If the httpserver address becomes configurable, then this must match it
	resp, err := http.Get("http://127.0.0.1:8080/health")
	if err != nil {
		return err
	}

	if resp.StatusCode > 299 {
		return fmt.Errorf("healthcheck failed with status code: %d", resp.StatusCode)
	}

	return nil
}

func main() {
	if err := Run(); err != nil {
		fmt.Println("ERROR", err.Error())
		os.Exit(1)
		return
	}

	os.Exit(0)
}
