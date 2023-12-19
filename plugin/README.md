Adds `--plugin-mode` flag to hops. When set, hops will run in plugin mode. In this mode, hops listens for commands from a server. The server runs hops in a subprocess and sends commands to it via the go-plugin interface.

Currently this just sends and received a `Start` message every second.

From root of hops:

```base
go build -o plugin/server/server plugin/server/main.go
go build -o hops .
plugin/server/server
```

To see hops running, look at output of `ps`.

To stop, press `Ctrl+C` in the terminal. This also kills the running hops.

## Notes / Findings

* Logging has to be directed to `Stderr` as the server will otherwise fail. See <https://github.com/hashicorp/go-plugin/issues/199>
* The server must handle graceful shutdowns, or `hops` stays running.
* go-plugin has a capability to reconnect to a process, but this was not investigated.
* Uses the `net/rpc` version, not the `gRPC` version.
* Shared interface should live in hops, but obviously changes to this interface would have to be dealt with as and when.
* Only useful for local communications.