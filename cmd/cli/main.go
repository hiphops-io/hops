package main

import (
	"fmt"
	"os"

	"github.com/alexflint/go-arg"
	_ "github.com/joho/godotenv/autoload"
)

type (
	Cmd struct {
		Down       *DownCmd `arg:"subcommand:down" help:"stop Hiphops"`
		Initialise *InitCmd `arg:"subcommand:init" help:"initialise a new Hiphops project"`
		Link       *LinkCmd `arg:"subcommand:link" help:"link to a hiphops.io account"`
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
	case cmd.Down != nil:
		return cmd.Down.Run()
	case cmd.Initialise != nil:
		return cmd.Initialise.Run()
	case cmd.Link != nil:
		return cmd.Link.Run()
	case cmd.Up != nil:
		return cmd.Up.Run()
	default:
		p.WriteHelp(os.Stdout)
		return nil
	}
}
