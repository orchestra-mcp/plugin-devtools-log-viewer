package internal

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/orchestra-mcp/plugin-devtools-log-viewer/internal/tools"
	"github.com/orchestra-mcp/sdk-go/plugin"
)

// ToolsPlugin registers all log viewer tools and manages background processes.
type ToolsPlugin struct {
	processes map[string]*ManagedProcess
	mu        sync.RWMutex
}

// NewToolsPlugin creates a ToolsPlugin with an initialized process map.
func NewToolsPlugin() *ToolsPlugin {
	return &ToolsPlugin{
		processes: make(map[string]*ManagedProcess),
	}
}

// RegisterTools registers all 11 log viewer tools with the plugin builder.
func (tp *ToolsPlugin) RegisterTools(builder *plugin.PluginBuilder) {
	// --- 5 existing stateless tools ---
	builder.RegisterTool("log_tail",
		"Read the last N lines of a log file",
		tools.LogTailSchema(), tools.LogTail())

	builder.RegisterTool("log_watch",
		"Return a snapshot of the last N lines of a log file (streaming not supported)",
		tools.LogWatchSchema(), tools.LogWatch())

	builder.RegisterTool("log_search",
		"Search a log file for lines matching a regular expression, with optional context lines",
		tools.LogSearchSchema(), tools.LogSearch())

	builder.RegisterTool("log_parse",
		"Parse a log file using json, syslog, or auto-detection format",
		tools.LogParseSchema(), tools.LogParse())

	builder.RegisterTool("log_list_sources",
		"List log files in a directory or common log locations",
		tools.LogListSourcesSchema(), tools.LogListSources())

	// --- 6 new stateful tools (process management) ---
	runner := &tools.Runner{PM: tp}

	builder.RegisterTool("log_run",
		"Run a shell command in the background and capture its output for live viewing",
		tools.LogRunSchema(), tools.LogRun(runner))

	builder.RegisterTool("log_run_status",
		"Check the status of a background process (running/finished/failed, PID, uptime, last N lines)",
		tools.LogRunStatusSchema(), tools.LogRunStatus(runner))

	builder.RegisterTool("log_run_output",
		"Get captured output from a background process (last N lines or filtered by regex)",
		tools.LogRunOutputSchema(), tools.LogRunOutput(runner))

	builder.RegisterTool("log_run_kill",
		"Kill a running background process by ID (SIGTERM then SIGKILL after 3s)",
		tools.LogRunKillSchema(), tools.LogRunKill(runner))

	builder.RegisterTool("log_run_restart",
		"Kill and re-run a background process with the same command and working directory",
		tools.LogRunRestartSchema(), tools.LogRunRestart(runner))

	builder.RegisterTool("log_run_list",
		"List all tracked background processes with their status",
		tools.LogRunListSchema(), tools.LogRunList(runner))
}

// KillAll kills all running processes. Called on plugin shutdown.
func (tp *ToolsPlugin) KillAll() {
	tp.mu.Lock()
	defer tp.mu.Unlock()
	for _, proc := range tp.processes {
		_ = proc.Kill()
	}
}

// --- ProcessManager interface implementation ---

func (tp *ToolsPlugin) StartProcess(ctx context.Context, command, workDir string) tools.ProcessInfo {
	id := GenerateProcessID()
	proc := StartProcess(ctx, id, command, workDir, 1000)
	tp.mu.Lock()
	tp.processes[id] = proc
	tp.mu.Unlock()
	return &processInfoAdapter{proc: proc}
}

func (tp *ToolsPlugin) GetProcess(id string) tools.ProcessInfo {
	tp.mu.RLock()
	defer tp.mu.RUnlock()
	proc, ok := tp.processes[id]
	if !ok {
		return nil
	}
	return &processInfoAdapter{proc: proc}
}

func (tp *ToolsPlugin) ListProcesses() []tools.ProcessInfo {
	tp.mu.RLock()
	defer tp.mu.RUnlock()
	result := make([]tools.ProcessInfo, 0, len(tp.processes))
	for _, proc := range tp.processes {
		result = append(result, &processInfoAdapter{proc: proc})
	}
	return result
}

func (tp *ToolsPlugin) KillProcess(id string) error {
	tp.mu.RLock()
	proc, ok := tp.processes[id]
	tp.mu.RUnlock()
	if !ok {
		return fmt.Errorf("no process found with ID %q", id)
	}
	return proc.Kill()
}

func (tp *ToolsPlugin) RestartProcess(ctx context.Context, id string) (tools.ProcessInfo, error) {
	tp.mu.Lock()
	oldProc, ok := tp.processes[id]
	if !ok {
		tp.mu.Unlock()
		return nil, fmt.Errorf("no process found with ID %q", id)
	}
	command := oldProc.Command
	workDir := oldProc.WorkDir
	tp.mu.Unlock()

	_ = oldProc.Kill()

	newProc := StartProcess(ctx, id, command, workDir, 1000)

	tp.mu.Lock()
	tp.processes[id] = newProc
	tp.mu.Unlock()

	return &processInfoAdapter{proc: newProc}, nil
}

// processInfoAdapter bridges internal.ManagedProcess to tools.ProcessInfo.
type processInfoAdapter struct {
	proc *ManagedProcess
}

func (a *processInfoAdapter) GetID() string             { return a.proc.ID }
func (a *processInfoAdapter) GetCommand() string         { return a.proc.Command }
func (a *processInfoAdapter) GetWorkDir() string         { return a.proc.WorkDir }
func (a *processInfoAdapter) GetStatus() string          { return string(a.proc.Status()) }
func (a *processInfoAdapter) GetPID() int                { return a.proc.PID }
func (a *processInfoAdapter) GetExitCode() int           { return a.proc.ExitCode }
func (a *processInfoAdapter) GetStartedAt() string       { return a.proc.StartedAt.Format(time.RFC3339) }
func (a *processInfoAdapter) GetUptimeSeconds() float64  { return a.proc.UptimeSeconds() }
func (a *processInfoAdapter) IsRunning() bool            { return a.proc.IsRunning() }
func (a *processInfoAdapter) GetLastNLines(n int) []string { return a.proc.Output.LastN(n) }
func (a *processInfoAdapter) GetAllLines() []string      { return a.proc.Output.All() }
func (a *processInfoAdapter) GetTotalLines() int         { return a.proc.Output.TotalLines() }
func (a *processInfoAdapter) GetError() string           { return a.proc.ErrorString() }
