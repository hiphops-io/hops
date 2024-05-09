package main

import (
	"os"

	"github.com/alexflint/go-arg"
	"github.com/hiphops-io/hops/config"
	"github.com/mitchellh/go-homedir"
)

type CLIArgs struct {
	Directory string `arg:"-d,--dir,env:HIPHOPS_DIR" default:"." help:"path to Hiphops dir" placeholder:"HIPHOPS_DIR"`
	Tag       string `arg:"-t,--tag,env:HIPHOPS_TAG" help:"config overlay to apply e.g. 'dev'"`
}

func (CLIArgs) Description() string {
	return "Hops provides core orchestration, event handling, and messaging for Hiphops"
}

func Run() error {
	cli := CLIArgs{}
	arg.MustParse(&cli)

	if expanded, err := homedir.Expand(cli.Directory); err != nil {
		return err
	} else {
		cli.Directory = expanded
	}

	config, err := config.LoadConfig(cli.Directory, cli.Tag)
	if err != nil {
		return err
	}

	return Start(config)
}

func main() {
	if err := Run(); err != nil {
		os.Exit(1)
	}
}
