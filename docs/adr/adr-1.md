# Workflow state & events design

|Metadata|Value|
|--------|-----|
|Date    |2023-05-17|
|Author  |@manterfield|
|Status  |Proposed|

## Context

Hops instances require low-latency access to the aggregate state (event bundle) for a workflow. The event bundle is processed in full by a hops config each time it is updated.

The design for handling state and events must support:

- The ability to run hops in a way that provides HA
- Hops' capability to run both locally by end users, self-hosted, and hosted by Hiphops.io
- Sub-millisecond latency when processing workflow events
- Stateful, accumulative local disk access across multiple tasks

## Design

Every hops instance will assign themselves a unique UUID 'lease tag' at startup, identifying that instance

When a raw event is broadcast to a cluster, a single hops instance will consume that event and take exclusive, transferrable ownership of any workflows and events ran against it using that lease.

The lease tag will be included with any task requests, responses, or other comms related to workflows processed by an instance. All instances will filter events by their own lease tag. This ensures that one and only one instance will ever process events relating to a single workflow.

All events for a workflow will be persisted to an append only log. Given the exclusive ownership of the workflow events by one instance, this means that instance will have access to the entire event bundle for that workflow on disk (or in memory for future performance gain).

The single event bundle will also be persisted remotely, allowing recovery of stopped or failed hops instance workflows.

Leases for stopped or failed workflows can be transferred to healthy instances

### Leases

- [ ] A hops instance ensures a single primary lease tag exists on disk at startup
- [ ] Lease tags will be random UUIDs
- [ ] All workflow events will be assigned a lease tag for filtering

### Leases phase 2

- [ ] A hops instance will register any leases with the rest of the cluster to ensure unique ownership
- [ ] Additional lease tags may exist, representing leases that were transferred from other instances
- [ ] New events will only ever be assigned to the primary lease tag. This effectively means transferred lease tags will be drained gracefully over time
- [ ] Leases must be renewed every `LEASE_RENEW_DEADLINE`
- [ ] If a lease has not been renewed within the `LEASE_RENEW_DEADLINE` the cluster will transfer it to a healthy instance
- [ ] An instance must not process any events without an active lease
- [ ] Transferred leases must be released (removed locally and removed from cluster) after all pipelines are completed, failed, or timed out (i.e. all pipelines are finalised)

### Events

- [ ] Raw events (a.k.a source events) are published on a shared stream in NATS JetStream with a retention policy of `WorkQueuePolicy`, ensuring exactly once delivery
- [ ] Workflow events (task requests, responses, lifecycle events, etc) are stored in a NATS Key/Value store named after the lease UUID
- [ ] The raw event will also be persisted to this KV store to ensure it is a complete record of all workflow run data


