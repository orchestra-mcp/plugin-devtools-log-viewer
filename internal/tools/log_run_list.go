package tools

import (
	"context"
	"fmt"
	"strings"

	pluginv1 "github.com/orchestra-mcp/gen-go/orchestra/plugin/v1"
	"github.com/orchestra-mcp/sdk-go/helpers"
	"google.golang.org/protobuf/types/known/structpb"
)

// LogRunListSchema returns the JSON Schema for the log_run_list tool.
func LogRunListSchema() *structpb.Struct {
	s, _ := structpb.NewStruct(map[string]any{
		"type":       "object",
		"properties": map[string]any{},
	})
	return s
}

// LogRunList returns a tool handler that lists all tracked background processes.
func LogRunList(runner *Runner) func(context.Context, *pluginv1.ToolRequest) (*pluginv1.ToolResponse, error) {
	return func(ctx context.Context, req *pluginv1.ToolRequest) (*pluginv1.ToolResponse, error) {
		procs := runner.PM.ListProcesses()

		if len(procs) == 0 {
			return helpers.TextResult("## Managed Processes\n\nNo tracked processes.\n"), nil
		}

		var b strings.Builder
		fmt.Fprintf(&b, "## Managed Processes (%d)\n\n", len(procs))
		fmt.Fprintf(&b, "| ID | Command | Status | PID | Uptime | Exit Code |\n")
		fmt.Fprintf(&b, "|----|---------|--------|-----|--------|-----------|\n")

		for _, p := range procs {
			exitCode := "—"
			if !p.IsRunning() {
				exitCode = fmt.Sprintf("%d", p.GetExitCode())
			}
			cmd := p.GetCommand()
			if len(cmd) > 40 {
				cmd = cmd[:37] + "..."
			}
			fmt.Fprintf(&b, "| %s | `%s` | %s | %d | %.1fs | %s |\n",
				p.GetID(), cmd, p.GetStatus(), p.GetPID(), p.GetUptimeSeconds(), exitCode)
		}

		return helpers.TextResult(b.String()), nil
	}
}
