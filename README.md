# Hiphops DevOps Control Plane (Hops)

![Hiphops Casey Logo](assets/casey-full.png)

Hiphops is a DevOps Control Plane and automation engine

Product info can be found on our site [hiphops.io](https://www.hiphops.io)
Full docs can be found at [docs.hiphops.io](https://docs.hiphops.io)


---

Our guide for contributing to hops itself is [here](./docs/contributing.md)

## Local development

### Build

```
go build -C cmd/hops -o bin/hops
```

### Run

Make a directory for your hops installation

```
mkdir my-hiphops-app
cd my-hiphops-app
```

#### Setup hops

```
HIPHOPS_WEB_URL="http://localhost:3000" /path/to/hops-repo/cmd/hops/bin/hops init
HIPHOPS_WEB_URL="http://localhost:3000" /path/to/hops-repo/cmd/hops/bin/hops link
```

#### Run hops

```
HIPHOPS_WEB_URL="http://localhost:3000" /path/to/hops-repo/cmd/hops/bin/hops up
```
