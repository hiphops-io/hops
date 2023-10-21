# Undistribute

This package handles sharing events and state in a way that allows a single instance of a distributed application to act as if isn't distributed - such as accessing long-lived state via local disk, using local memory cache, etc.

> Important! This package is not yet ready for production use. Recovery/contention logic has yet to be implemented.

## Why?

Whilst a traditional distributed architecture is powerful, it makes some solutions incredibly challenging - especially at low latency.

In particular systems processing multi-step, stateful workflows face difficulties. Traditionally they can either access all state via some shared authoratative source over the network (e.g. a database), or not be distributed at all.

_Most_ applications will be served better by using the shared data store, but when extremely low latency is a concern, the additional network calls to fetch state are problematic.

Additionally, workloads that require large state to be preserved across multiple steps are hit even harder, needing to restore that state on every step. For state over 100's of MBs this can introduce latency beyond even modest requirements.

## How?

If you've used a load balancer for a webserver, conceptually it's similar to sticky sessions (but for backend workflows and state).

We use NATS and the local disk to create a per-instance 'lease'

- Undistribute automatically configures a JetStream stream, subjects, and consumers for leasing work.
- Unleased events are published and distributed to leaseholders on a first come first served basis.
- All subsquent events in a workflow can be routed to that leaseholder's specific subject, ensuring it is the only instance that receives them.
- The leaseholder instance can maintain full state locally, knowing it has exclusive and exhaustive access.

## Important Caveats

Our setup is quite opinionated, based on common requirements for workflows.

Every message must be published to a unique subject, or it will fail.
> This ensures long-lived de-duplication of messages. If you genuinely want to re-publish, then alter the subject (perhaps by appending an `attempts` count).

Streams are configured on a workqueue delivery policy. One and only one consumer will receive any event.


## Coming next

The above setup works well on the happy path, but undistributing isn't particularly clever unless you can recover from failed instances. This is designed, but yet to be implemented.

At a high level we'll support registering/refreshing leases centrally, with a mechanism for expired leases being 'ceded' to a healthy instance along with their entire state.
