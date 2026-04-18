# Contributing to Cidrator

Thanks for taking the time to contribute.

Cidrator is deliberately narrow in scope. Good contributions usually improve correctness, portability, tests, documentation, or operator usability without expanding the public surface unnecessarily.

## Before you start

- For substantial changes, open an issue or discussion first so scope and direction are clear before implementation.
- For behavior changes, assume the CLI output may already be used in automation. Treat output changes as compatibility-sensitive.
- For MTU work, be explicit about platform assumptions and failure modes. Networking code that is merely plausible is not enough.

## Development environment

Recommended local requirements:

- Go toolchain `1.24.5`
- `make`
- `git`

Optional but useful:

- `golangci-lint`
- `pre-commit`

Linux is required for the namespace-based MTU lab tests.

## Setup

```bash
git clone https://github.com/YOUR_USERNAME/cidrator.git
cd cidrator
make setup
```

`make setup` builds the project, downloads dependencies, runs a quick test pass, and offers to install optional development tools.

If you prefer a manual setup:

```bash
go mod download
go mod tidy
make build
make test-quick
```

## Development workflow

1. Create a branch for your change.
2. Make the smallest change that solves the problem cleanly.
3. Run fast checks during iteration.
4. Run the full checks before opening a pull request.
5. Update tests and documentation when behavior changes.

Typical loop:

```bash
make dev
make run ARGS="cidr explain 192.168.1.0/24"
make check
```

## Standards

### Scope

- Keep the CLI surface tight.
- Do not add placeholder commands or speculative features.
- Prefer depth and correctness over breadth.

### Code

- Follow normal Go formatting and naming conventions.
- Keep command handlers thin where possible.
- Prefer explicit errors over implicit fallback behavior.
- Avoid dead abstractions and scaffolding that does not serve a real path.

### Output

- `stdout` is for command results.
- `stderr` is for diagnostics and warnings.
- JSON output must remain machine-readable. Do not mix status text into `--json` output paths.

### Documentation

- Match the actual behavior of the repository.
- Prefer short, factual explanations over marketing language.
- Document operational caveats, especially for networking and platform-specific behavior.

## Testing

Run the appropriate level of testing for the change.

### Fast local checks

```bash
make test-quick
make dev
```

### Full local checks

```bash
make check
make test-integration
```

### MTU-specific Linux labs

These are required when you change MTU discovery logic, peer behavior, or related JSON/output contracts.

```bash
make test-lab
make test-lab-hops
make test-lab-plpmtud
```

Requirements for the MTU lab targets:

- Linux
- `iproute2`
- `ping`
- `iptables` for the PLPMTUD black-hole lab
- passwordless `sudo`

## Pull requests

A good pull request should:

- explain the problem and the chosen approach
- call out user-visible behavior changes
- mention platform-specific limitations or assumptions
- include or update tests
- include documentation updates when needed

If the change affects MTU behavior, note which of these were run:

- `go test ./cmd/mtu`
- `go test ./...`
- namespace lab targets on Linux

## Commit messages

Conventional commits are preferred:

```bash
feat: add hop-by-hop MTU lab coverage
fix: preserve JSON-only output for mtu watch
docs: rewrite development guide
```

Common prefixes:

- `feat`
- `fix`
- `docs`
- `test`
- `refactor`
- `chore`

## Getting help

- Issues: <https://github.com/euan-cowie/cidrator/issues>
- Discussions: <https://github.com/euan-cowie/cidrator/discussions>

If you are unsure whether a change fits the project, ask before investing in a large branch.
