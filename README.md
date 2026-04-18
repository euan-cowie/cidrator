# Cidrator

Cidrator is a Go CLI for three network tasks:

- CIDR inspection and manipulation
- DNS lookups and reverse lookups
- Path MTU discovery and MTU-related sizing

The project is designed for interactive troubleshooting and shell-friendly automation. It favors a small, credible surface area over broad feature count.

## Scope

`cidrator` currently ships three command groups:

- `cidr`: explain, expand, contains, count, overlaps, and divide IPv4 or IPv6 CIDR ranges
- `dns`: query common DNS record types and perform PTR lookups
- `mtu`: discover Path MTU, monitor changes, inspect local interfaces, calculate payload suggestions, and run an advanced peer-assisted endpoint

Commands exposed in the CLI are expected to be implemented, tested, and documented. Experimental or incomplete features are intentionally kept out of the public surface.

## Installation

### Go

```bash
go install github.com/euan-cowie/cidrator@latest
```

### Release binaries

Prebuilt binaries are published on the [Releases](https://github.com/euan-cowie/cidrator/releases) page.

### From source

```bash
git clone https://github.com/euan-cowie/cidrator.git
cd cidrator
make build
./bin/cidrator --help
```

## Quick start

```bash
# Show command groups
cidrator --help

# Explain a network
cidrator cidr explain 192.168.1.0/24

# Check membership
cidrator cidr contains 192.168.1.0/24 192.168.1.42

# Query DNS
cidrator dns lookup example.com --type MX

# Reverse DNS
cidrator dns reverse 8.8.8.8

# Discover Path MTU
cidrator mtu discover example.com

# Machine-readable MTU output
cidrator mtu discover example.com --proto tcp --json
```

## Command overview

### `cidr`

The `cidr` command group focuses on deterministic network calculations for IPv4 and IPv6.

Common commands:

```bash
cidrator cidr explain 10.0.0.0/16 --format json
cidrator cidr count 2001:db8::/48
cidrator cidr overlaps 10.0.0.0/16 10.0.1.0/24
cidrator cidr divide 192.168.0.0/24 4
cidrator cidr expand 192.168.1.0/30
```

### `dns`

The `dns` command group supports forward lookups for common record types and reverse lookups for IP addresses.

Common commands:

```bash
cidrator dns lookup example.com --type A
cidrator dns lookup example.com --type ALL --format yaml
cidrator dns lookup example.com --server 1.1.1.1
cidrator dns reverse 2001:4860:4860::8888
```

### `mtu`

The `mtu` command group covers Path MTU discovery, monitoring, interface inspection, and size recommendations derived from the discovered path.

Common commands:

```bash
cidrator mtu discover example.com
cidrator mtu discover example.com --proto udp --port 4821
cidrator mtu watch example.com --interval 30s
cidrator mtu interfaces --json
cidrator mtu suggest example.com --json
```

Supported MTU probe modes:

- `icmp`: default Path MTU discovery
- `tcp`: peer-assisted or service-assisted probing over TCP
- `udp`: peer-assisted or service-assisted probing over UDP

Advanced MTU topics are documented separately in [cmd/mtu/mtu_guide.md](cmd/mtu/mtu_guide.md).

### Advanced peer-assisted MTU mode

`cidrator mtu peer` is intended for controlled environments where you manage both ends of the path. It is useful when you want to verify application-to-application MTU behavior instead of relying on whatever service happens to be reachable on the remote host.

Typical workflow:

```bash
# On the remote host
cidrator mtu peer --proto udp --listen 0.0.0.0 --allow-remote --port 4821

# From another host
cidrator mtu discover remote-host.example.com --proto udp --port 4821
```

The peer endpoint binds to localhost by default and requires `--allow-remote` for non-loopback addresses.

## Output formats

The CLI supports structured output where it is useful for automation:

- `cidr` and `dns` commands support `table`, `json`, and `yaml` output where applicable
- `mtu` commands support `--json`

The project treats structured output as part of the command contract. Changes to JSON shape or mixed stdout/stderr behavior should be made carefully and tested explicitly.

## Development

The repository uses Go toolchain `1.24.5` via the `toolchain` directive in `go.mod`.

Common commands:

```bash
make build
make test
make check
make run ARGS="cidr explain 192.168.1.0/24"
```

Additional development documentation:

- [CONTRIBUTING.md](CONTRIBUTING.md)
- [DEVELOPMENT.md](DEVELOPMENT.md)

## Testing

The repository includes unit tests, integration tests, and Linux namespace-based MTU labs.

Examples:

```bash
make test
make test-integration
make test-lab
make test-lab-hops
make test-lab-plpmtud
```

The MTU lab targets require Linux and passwordless `sudo`.

## License

This project is licensed under the terms of the [MIT License](LICENSE).
