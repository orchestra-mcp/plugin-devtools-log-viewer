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

// LogTailSchema returns the JSON Schema for the log_tail tool.
func LogTailSchema() *structpb.Struct {
	s, _ := structpb.NewStruct(map[string]any{
		"type": "object",
		"properties": map[string]any{
			"path": map[string]any{
				"type":        "string",
				"description": "Absolute path to the log file",
			},
			"lines": map[string]any{
				"type":        "number",
				"description": "Number of lines to return from the end of the file (default 50)",
			},
		},
		"required": []any{"path"},
	})
	return s
}

// LogTail returns a tool handler that reads the last N lines of a log file.
func LogTail() func(context.Context, *pluginv1.ToolRequest) (*pluginv1.ToolResponse, error) {
	return func(ctx context.Context, req *pluginv1.ToolRequest) (*pluginv1.ToolResponse, error) {
		if err := helpers.ValidateRequired(req.Arguments, "path"); err != nil {
			return helpers.ErrorResult("validation_error", err.Error()), nil
		}

		path := helpers.GetString(req.Arguments, "path")
		n := helpers.GetInt(req.Arguments, "lines")
		if n <= 0 {
			n = 50
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
			// The oldest line is at buf[pos%n].
			start := pos % n
			lines = make([]string, n)
			for i := 0; i < n; i++ {
				lines[i] = buf[(start+i)%n]
			}
		}

		return helpers.TextResult(strings.Join(lines, "\n")), nil
	}
}
