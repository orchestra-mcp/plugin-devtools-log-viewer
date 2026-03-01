package tools

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"regexp"
	"strings"

	pluginv1 "github.com/orchestra-mcp/gen-go/orchestra/plugin/v1"
	"github.com/orchestra-mcp/sdk-go/helpers"
	"google.golang.org/protobuf/types/known/structpb"
)

// LogSearchSchema returns the JSON Schema for the log_search tool.
func LogSearchSchema() *structpb.Struct {
	s, _ := structpb.NewStruct(map[string]any{
		"type": "object",
		"properties": map[string]any{
			"path": map[string]any{
				"type":        "string",
				"description": "Absolute path to the log file",
			},
			"pattern": map[string]any{
				"type":        "string",
				"description": "Regular expression pattern to search for",
			},
			"context_lines": map[string]any{
				"type":        "number",
				"description": "Number of context lines to include before and after each match (default 2)",
			},
		},
		"required": []any{"path", "pattern"},
	})
	return s
}

// LogSearch returns a tool handler that searches a log file for regex matches.
func LogSearch() func(context.Context, *pluginv1.ToolRequest) (*pluginv1.ToolResponse, error) {
	return func(ctx context.Context, req *pluginv1.ToolRequest) (*pluginv1.ToolResponse, error) {
		if err := helpers.ValidateRequired(req.Arguments, "path", "pattern"); err != nil {
			return helpers.ErrorResult("validation_error", err.Error()), nil
		}

		path := helpers.GetString(req.Arguments, "path")
		pattern := helpers.GetString(req.Arguments, "pattern")
		contextLines := helpers.GetInt(req.Arguments, "context_lines")
		if contextLines < 0 {
			contextLines = 2
		}
		if contextLines == 0 {
			// GetInt returns 0 when not set, so treat 0 as the default.
			contextLines = 2
		}

		re, err := regexp.Compile(pattern)
		if err != nil {
			return helpers.ErrorResult("invalid_pattern", fmt.Sprintf("invalid regexp %q: %v", pattern, err)), nil
		}

		f, err := os.Open(path)
		if err != nil {
			return helpers.ErrorResult("open_error", fmt.Sprintf("cannot open %s: %v", path, err)), nil
		}
		defer f.Close()

		// Read all lines into memory so we can emit context.
		var allLines []string
		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			allLines = append(allLines, scanner.Text())
		}
		if err := scanner.Err(); err != nil {
			return helpers.ErrorResult("read_error", fmt.Sprintf("error reading %s: %v", path, err)), nil
		}

		// Track which line numbers to include (using a set).
		include := make(map[int]bool)
		for i, line := range allLines {
			if re.MatchString(line) {
				start := i - contextLines
				if start < 0 {
					start = 0
				}
				end := i + contextLines
				if end >= len(allLines) {
					end = len(allLines) - 1
				}
				for j := start; j <= end; j++ {
					include[j] = true
				}
			}
		}

		if len(include) == 0 {
			return helpers.TextResult(fmt.Sprintf("No matches found for pattern %q in %s", pattern, path)), nil
		}

		// Emit lines in order, adding separator between non-contiguous blocks.
		var out strings.Builder
		fmt.Fprintf(&out, "Matches for %q in %s:\n\n", pattern, path)

		prevLineNo := -2
		for i := 0; i < len(allLines); i++ {
			if !include[i] {
				continue
			}
			if prevLineNo >= 0 && i > prevLineNo+1 {
				out.WriteString("--\n")
			}
			marker := "  "
			if re.MatchString(allLines[i]) {
				marker = "> "
			}
			fmt.Fprintf(&out, "%s%4d: %s\n", marker, i+1, allLines[i])
			prevLineNo = i
		}

		return helpers.TextResult(out.String()), nil
	}
}
