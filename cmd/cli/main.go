package main

import (
	"fmt"
	"os"

	"github.com/alexflint/go-arg"
)

type (
	Cmd struct {
		Initialise *InitCmd `arg:"subcommand:init" help:"initialise a new Hiphops project"`
		// Up - command to start hops
		// Down - command to stop hops
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
	arg.MustParse(cmd)

	switch {
	case cmd.Initialise != nil:
		return cmd.Initialise.Run()
	default:
		fmt.Println("No command")
	}

	return nil
}
