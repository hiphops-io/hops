# Contributing

Instructions for contributing to the hops project. This is a work in progress.

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
