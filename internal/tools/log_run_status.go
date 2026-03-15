package tools

import (
	"context"
	"fmt"
	"strings"

	pluginv1 "github.com/orchestra-mcp/gen-go/orchestra/plugin/v1"
	"github.com/orchestra-mcp/sdk-go/helpers"
	"google.golang.org/protobuf/types/known/structpb"
)

// LogRunStatusSchema returns the JSON Schema for the log_run_status tool.
func LogRunStatusSchema() *structpb.Struct {
	s, _ := structpb.NewStruct(map[string]any{
		"type": "object",
		"properties": map[string]any{
			"id": map[string]any{
				"type":        "string",
				"description": "Process ID returned by log_run",
			},
			"tail": map[string]any{
				"type":        "number",
				"description": "Number of recent output lines to include (default 20)",
			},
		},
		"required": []any{"id"},
	})
	return s
}

// LogRunStatus returns a tool handler that checks the status of a background process.
func LogRunStatus(runner *Runner) func(context.Context, *pluginv1.ToolRequest) (*pluginv1.ToolResponse, error) {
	return func(ctx context.Context, req *pluginv1.ToolRequest) (*pluginv1.ToolResponse, error) {
		if err := helpers.ValidateRequired(req.Arguments, "id"); err != nil {
			return helpers.ErrorResult("validation_error", err.Error()), nil
		}

		id := helpers.GetString(req.Arguments, "id")
		tail := helpers.GetInt(req.Arguments, "tail")
		if tail <= 0 {
			tail = 20
		}

		proc := runner.PM.GetProcess(id)
		if proc == nil {
			return helpers.ErrorResult("not_found",
				fmt.Sprintf("no process found with ID %q", id)), nil
		}

		var b strings.Builder
		fmt.Fprintf(&b, "## Process: %s\n\n", id)
		fmt.Fprintf(&b, "- **Command:** `%s`\n", proc.GetCommand())
		fmt.Fprintf(&b, "- **Working Dir:** %s\n", proc.GetWorkDir())
		fmt.Fprintf(&b, "- **Status:** %s\n", proc.GetStatus())
		fmt.Fprintf(&b, "- **PID:** %d\n", proc.GetPID())
		fmt.Fprintf(&b, "- **Started:** %s\n", proc.GetStartedAt())
		fmt.Fprintf(&b, "- **Uptime:** %.1fs\n", proc.GetUptimeSeconds())

		if !proc.IsRunning() {
			fmt.Fprintf(&b, "- **Exit Code:** %d\n", proc.GetExitCode())
			if errMsg := proc.GetError(); errMsg != "" {
				fmt.Fprintf(&b, "- **Error:** %s\n", errMsg)
			}
		}

		fmt.Fprintf(&b, "- **Total Output Lines:** %d\n", proc.GetTotalLines())

		lines := proc.GetLastNLines(tail)
		if len(lines) > 0 {
			fmt.Fprintf(&b, "\n### Last %d Lines\n\n```\n", len(lines))
			for _, line := range lines {
				fmt.Fprintf(&b, "%s\n", line)
			}
			b.WriteString("```\n")
		}

		return helpers.TextResult(b.String()), nil
	}
}
