package main

import (
	"fmt"
	"os"

	"github.com/alexflint/go-arg"
)

type (
	Cmd struct {
		Initialise *InitCmd `arg:"subcommand:init" help:"initialise a new Hiphops project"`
		Down       *DownCmd `arg:"subcommand:down" help:"stop Hiphops"`
		Up         *UpCmd   `arg:"subcommand:up" help:"start Hiphops"`
		// Create flow (add empty flow or add from template, default to blank)
	}
)

func main() {
	if err := runCmd(); err != nil {
		fmt.Println("ERROR", err.Error())
		os.Exit(1)
	}

	os.Exit(0)
}

func runCmd() error {
	cmd := &Cmd{}
	p := arg.MustParse(cmd)

	switch {
	case cmd.Initialise != nil:
		return cmd.Initialise.Run()
	case cmd.Up != nil:
		return cmd.Up.Run()
	case cmd.Down != nil:
		return cmd.Down.Run()
	default:
		p.WriteHelp(os.Stdout)
		return nil
	}
}
