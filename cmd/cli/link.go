package main

import (
	"fmt"
)

type LinkCmd struct {
	Dir string `arg:"positional" default:"." help:"path to Hiphops dir - defaults to current directory"`
}

func (l *LinkCmd) Run() error {
	fmt.Println("Linking to hiphops.io")
	return nil
}
