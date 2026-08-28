<p align="center">
  <img src="assets/lantern-tui.png" alt="Lantern TUI logo" width="180">
</p>

<h1 align="center">Lantern TUI</h1>

<p align="center">
  A terminal interface for exploring Lantern audits, runs, queues, and configuration.
</p>

Lantern TUI is the interactive terminal companion to [Lantern](https://github.com/TheHarborProject/lantern). It provides keyboard- and mouse-driven workspaces for reviewing components, states, checks, evidence, previous runs, queued work, and project configuration.

## Requirements

- Go 1.25.4 or newer
- A terminal with true-color support

## Run locally

```bash
go run .
```

The application reads Lantern configuration from the current working directory. A sample `lantern.config.json` is included in this repository.

## Development

Run the test suite:

```bash
go test ./...
```

Build the executable:

```bash
go build -o lantern-tui .
```

## Current workspaces

- **Audit** — browse components, states, checks, and supporting evidence.
- **Runs** — inspect stored survey runs and their details.
- **Queue** — review and control queued audit work.
- **Config** — inspect and edit authored Lantern configuration.

The application is under active development. The current audit and run data shown at startup is fixture-backed while integration with Lantern's persisted survey output evolves.
