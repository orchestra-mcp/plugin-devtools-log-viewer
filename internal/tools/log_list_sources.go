package tools

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	pluginv1 "github.com/orchestra-mcp/gen-go/orchestra/plugin/v1"
	"github.com/orchestra-mcp/sdk-go/helpers"
	"google.golang.org/protobuf/types/known/structpb"
)

// LogListSourcesSchema returns the JSON Schema for the log_list_sources tool.
func LogListSourcesSchema() *structpb.Struct {
	s, _ := structpb.NewStruct(map[string]any{
		"type": "object",
		"properties": map[string]any{
			"directory": map[string]any{
				"type":        "string",
				"description": "Directory to search for *.log files. If omitted, checks common log directories.",
			},
		},
	})
	return s
}

// LogListSources returns a tool handler that lists log files.
func LogListSources() func(context.Context, *pluginv1.ToolRequest) (*pluginv1.ToolResponse, error) {
	return func(ctx context.Context, req *pluginv1.ToolRequest) (*pluginv1.ToolResponse, error) {
		directory := helpers.GetString(req.Arguments, "directory")

		var searchDirs []string
		if directory != "" {
			searchDirs = []string{directory}
		} else {
			// Common log directories.
			homeDir, err := os.UserHomeDir()
			if err != nil {
				homeDir = ""
			}
			searchDirs = []string{"/var/log", "/tmp"}
			if homeDir != "" {
				searchDirs = append(searchDirs, filepath.Join(homeDir, "Library", "Logs"))
			}
		}

		type logEntry struct {
			path    string
			size    int64
			modTime string
		}

		var found []logEntry
		for _, dir := range searchDirs {
			info, err := os.Stat(dir)
			if err != nil || !info.IsDir() {
				continue
			}
			matches, err := filepath.Glob(filepath.Join(dir, "*.log"))
			if err != nil {
				continue
			}
			for _, match := range matches {
				fi, err := os.Stat(match)
				if err != nil {
					continue
				}
				found = append(found, logEntry{
					path:    match,
					size:    fi.Size(),
					modTime: fi.ModTime().Format("2006-01-02 15:04:05"),
				})
			}
		}

		if len(found) == 0 {
			dirs := strings.Join(searchDirs, ", ")
			return helpers.TextResult(fmt.Sprintf("No .log files found in: %s", dirs)), nil
		}

		var out strings.Builder
		fmt.Fprintf(&out, "Found %d log file(s):\n\n", len(found))
		fmt.Fprintf(&out, "%-60s %10s  %s\n", "Path", "Size", "Modified")
		fmt.Fprintf(&out, "%s\n", strings.Repeat("-", 90))
		for _, e := range found {
			fmt.Fprintf(&out, "%-60s %10s  %s\n", e.path, formatBytes(e.size), e.modTime)
		}

		return helpers.TextResult(out.String()), nil
	}
}

func formatBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}
