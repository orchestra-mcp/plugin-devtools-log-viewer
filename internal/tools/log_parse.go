package tools

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"

	pluginv1 "github.com/orchestra-mcp/gen-go/orchestra/plugin/v1"
	"github.com/orchestra-mcp/sdk-go/helpers"
	"google.golang.org/protobuf/types/known/structpb"
)

// LogParseSchema returns the JSON Schema for the log_parse tool.
func LogParseSchema() *structpb.Struct {
	s, _ := structpb.NewStruct(map[string]any{
		"type": "object",
		"properties": map[string]any{
			"path": map[string]any{
				"type":        "string",
				"description": "Absolute path to the log file",
			},
			"format": map[string]any{
				"type":        "string",
				"description": "Log format: json, syslog, or auto (default auto)",
				"enum":        []any{"json", "syslog", "auto"},
			},
		},
		"required": []any{"path"},
	})
	return s
}

// syslogRe matches lines like: Jan  2 15:04:05 hostname proc[pid]: message
var syslogRe = regexp.MustCompile(`^(\w{3}\s+\d{1,2}\s+\d{2}:\d{2}:\d{2})\s+(\S+)\s+([^:]+):\s+(.*)$`)

// LogParse returns a tool handler that parses a log file in the specified format.
func LogParse() func(context.Context, *pluginv1.ToolRequest) (*pluginv1.ToolResponse, error) {
	return func(ctx context.Context, req *pluginv1.ToolRequest) (*pluginv1.ToolResponse, error) {
		if err := helpers.ValidateRequired(req.Arguments, "path"); err != nil {
			return helpers.ErrorResult("validation_error", err.Error()), nil
		}

		path := helpers.GetString(req.Arguments, "path")
		format := helpers.GetStringOr(req.Arguments, "format", "auto")

		f, err := os.Open(path)
		if err != nil {
			return helpers.ErrorResult("open_error", fmt.Sprintf("cannot open %s: %v", path, err)), nil
		}
		defer f.Close()

		var lines []string
		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			lines = append(lines, scanner.Text())
		}
		if err := scanner.Err(); err != nil {
			return helpers.ErrorResult("read_error", fmt.Sprintf("error reading %s: %v", path, err)), nil
		}

		switch format {
		case "json":
			return parseJSON(lines, path)
		case "syslog":
			return parseSyslog(lines, path)
		default: // auto
			// Try JSON first.
			result, err := parseJSON(lines, path)
			if err == nil && result.Success {
				return result, nil
			}
			// Try syslog next.
			result, err = parseSyslog(lines, path)
			if err == nil && result.Success {
				return result, nil
			}
			// Fall back to raw.
			return helpers.TextResult(strings.Join(lines, "\n")), nil
		}
	}
}

func parseJSON(lines []string, path string) (*pluginv1.ToolResponse, error) {
	const maxEntries = 50
	var out strings.Builder
	fmt.Fprintf(&out, "Parsed JSON log: %s\n\n", path)

	parsed := 0
	failed := 0
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var entry map[string]any
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			failed++
			continue
		}
		if parsed >= maxEntries {
			break
		}
		pretty, err := json.MarshalIndent(entry, "", "  ")
		if err != nil {
			continue
		}
		fmt.Fprintf(&out, "%s\n\n", pretty)
		parsed++
	}

	if parsed == 0 {
		return helpers.ErrorResult("parse_error", "no valid JSON lines found"), nil
	}
	fmt.Fprintf(&out, "--- Parsed %d entries, %d failed ---\n", parsed, failed)
	return helpers.TextResult(out.String()), nil
}

func parseSyslog(lines []string, path string) (*pluginv1.ToolResponse, error) {
	var out strings.Builder
	fmt.Fprintf(&out, "Parsed syslog: %s\n\n", path)

	matched := 0
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		m := syslogRe.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		// m[1]=timestamp, m[2]=host, m[3]=process, m[4]=message
		fmt.Fprintf(&out, "[%s] %s %s: %s\n", m[1], m[2], m[3], m[4])
		matched++
	}

	if matched == 0 {
		return helpers.ErrorResult("parse_error", "no syslog-format lines found"), nil
	}
	fmt.Fprintf(&out, "\n--- Parsed %d syslog entries ---\n", matched)
	return helpers.TextResult(out.String()), nil
}
