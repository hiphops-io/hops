# Events, State, and Messages

We use NATS (in particular JetStream) throughout hops. We use NATS to:

- Receive source a.k.a. 'raw' events that trigger workflow runs
- Receive task response events that resume workflow runs
- De-duplicate task dispatches for idempotency
- Store the state of workflow runs


## Setup

To use a seperate NATS server in local dev, you can run: `docker run -p 4222:4222 nats -js`

You'll want to download the `nats` cli tool, on mac this can be installed with:
```bash
brew tap nats-io/nats-tools
brew install nats-io/nats-tools/nats
```

Run hops: `go run ~/Code/hops serve -d --host=https://foo.com -f ~/.hops/main.hops` (your paths and file names may vary)

