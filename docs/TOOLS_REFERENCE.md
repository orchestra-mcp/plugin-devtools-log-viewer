# Tools Reference

## Log File Tools

### log_tail

Read the last N lines of a log file.

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `path` | string | Yes | — | Absolute path to the log file |
| `lines` | number | No | 50 | Number of lines to return from the end |

### log_watch

Return a snapshot of the last N lines of a log file.

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `path` | string | Yes | — | Absolute path to the log file |
| `lines` | number | No | 20 | Number of recent lines to return |

### log_search

Search a log file for lines matching a regular expression, with optional context lines.

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `path` | string | Yes | — | Absolute path to the log file |
| `pattern` | string | Yes | — | Regular expression pattern to search for |
| `context_lines` | number | No | 2 | Number of context lines before and after each match |

### log_parse

Parse a log file using JSON, syslog, or auto-detection format.

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `path` | string | Yes | — | Absolute path to the log file |
| `format` | string | No | `auto` | Log format: `json`, `syslog`, or `auto` |

### log_list_sources

List `.log` files in a directory or common log locations (`/var/log`, `/tmp`, `~/Library/Logs`).

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `directory` | string | No | — | Directory to search. If omitted, checks common log directories. |

---

## Process Management Tools

### log_run

Run a shell command in the background and capture its output for live viewing. Returns a process ID immediately.

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `command` | string | Yes | — | Shell command to run (e.g., `make dev`, `npm start`) |
| `working_directory` | string | No | `ORCHESTRA_WORKSPACE` or cwd | Working directory for the command |

**Response includes:** process ID, command, working directory, PID, status.

Commands run via `sh -c` with process group isolation (`Setpgid: true`). Combined stdout+stderr is captured into a ring buffer (last 1000 lines).

### log_run_status

Check the status of a background process.

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `id` | string | Yes | — | Process ID returned by `log_run` |
| `tail` | number | No | 20 | Number of recent output lines to include |

**Response includes:** command, working directory, status (`running`/`finished`/`failed`), PID, start time, uptime, exit code (if finished), error message (if failed), total output lines, and last N lines.

### log_run_output

Get captured output from a background process.

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `id` | string | Yes | — | Process ID returned by `log_run` |
| `lines` | number | No | 100 | Number of recent lines to return. `0` returns all buffered lines. |
| `pattern` | string | No | — | Regex pattern to filter output lines |

### log_run_kill

Kill a running background process by ID. Sends SIGTERM to the process group, waits 3 seconds, then sends SIGKILL if still alive.

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `id` | string | Yes | — | Process ID to kill |

### log_run_restart

Kill and re-run a background process with the same command and working directory. The process keeps the same ID but gets a new PID.

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `id` | string | Yes | — | Process ID to restart |

### log_run_list

List all tracked background processes with their status. Takes no arguments.

**Response:** Markdown table with columns: ID, Command, Status, PID, Uptime, Exit Code.
