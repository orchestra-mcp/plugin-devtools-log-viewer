package tools

import (
	"context"
	"fmt"

	pluginv1 "github.com/orchestra-mcp/gen-go/orchestra/plugin/v1"
	"github.com/orchestra-mcp/sdk-go/helpers"
	"google.golang.org/protobuf/types/known/structpb"
)

// LogRunRestartSchema returns the JSON Schema for the log_run_restart tool.
func LogRunRestartSchema() *structpb.Struct {
	s, _ := structpb.NewStruct(map[string]any{
		"type": "object",
		"properties": map[string]any{
			"id": map[string]any{
				"type":        "string",
				"description": "Process ID to restart (re-runs the same command in the same directory)",
			},
		},
		"required": []any{"id"},
	})
	return s
}

// LogRunRestart returns a tool handler that kills and re-runs a background process.
func LogRunRestart(runner *Runner) func(context.Context, *pluginv1.ToolRequest) (*pluginv1.ToolResponse, error) {
	return func(ctx context.Context, req *pluginv1.ToolRequest) (*pluginv1.ToolResponse, error) {
		if err := helpers.ValidateRequired(req.Arguments, "id"); err != nil {
			return helpers.ErrorResult("validation_error", err.Error()), nil
		}

		id := helpers.GetString(req.Arguments, "id")

		newProc, err := runner.PM.RestartProcess(ctx, id)
		if err != nil {
			return helpers.ErrorResult("restart_error", err.Error()), nil
		}

		return helpers.TextResult(fmt.Sprintf(
			"## Process Restarted\n\n"+
				"- **ID:** %s\n"+
				"- **Command:** `%s`\n"+
				"- **New PID:** %d\n"+
				"- **Status:** %s\n",
			newProc.GetID(), newProc.GetCommand(), newProc.GetPID(), newProc.GetStatus(),
		)), nil
	}
}
