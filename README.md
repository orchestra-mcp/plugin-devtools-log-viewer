# Orchestra Plugin: devtools-log-viewer

Log file viewer, search, and background process management tools for the [Orchestra MCP](https://github.com/orchestra-mcp/framework) framework.

## Install

```bash
orchestra plugin install github.com/orchestra-mcp/plugin-devtools-log-viewer
```

## Tools

### Log File Tools

| Tool | Description |
|------|-------------|
| `log_tail` | Read the last N lines of a log file |
| `log_watch` | Return a snapshot of the last N lines of a log file |
| `log_search` | Search a log file for lines matching a regex, with context lines |
| `log_parse` | Parse a log file using JSON, syslog, or auto-detection |
| `log_list_sources` | List `.log` files in a directory or common log locations |

### Process Management Tools

| Tool | Description |
|------|-------------|
| `log_run` | Run a shell command in the background and capture its output |
| `log_run_status` | Check status of a background process (running/finished/failed) |
| `log_run_output` | Get captured output, optionally filtered by regex |
| `log_run_kill` | Kill a running process (SIGTERM, then SIGKILL after 3s) |
| `log_run_restart` | Kill and re-run a process with the same command |
| `log_run_list` | List all tracked background processes |

## Usage Examples

### View log files

```
log_tail       path="/var/log/system.log" lines=50
log_search     path="/var/log/system.log" pattern="ERROR" context_lines=3
log_parse      path="/tmp/app.log" format="json"
log_list_sources directory="/var/log"
```

### Run and manage background processes

```
# Start a dev server
log_run        command="make dev" working_directory="/path/to/project"
# Returns: proc-a1b2c3d4

# Check status and recent output
log_run_status id="proc-a1b2c3d4" tail=20

# Search output for errors
log_run_output id="proc-a1b2c3d4" pattern="ERROR|WARN"

# Restart after code changes
log_run_restart id="proc-a1b2c3d4"

# Stop the server
log_run_kill   id="proc-a1b2c3d4"

# See all running processes
log_run_list
```

## Architecture

- **Log file tools** are stateless — each call reads directly from the filesystem
- **Process management tools** are stateful — the plugin maintains an in-memory map of `ManagedProcess` structs, each with a thread-safe ring buffer (1000 lines) capturing combined stdout+stderr
- Commands run via `sh -c` with `Setpgid: true` so kill signals reach the entire process group
- All managed processes are killed on plugin shutdown

## Related Packages

- [sdk-go](https://github.com/orchestra-mcp/sdk-go) — Plugin SDK
- [gen-go](https://github.com/orchestra-mcp/gen-go) — Generated Protobuf types
