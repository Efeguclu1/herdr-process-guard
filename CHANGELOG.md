# Changelog

All notable changes are documented here. The project follows Semantic
Versioning while pre-1.0 APIs remain explicitly unstable.

## [Unreleased]

## [0.1.3] - 2026-08-24

### Added

- Human-readable Herdr workspace, tab, pane-title, and working-directory
  attribution.
- Explicit relationship, lineage, activity, and leftover explanations.
- Stable workload identity across partial macOS `lsof` telemetry.
- Launch documentation, safety policy, CI, release packaging, and visual assets.

### Safety

- Meaningful-CPU threshold prevents housekeeping activity from keeping an idle
  workload active indefinitely.
- Stop previews continue to require exact PID/start-time and tree membership.

[Unreleased]: https://github.com/Efeguclu1/herdr-process-guard/compare/v0.1.3...HEAD
[0.1.3]: https://github.com/Efeguclu1/herdr-process-guard/releases/tag/v0.1.3
