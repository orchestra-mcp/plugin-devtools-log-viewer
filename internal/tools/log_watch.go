package tools

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	pluginv1 "github.com/orchestra-mcp/gen-go/orchestra/plugin/v1"
	"github.com/orchestra-mcp/sdk-go/helpers"
	"google.golang.org/protobuf/types/known/structpb"
)

// LogWatchSchema returns the JSON Schema for the log_watch tool.
func LogWatchSchema() *structpb.Struct {
	s, _ := structpb.NewStruct(map[string]any{
		"type": "object",
		"properties": map[string]any{
			"path": map[string]any{
				"type":        "string",
				"description": "Absolute path to the log file",
			},
			"lines": map[string]any{
				"type":        "number",
				"description": "Number of recent lines to return (default 20)",
			},
		},
		"required": []any{"path"},
	})
	return s
}

// LogWatch returns a tool handler that returns a snapshot of the last N lines.
// Note: streaming is not supported — this returns a point-in-time snapshot.
func LogWatch() func(context.Context, *pluginv1.ToolRequest) (*pluginv1.ToolResponse, error) {
	return func(ctx context.Context, req *pluginv1.ToolRequest) (*pluginv1.ToolResponse, error) {
		if err := helpers.ValidateRequired(req.Arguments, "path"); err != nil {
			return helpers.ErrorResult("validation_error", err.Error()), nil
		}

		path := helpers.GetString(req.Arguments, "path")
		n := helpers.GetInt(req.Arguments, "lines")
		if n <= 0 {
			n = 20
		}

		f, err := os.Open(path)
		if err != nil {
			return helpers.ErrorResult("open_error", fmt.Sprintf("cannot open %s: %v", path, err)), nil
		}
		defer f.Close()

		// Circular buffer to hold the last N lines.
		buf := make([]string, n)
		pos := 0
		count := 0

		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			buf[pos%n] = scanner.Text()
			pos++
			count++
		}
		if err := scanner.Err(); err != nil {
			return helpers.ErrorResult("read_error", fmt.Sprintf("error reading %s: %v", path, err)), nil
		}

		// Reconstruct the lines in order.
		var lines []string
		if count <= n {
			lines = buf[:count]
		} else {
			start := pos % n
			lines = make([]string, n)
			for i := 0; i < n; i++ {
				lines[i] = buf[(start+i)%n]
			}
		}

		result := fmt.Sprintf("[snapshot of last %d lines from %s]\n%s", len(lines), path, strings.Join(lines, "\n"))
		return helpers.TextResult(result), nil
	}
}
