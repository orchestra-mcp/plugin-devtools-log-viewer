package tools

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	pluginv1 "github.com/orchestra-mcp/gen-go/orchestra/plugin/v1"
	"github.com/orchestra-mcp/sdk-go/helpers"
	"google.golang.org/protobuf/types/known/structpb"
)

// LogRunOutputSchema returns the JSON Schema for the log_run_output tool.
func LogRunOutputSchema() *structpb.Struct {
	s, _ := structpb.NewStruct(map[string]any{
		"type": "object",
		"properties": map[string]any{
			"id": map[string]any{
				"type":        "string",
				"description": "Process ID returned by log_run",
			},
			"lines": map[string]any{
				"type":        "number",
				"description": "Number of recent lines to return (default 100, 0 for all buffered lines)",
			},
			"pattern": map[string]any{
				"type":        "string",
				"description": "Optional regex pattern to filter output lines",
			},
		},
		"required": []any{"id"},
	})
	return s
}

// LogRunOutput returns a tool handler that retrieves captured output from a background process.
func LogRunOutput(runner *Runner) func(context.Context, *pluginv1.ToolRequest) (*pluginv1.ToolResponse, error) {
	return func(ctx context.Context, req *pluginv1.ToolRequest) (*pluginv1.ToolResponse, error) {
		if err := helpers.ValidateRequired(req.Arguments, "id"); err != nil {
			return helpers.ErrorResult("validation_error", err.Error()), nil
		}

		id := helpers.GetString(req.Arguments, "id")
		n := helpers.GetInt(req.Arguments, "lines")
		pattern := helpers.GetString(req.Arguments, "pattern")

		proc := runner.PM.GetProcess(id)
		if proc == nil {
			return helpers.ErrorResult("not_found",
				fmt.Sprintf("no process found with ID %q", id)), nil
		}

		var allLines []string
		if n <= 0 {
			allLines = proc.GetAllLines()
		} else {
			allLines = proc.GetLastNLines(n)
		}

		if pattern != "" {
			re, err := regexp.Compile(pattern)
			if err != nil {
				return helpers.ErrorResult("invalid_pattern",
					fmt.Sprintf("invalid regex %q: %v", pattern, err)), nil
			}
			var filtered []string
			for _, line := range allLines {
				if re.MatchString(line) {
					filtered = append(filtered, line)
				}
			}
			allLines = filtered
		}

		if len(allLines) == 0 {
			return helpers.TextResult(fmt.Sprintf(
				"No output lines for process %s (total captured: %d)",
				id, proc.GetTotalLines())), nil
		}

		var b strings.Builder
		fmt.Fprintf(&b, "## Output: %s (%d lines)\n\n```\n", id, len(allLines))
		for _, line := range allLines {
			fmt.Fprintf(&b, "%s\n", line)
		}
		b.WriteString("```\n")

		return helpers.TextResult(b.String()), nil
	}
}
