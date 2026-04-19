# Development guide

This document is a working reference for local development on `cidrator`.

## Requirements

- Go `1.24` with toolchain `1.24.5`
- `make`
- `git`

For Linux MTU labs:

- Linux
- `iproute2`
- `ping`
- `iptables` for the PLPMTUD black-hole lab
- passwordless `sudo`

## Initial setup

```bash
make setup
```

`make setup` downloads modules, builds the project, and runs a quick test pass.

If you also want optional local tooling:

```bash
make setup-tools
```

## Common commands

### Build and run

```bash
make build
make run ARGS="--help"
make run ARGS="cidr explain 192.168.1.0/24"
```

### Day-to-day development

```bash
make dev
make test-quick
make check
```

### Full test targets

```bash
make test
make test-integration
```

### MTU lab targets

```bash
make test-lab
make test-lab-hops
make test-lab-plpmtud
```

### Quality tools

```bash
make fmt
make vet
make lint
make lint-if-available
```

### Other useful targets

```bash
make build-all
make clean
make deps
make help
```

## Suggested workflow

For most changes:

```bash
make dev
make run ARGS="cidr explain 10.0.0.0/16"
make check
```

For MTU changes:

```bash
go test ./cmd/mtu
go test ./...
make test-lab           # Linux only
make test-lab-hops      # Linux only
make test-lab-plpmtud   # Linux only
```

If a change affects structured output, verify the JSON path directly.

## Repository layout

```text
cmd/
  cidr/    CLI commands for CIDR analysis
  dns/     CLI commands for DNS lookups
  mtu/     CLI commands and MTU discovery logic

internal/
  cidr/        CIDR implementation
  dns/         DNS implementation

test/labs/
  Linux namespace-based MTU integration labs
```

## MTU-specific notes

The MTU package has three layers of verification:

1. Unit tests for local logic and parsing
2. Package-level tests for probe and command behavior
3. Linux namespace labs for routed-path validation

The Linux labs are part of CI and run on pull requests. They are the main confidence check for behavior that depends on real forwarding, MTU bottlenecks, or ICMP handling.

Relevant lab scripts:

- `test/labs/mtu-namespaces.sh`
- `test/labs/mtu-hop-by-hop.sh`
- `test/labs/mtu-plpmtud-blackhole.sh`

## Troubleshooting

### `golangci-lint` is not in `PATH`

Install it explicitly:

```bash
make install-tools
```

If the binary still is not visible, make sure `$(go env GOPATH)/bin` is on your `PATH`.

Typical fix:

```bash
source ~/.zshrc
```

### Tests fail after dependency or toolchain changes

```bash
go mod download
go mod tidy
make clean
make test
```

### MTU labs fail locally

Check the host requirements first:

- Linux, not macOS
- passwordless `sudo`
- required commands installed

Then rerun the specific target with a fresh build:

```bash
make build
make test-lab
```

### Need the full command list

```bash
make help
```
