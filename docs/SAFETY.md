# Safety model

Process Guard treats classification and termination as separate systems. A
classification can suggest review; it never authorizes a signal by itself.

## Discovery invariants

- Only processes owned by the current user are inspected.
- Herdr pane shells, coding agents, and Process Guard are protected
  infrastructure.
- PID reuse is handled with a stable identity composed of PID and process start
  time.
- Direct live ancestry is stronger evidence than pane-session proximity or a
  historical agent activity window.
- Missing socket telemetry cannot silently erase a previously confirmed live
  workload identity.

## Leftover eligibility

By default, `LEFTOVER: LIKELY` requires all of the following:

- policy is not `intentional`
- process is idle
- agent is `done`, `idle`, or no longer reported
- at least two observations exist
- the first observation is at least 30 minutes old
- attribution confidence is medium or high
- no established network connection is visible

A listening socket alone does not mean a service is in use. An established
connection does.

## Termination protocol

1. Rescan immediately before building a stop plan.
2. Capture every target's PID, start time, session ID, executable, and command
   hash.
3. Reject the plan if any identity or membership changes.
4. Display the full target tree and listening ports.
5. Require the user to type the exact confirmation phrase.
6. Send `SIGTERM` for a graceful stop.
7. Record exact survivors; do not escalate automatically.
8. Permit `SIGKILL` only through a separate force preview and confirmation for
   those recent exact survivors.

There is no rollback for process termination. This is why Process Guard favors
false negatives and additional observation over aggressive cleanup.
