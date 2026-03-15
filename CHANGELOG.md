# Changelog

## [0.2.0] - 2026-03-15

### Added
- **Background process management** — 6 new stateful tools for running and managing shell commands
  - `log_run` — Run a shell command in the background and capture its output
  - `log_run_status` — Check status of a background process (running/finished/failed)
  - `log_run_output` — Get captured output, optionally filtered by regex
  - `log_run_kill` — Kill a running process (SIGTERM, then SIGKILL after 3s)
  - `log_run_restart` — Kill and re-run a process with the same command and ID
  - `log_run_list` — List all tracked background processes
- Thread-safe ring buffer (1000 lines) for capturing combined stdout+stderr
- Process group isolation (`Setpgid: true`) for clean process tree kills
- `ProcessManager` interface for dependency injection and testability
- Integration tests for all 6 new tools (14 test cases)
- Ring buffer unit tests (7 test cases)

### Changed
- `ToolsPlugin` is now stateful with an in-memory process map and mutex
- Added `NewToolsPlugin()` constructor
- `Register()` in `export.go` now returns a cleanup function for shutdown
- `cmd/main.go` calls `KillAll()` on SIGTERM/SIGINT for graceful shutdown
- Refactored existing 5 log file tools into individual files under `internal/tools/`

## [0.1.0] - Initial Release

- Scaffolded from `scripts/new-plugin.sh`
