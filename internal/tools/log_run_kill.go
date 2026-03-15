package tools

import (
	"context"
	"fmt"

	pluginv1 "github.com/orchestra-mcp/gen-go/orchestra/plugin/v1"
	"github.com/orchestra-mcp/sdk-go/helpers"
	"google.golang.org/protobuf/types/known/structpb"
)

// LogRunKillSchema returns the JSON Schema for the log_run_kill tool.
func LogRunKillSchema() *structpb.Struct {
	s, _ := structpb.NewStruct(map[string]any{
		"type": "object",
		"properties": map[string]any{
			"id": map[string]any{
				"type":        "string",
				"description": "Process ID to kill",
			},
		},
		"required": []any{"id"},
	})
	return s
}

// LogRunKill returns a tool handler that kills a running background process.
func LogRunKill(runner *Runner) func(context.Context, *pluginv1.ToolRequest) (*pluginv1.ToolResponse, error) {
	return func(ctx context.Context, req *pluginv1.ToolRequest) (*pluginv1.ToolResponse, error) {
		if err := helpers.ValidateRequired(req.Arguments, "id"); err != nil {
			return helpers.ErrorResult("validation_error", err.Error()), nil
		}

		id := helpers.GetString(req.Arguments, "id")

		if err := runner.PM.KillProcess(id); err != nil {
			return helpers.ErrorResult("kill_error", err.Error()), nil
		}

		return helpers.TextResult(fmt.Sprintf("Process **%s** killed successfully.", id)), nil
	}
}
