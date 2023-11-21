# Hiphops automation engine (Hops)

![Hiphops Casey Logo](assets/casey-full.png)

Hiphops is an automation engine for tech teams.

Product info can be found on our site [hiphops.io](https://www.hiphops.io)
Full docs can be found at [docs.hiphops.io](https://docs.hiphops.io)

## Hosting

Hops is fully open source and can be used without hiphops.io, though this does require hosting and configuring your own NATS server and third party apps.

The automation engine `hops` (this repo) is always self-hosted, and designed to run pretty much anywhere. Right now our docs focus on deploying in a Kubernetes environment, but that is by no means required.

All it really needs is a runtime, with access to a NATS server (ours or yours)

### Fully self hosting

We'll improve the docs for fully self hosted in future, but in the meantime we have an example (local) NATS server with config in `nats/server.go` config is in `testdata/hub-nats.conf`.

This isn't our exact config on hiphops.io (since we have extra bits around multi-tenancy etc), but it's close enough in behaviour that we use it for tests. It will also be the basis for a fully local running mode coming soon.

See the section on [Creating custom apps](#creating-custom-apps) below to add app integrations.


### Hosting hops + hiphops.io account

hiphops.io provides a hosted NATs hub for connectivity in addition to integrations with third party services, and a suite of first party apps to solve common challenges faced by teams that automate their workflow.

For most users the easiest way to get started will be using a hiphops.io account.

---


## Creating custom apps

An app in Hiphops is just code that handles `calls` (a worker) and/or receives events from a data source and adds them to NATS (a listener). In hiphops.io, they may also optionally handle an OAuth flow to provide credential free auth in pipelines.<br>
As long as events follow the subject and body schema laid out in `nats/msgschema.go`, then your app should work.

There is an example 'custom' worker in place already, along with a `worker` package to make this easier to create in go. The example worker is `internal/k8sapp`.<br>
The Kubernetes worker is more complicated than the typical worker, as it also has logic to watch Kubernetes resources and respond asynchronously to `calls`.

Right now there's no `listener` package, though it is planned. `listener` logic is much simpler, so you may find you don't need it (effectively all a listener needs to do is gather/receive events from some third party and pop them onto the correct subject as source events with added `hops` metadata)

> Note: You don't need to create a custom app for the vast majority of use cases. Custom code can be executed via containers within pipelines. Apps are most useful when they handle comms with an API you don't control but must use frequently.

---

Our guide for contributing and developing to hops itself is [here](./docs/contributing.md)
