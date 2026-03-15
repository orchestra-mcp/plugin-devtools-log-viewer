package tools

import "context"

// ProcessManager is the interface that stateful tools use to interact with
// the plugin's process map. Avoids circular imports between internal and tools.
type ProcessManager interface {
	StartProcess(ctx context.Context, command, workDir string) ProcessInfo
	GetProcess(id string) ProcessInfo
	ListProcesses() []ProcessInfo
	KillProcess(id string) error
	RestartProcess(ctx context.Context, id string) (ProcessInfo, error)
}

// ProcessInfo provides a read-only view of a managed process.
type ProcessInfo interface {
	GetID() string
	GetCommand() string
	GetWorkDir() string
	GetStatus() string // "running", "finished", "failed"
	GetPID() int
	GetExitCode() int
	GetStartedAt() string // ISO 8601
	GetUptimeSeconds() float64
	GetError() string
	GetLastNLines(n int) []string
	GetAllLines() []string
	GetTotalLines() int
	IsRunning() bool
}

// Runner holds the injected ProcessManager that stateful tool handlers need.
type Runner struct {
	PM ProcessManager
}
