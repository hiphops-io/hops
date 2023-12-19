// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0
// Changes by Lukas Oberhuber

package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"

	hclog "github.com/hashicorp/go-hclog"
	"github.com/hashicorp/go-plugin"
	"github.com/hiphops-io/hops/plugin/shared"
)

func main() {
	c := make(chan os.Signal, 1) // we need to reserve to buffer size 1, so the notifier are not blocked
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	// Create an hclog.Logger
	logger := hclog.New(&hclog.LoggerOptions{
		Name:   "plugin",
		Output: os.Stdout,
		Level:  hclog.Debug,
	})

	fmt.Println(os.Getwd())

	// We're a host! Start by launching the plugin process.
	client := plugin.NewClient(&plugin.ClientConfig{
		HandshakeConfig: handshakeConfig,
		Plugins:         pluginMap,
		Cmd:             exec.Command("./hops", "start", "--address=0.0.0.0:8916", "--keyfile=~/.hops/hiphops.key", "--plugin-mode"),
		Logger:          logger,
	})
	defer client.Kill()

	// Connect via RPC
	rpcClient, err := client.Client()
	if err != nil {
		log.Fatal(err)
		// return err
	}

	// Request the plugin
	raw, err := rpcClient.Dispense("hops")
	if err != nil {
		log.Fatal(err)
		// return err
	}

	// We should have a Greeter now! This feels like a normal interface
	// implementation but is in fact over an RPC connection.
	hops := raw.(shared.Hops)

	for {
		select {
		case <-c:
			client.Kill()
			os.Exit(1)
		case <-time.After(1 * time.Second):
			fmt.Println(hops.Start())
		}
	}
}

// handshakeConfigs are used to just do a basic handshake between
// a plugin and host. If the handshake fails, a user friendly error is shown.
// This prevents users from executing bad plugins or executing a plugin
// directory. It is a UX feature, not a security feature.
var handshakeConfig = plugin.HandshakeConfig{
	ProtocolVersion:  1,
	MagicCookieKey:   "HOPS_PLUGIN",
	MagicCookieValue: "runningasexpected",
}

// pluginMap is the map of plugins we can dispense.
var pluginMap = map[string]plugin.Plugin{
	"hops": &shared.HopsPlugin{},
}
