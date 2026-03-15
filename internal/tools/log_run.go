package tools

import (
	"context"
	"fmt"
	"os"

	pluginv1 "github.com/orchestra-mcp/gen-go/orchestra/plugin/v1"
	"github.com/orchestra-mcp/sdk-go/helpers"
	"google.golang.org/protobuf/types/known/structpb"
)

// LogRunSchema returns the JSON Schema for the log_run tool.
func LogRunSchema() *structpb.Struct {
	s, _ := structpb.NewStruct(map[string]any{
		"type": "object",
		"properties": map[string]any{
			"command": map[string]any{
				"type":        "string",
				"description": "Shell command to run in the background (e.g., 'make dev', 'npm start')",
			},
			"working_directory": map[string]any{
				"type":        "string",
				"description": "Working directory for the command. Defaults to ORCHESTRA_WORKSPACE or cwd.",
			},
		},
		"required": []any{"command"},
	})
	return s
}

// LogRun returns a tool handler that starts a background process and returns its ID.
func LogRun(runner *Runner) func(context.Context, *pluginv1.ToolRequest) (*pluginv1.ToolResponse, error) {
	return func(ctx context.Context, req *pluginv1.ToolRequest) (*pluginv1.ToolResponse, error) {
		if err := helpers.ValidateRequired(req.Arguments, "command"); err != nil {
			return helpers.ErrorResult("validation_error", err.Error()), nil
		}

		command := helpers.GetString(req.Arguments, "command")
		workDir := helpers.GetString(req.Arguments, "working_directory")
		if workDir == "" {
			workDir = os.Getenv("ORCHESTRA_WORKSPACE")
			if workDir == "" {
				workDir, _ = os.Getwd()
			}
		}

		proc := runner.PM.StartProcess(ctx, command, workDir)

		return helpers.TextResult(fmt.Sprintf(
			"## Process Started\n\n"+
				"- **ID:** %s\n"+
				"- **Command:** `%s`\n"+
				"- **Working Dir:** %s\n"+
				"- **PID:** %d\n"+
				"- **Status:** %s\n\n"+
				"Use `log_run_status` with id `%s` to check progress.\n"+
				"Use `log_run_output` with id `%s` to see output.\n",
			proc.GetID(), command, workDir, proc.GetPID(), proc.GetStatus(),
			proc.GetID(), proc.GetID(),
		)), nil
	}
}
