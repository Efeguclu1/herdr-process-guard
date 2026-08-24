# Contributing

Thanks for helping make agent process lifecycle behavior safer and easier to
understand.

## Development setup

Requirements: macOS, Go 1.24+, and Herdr 0.8.0+.

```sh
git clone https://github.com/Efeguclu1/herdr-process-guard.git
cd herdr-process-guard
make check
make build
herdr plugin link "$PWD"
```

Open the dashboard with:

```sh
herdr plugin pane open --plugin herdr.process-guard --entrypoint dashboard
```

## Pull requests

- Keep discovery and classification separate from termination.
- Add deterministic fixtures for new process-tree or agent signatures.
- Never weaken PID/start-time identity checks or typed confirmations.
- Do not include raw commands, tokens, usernames, or private paths in fixtures.
- Run `make check` before opening a pull request.

Changes to stop behavior should include both a dry-run test and a stale-preview
test. New classification evidence should be explainable in plain language and
represented in JSON.

## Adding agent or workload recognition

Open an issue first when a detector could match generic processes such as
`node`, `python`, or `java`. Include a sanitized ancestry tree, process start
times, listener state, and the Herdr agent lifecycle transition. Generic name
matching alone is not sufficient attribution evidence.
