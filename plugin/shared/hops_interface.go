// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package shared

import (
	"net/rpc"

	"github.com/hashicorp/go-plugin"
)

// Hops is the interface that we're exposing as a plugin.
type Hops interface {
	Start() string
}

// Here is an implementation that talks over RPC
type HopsRPC struct{ client *rpc.Client }

func (g *HopsRPC) Start() string {
	var resp string
	err := g.client.Call("Plugin.Start", new(interface{}), &resp)
	if err != nil {
		// You usually want your interfaces to return errors. If they don't,
		// there isn't much other choice here.
		panic(err)
	}

	return resp
}

// Here is the RPC server that HopsRPC talks to, conforming to
// the requirements of net/rpc
type HopsRPCServer struct {
	// This is the real implementation
	Impl Hops
}

func (s *HopsRPCServer) Start(args interface{}, resp *string) error {
	*resp = s.Impl.Start()
	return nil
}

// This is the implementation of plugin.Plugin so we can serve/consume this
//
// This has two methods: Server must return an RPC server for this plugin
// type. We construct a HopsRPCServer for this.
//
// Client must return an implementation of our interface that communicates
// over an RPC client. We return HopsRPC for this.
//
// Ignore MuxBroker. That is used to create more multiplexed streams on our
// plugin connection and is a more advanced use case.
type HopsPlugin struct {
	// Impl Injection
	Impl Hops
}

func (p *HopsPlugin) Server(*plugin.MuxBroker) (interface{}, error) {
	return &HopsRPCServer{Impl: p.Impl}, nil
}

func (HopsPlugin) Client(b *plugin.MuxBroker, c *rpc.Client) (interface{}, error) {
	return &HopsRPC{client: c}, nil
}
