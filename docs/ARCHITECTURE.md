# Architecture

Process Guard is an out-of-process Herdr plugin written in Go. It installs no
daemon.

```text
Herdr snapshot + pane process info
                │
                ▼
macOS ps / lsof / getsid ──► process graph
                │                  │
                │                  ▼
                └──────────► attribution + lineage
                                   │
recorded observations ─────────────┤
                                   ▼
                         lifecycle classification
                                   │
                     ┌─────────────┴─────────────┐
                     ▼                           ▼
               text / JSON report          Bubble Tea UI
                                                 │
                                                 ▼
                                       explicit stop preview
```

## Components

- `internal/herdr`: typed client for Herdr's Unix-socket API
- `internal/platform`: process, working-directory, and socket collection
- `internal/guard`: graph construction, attribution, classification, policy,
  and stop-plan validation
- `internal/store`: locked local observation and approval state
- `internal/presentation`: shared human explanations
- `internal/ui`: interactive dashboard and typed stop confirmation

## State

The state file contains stable process identities, sanitized command summaries,
command hashes, ports, timestamps, agent intervals, intentional approvals, and
recent termination survivors. It does not contain file contents and is never
uploaded.

Exact live identity is always authoritative. Historical records can strengthen
attribution or keep a known workload visible during a telemetry miss, but they
cannot make an exited or PID-reused process a valid stop target.
