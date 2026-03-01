package tools

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	pluginv1 "github.com/orchestra-mcp/gen-go/orchestra/plugin/v1"
	"google.golang.org/protobuf/types/known/structpb"
)

// callTool is a helper that builds a ToolRequest from a map of args and invokes
// the given handler. The test is failed immediately on any structural error.
func callTool(t *testing.T, handler func(context.Context, *pluginv1.ToolRequest) (*pluginv1.ToolResponse, error), args map[string]any) *pluginv1.ToolResponse {
	t.Helper()
	s, err := structpb.NewStruct(args)
	if err != nil {
		t.Fatalf("NewStruct: %v", err)
	}
	resp, err := handler(context.Background(), &pluginv1.ToolRequest{Arguments: s})
	if err != nil {
		t.Fatalf("handler returned unexpected error: %v", err)
	}
	return resp
}

// isError returns true when the response represents a tool-level error.
// ErrorResult sets Success=false with ErrorCode populated.
func isError(resp *pluginv1.ToolResponse) bool {
	if resp == nil {
		return false
	}
	return !resp.Success
}

// getText extracts the "text" field from a successful ToolResponse.
func getText(resp *pluginv1.ToolResponse) string {
	if resp == nil {
		return ""
	}
	r := resp.GetResult()
	if r == nil {
		return ""
	}
	f := r.GetFields()
	if f == nil {
		return ""
	}
	if tf, ok := f["text"]; ok {
		return tf.GetStringValue()
	}
	return ""
}

// writeLines writes the given lines (joined with "\n") to a new file inside dir.
func writeLines(t *testing.T, dir, name string, lines []string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	content := strings.Join(lines, "\n") + "\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile %s: %v", path, err)
	}
	return path
}

// makeLines generates n lines of the form "line 001", "line 002", …
func makeLines(n int) []string {
	lines := make([]string, n)
	for i := range lines {
		lines[i] = fmt.Sprintf("line %03d", i+1)
	}
	return lines
}

// ─── LogTail ─────────────────────────────────────────────────────────────────

func TestLogTail_HappyPath(t *testing.T) {
	dir := t.TempDir()
	path := writeLines(t, dir, "app.log", makeLines(100))

	handler := LogTail()
	resp := callTool(t, handler, map[string]any{"path": path, "lines": float64(10)})

	if isError(resp) {
		t.Fatalf("expected success, got error: %s", resp.ErrorMessage)
	}
	text := getText(resp)
	// The last 10 lines should be 091 … 100.
	for i := 91; i <= 100; i++ {
		want := fmt.Sprintf("line %03d", i)
		if !strings.Contains(text, want) {
			t.Errorf("expected %q in output, got:\n%s", want, text)
		}
	}
	// The first 90 lines should not appear.
	if strings.Contains(text, "line 001") {
		t.Errorf("line 001 should not be in the tail output")
	}
}

func TestLogTail_DefaultLines(t *testing.T) {
	dir := t.TempDir()
	// Write 30 lines — fewer than the 50-line default.
	path := writeLines(t, dir, "small.log", makeLines(30))

	handler := LogTail()
	resp := callTool(t, handler, map[string]any{"path": path})

	if isError(resp) {
		t.Fatalf("expected success, got error: %s", resp.ErrorMessage)
	}
	text := getText(resp)
	// All 30 lines should be present when the file is smaller than the default.
	if !strings.Contains(text, "line 001") || !strings.Contains(text, "line 030") {
		t.Errorf("expected all 30 lines; got:\n%s", text)
	}
}

func TestLogTail_MissingPath(t *testing.T) {
	handler := LogTail()
	resp := callTool(t, handler, map[string]any{"path": "/does/not/exist/file.log"})

	if !isError(resp) {
		t.Fatalf("expected error response for missing file, got success")
	}
	if resp.ErrorCode != "open_error" {
		t.Errorf("expected error code 'open_error', got %q", resp.ErrorCode)
	}
}

func TestLogTail_SmallFile(t *testing.T) {
	dir := t.TempDir()
	path := writeLines(t, dir, "tiny.log", []string{"alpha", "beta", "gamma"})

	handler := LogTail()
	resp := callTool(t, handler, map[string]any{"path": path, "lines": float64(10)})

	if isError(resp) {
		t.Fatalf("expected success, got error: %s", resp.ErrorMessage)
	}
	text := getText(resp)
	for _, want := range []string{"alpha", "beta", "gamma"} {
		if !strings.Contains(text, want) {
			t.Errorf("expected %q in output, got:\n%s", want, text)
		}
	}
}

func TestLogTail_MissingPathArg(t *testing.T) {
	handler := LogTail()
	resp := callTool(t, handler, map[string]any{})

	if !isError(resp) {
		t.Fatalf("expected validation error when path is missing")
	}
	if resp.ErrorCode != "validation_error" {
		t.Errorf("expected error code 'validation_error', got %q", resp.ErrorCode)
	}
}

// ─── LogWatch ────────────────────────────────────────────────────────────────

func TestLogWatch_HappyPath(t *testing.T) {
	dir := t.TempDir()
	path := writeLines(t, dir, "watch.log", makeLines(50))

	handler := LogWatch()
	resp := callTool(t, handler, map[string]any{"path": path, "lines": float64(5)})

	if isError(resp) {
		t.Fatalf("expected success, got error: %s", resp.ErrorMessage)
	}
	text := getText(resp)
	// The snapshot header should mention the file path.
	if !strings.Contains(text, "[snapshot") {
		t.Errorf("expected snapshot prefix in output; got:\n%s", text)
	}
	// The last 5 lines are 046 … 050.
	for i := 46; i <= 50; i++ {
		want := fmt.Sprintf("line %03d", i)
		if !strings.Contains(text, want) {
			t.Errorf("expected %q in snapshot output, got:\n%s", want, text)
		}
	}
}

func TestLogWatch_MissingPath(t *testing.T) {
	handler := LogWatch()
	resp := callTool(t, handler, map[string]any{"path": "/no/such/file.log"})

	if !isError(resp) {
		t.Fatalf("expected error for nonexistent file")
	}
	if resp.ErrorCode != "open_error" {
		t.Errorf("expected 'open_error', got %q", resp.ErrorCode)
	}
}

func TestLogWatch_DefaultLines(t *testing.T) {
	dir := t.TempDir()
	// 20 lines is exactly the default; write 25 so the default slices to 20.
	path := writeLines(t, dir, "default.log", makeLines(25))

	handler := LogWatch()
	resp := callTool(t, handler, map[string]any{"path": path})

	if isError(resp) {
		t.Fatalf("expected success, got error: %s", resp.ErrorMessage)
	}
	text := getText(resp)
	// Lines 6-25 should be present; lines 1-5 should not (default=20).
	if strings.Contains(text, "line 001") {
		t.Errorf("line 001 should not appear with default (20) watch lines; got:\n%s", text)
	}
	if !strings.Contains(text, "line 025") {
		t.Errorf("expected line 025 in default watch output; got:\n%s", text)
	}
}

func TestLogWatch_MissingPathArg(t *testing.T) {
	handler := LogWatch()
	resp := callTool(t, handler, map[string]any{})

	if !isError(resp) {
		t.Fatalf("expected validation error when path is missing")
	}
	if resp.ErrorCode != "validation_error" {
		t.Errorf("expected 'validation_error', got %q", resp.ErrorCode)
	}
}

// ─── LogSearch ───────────────────────────────────────────────────────────────

func TestLogSearch_MatchFound(t *testing.T) {
	dir := t.TempDir()
	lines := []string{
		"INFO starting server",
		"INFO server ready",
		"ERROR disk full",
		"INFO request received",
		"ERROR connection timeout",
	}
	path := writeLines(t, dir, "search.log", lines)

	handler := LogSearch()
	resp := callTool(t, handler, map[string]any{"path": path, "pattern": "ERROR"})

	if isError(resp) {
		t.Fatalf("expected success, got error: %s", resp.ErrorMessage)
	}
	text := getText(resp)
	if !strings.Contains(text, "ERROR disk full") {
		t.Errorf("expected 'ERROR disk full' in output; got:\n%s", text)
	}
	if !strings.Contains(text, "ERROR connection timeout") {
		t.Errorf("expected 'ERROR connection timeout' in output; got:\n%s", text)
	}
}

func TestLogSearch_NoMatch(t *testing.T) {
	dir := t.TempDir()
	path := writeLines(t, dir, "nomatch.log", []string{"INFO hello", "INFO world"})

	handler := LogSearch()
	resp := callTool(t, handler, map[string]any{"path": path, "pattern": "CRITICAL"})

	if isError(resp) {
		t.Fatalf("expected success (no-match) response, got error: %s", resp.ErrorMessage)
	}
	text := getText(resp)
	if !strings.Contains(text, "No matches") {
		t.Errorf("expected 'No matches' message; got:\n%s", text)
	}
}

func TestLogSearch_InvalidRegex(t *testing.T) {
	dir := t.TempDir()
	path := writeLines(t, dir, "regex.log", []string{"some line"})

	handler := LogSearch()
	resp := callTool(t, handler, map[string]any{"path": path, "pattern": "[invalid"})

	if !isError(resp) {
		t.Fatalf("expected error for invalid regex, got success")
	}
	if resp.ErrorCode != "invalid_pattern" {
		t.Errorf("expected 'invalid_pattern', got %q", resp.ErrorCode)
	}
}

func TestLogSearch_ContextLines(t *testing.T) {
	dir := t.TempDir()
	lines := []string{
		"INFO  before-1",
		"INFO  before-2",
		"ERROR the failure",
		"INFO  after-1",
		"INFO  after-2",
		"INFO  far-away",
	}
	path := writeLines(t, dir, "ctx.log", lines)

	handler := LogSearch()
	resp := callTool(t, handler, map[string]any{
		"path":          path,
		"pattern":       "ERROR",
		"context_lines": float64(2),
	})

	if isError(resp) {
		t.Fatalf("expected success, got error: %s", resp.ErrorMessage)
	}
	text := getText(resp)
	// The two lines before and two after the ERROR line should appear.
	for _, want := range []string{"before-1", "before-2", "the failure", "after-1", "after-2"} {
		if !strings.Contains(text, want) {
			t.Errorf("expected %q in context output; got:\n%s", want, text)
		}
	}
	// "far-away" is 3 lines away from the match so it should not appear.
	if strings.Contains(text, "far-away") {
		t.Errorf("'far-away' should not appear in context-2 output; got:\n%s", text)
	}
}

func TestLogSearch_MissingPathArg(t *testing.T) {
	handler := LogSearch()
	resp := callTool(t, handler, map[string]any{"pattern": "ERROR"})

	if !isError(resp) {
		t.Fatalf("expected validation error when path is missing")
	}
	if resp.ErrorCode != "validation_error" {
		t.Errorf("expected 'validation_error', got %q", resp.ErrorCode)
	}
}

// ─── LogParse ────────────────────────────────────────────────────────────────

func TestLogParse_JSONFormat(t *testing.T) {
	dir := t.TempDir()
	lines := []string{
		`{"level":"info","msg":"server started","port":8080}`,
		`{"level":"error","msg":"disk full","code":500}`,
	}
	path := writeLines(t, dir, "json.log", lines)

	handler := LogParse()
	resp := callTool(t, handler, map[string]any{"path": path, "format": "json"})

	if isError(resp) {
		t.Fatalf("expected success, got error: %s", resp.ErrorMessage)
	}
	text := getText(resp)
	if !strings.Contains(text, "server started") {
		t.Errorf("expected 'server started' in parsed output; got:\n%s", text)
	}
	if !strings.Contains(text, "disk full") {
		t.Errorf("expected 'disk full' in parsed output; got:\n%s", text)
	}
	// Summary line should report 2 parsed entries.
	if !strings.Contains(text, "Parsed 2 entries") {
		t.Errorf("expected 'Parsed 2 entries' in output; got:\n%s", text)
	}
}

func TestLogParse_SyslogFormat(t *testing.T) {
	dir := t.TempDir()
	lines := []string{
		"Jan  2 15:04:05 myhost myproc[123]: connection established",
		"Jan  2 15:04:06 myhost myproc[123]: connection closed",
	}
	path := writeLines(t, dir, "syslog.log", lines)

	handler := LogParse()
	resp := callTool(t, handler, map[string]any{"path": path, "format": "syslog"})

	if isError(resp) {
		t.Fatalf("expected success, got error: %s", resp.ErrorMessage)
	}
	text := getText(resp)
	if !strings.Contains(text, "connection established") {
		t.Errorf("expected 'connection established' in syslog output; got:\n%s", text)
	}
	if !strings.Contains(text, "Parsed 2 syslog entries") {
		t.Errorf("expected 'Parsed 2 syslog entries'; got:\n%s", text)
	}
}

func TestLogParse_AutoDetectsJSON(t *testing.T) {
	dir := t.TempDir()
	lines := []string{
		`{"level":"warn","msg":"low memory"}`,
		`{"level":"info","msg":"gc ran"}`,
	}
	path := writeLines(t, dir, "auto.log", lines)

	handler := LogParse()
	// No format argument — should auto-detect JSON.
	resp := callTool(t, handler, map[string]any{"path": path})

	if isError(resp) {
		t.Fatalf("expected success with auto-detect, got error: %s", resp.ErrorMessage)
	}
	text := getText(resp)
	if !strings.Contains(text, "low memory") {
		t.Errorf("expected 'low memory' in auto-detected JSON output; got:\n%s", text)
	}
}

func TestLogParse_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	lines := []string{
		"this is not json at all",
		"neither is this line",
	}
	path := writeLines(t, dir, "bad.log", lines)

	handler := LogParse()
	resp := callTool(t, handler, map[string]any{"path": path, "format": "json"})

	if !isError(resp) {
		t.Fatalf("expected parse_error for all-bad JSON lines, got success")
	}
	if resp.ErrorCode != "parse_error" {
		t.Errorf("expected 'parse_error', got %q", resp.ErrorCode)
	}
}

func TestLogParse_MissingPathArg(t *testing.T) {
	handler := LogParse()
	resp := callTool(t, handler, map[string]any{})

	if !isError(resp) {
		t.Fatalf("expected validation error when path is missing")
	}
	if resp.ErrorCode != "validation_error" {
		t.Errorf("expected 'validation_error', got %q", resp.ErrorCode)
	}
}

// ─── LogListSources ───────────────────────────────────────────────────────────

func TestLogListSources_WithDirectory(t *testing.T) {
	dir := t.TempDir()
	// Create two .log files and one non-log file.
	for _, name := range []string{"access.log", "error.log", "README.md"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("data\n"), 0o644); err != nil {
			t.Fatalf("WriteFile: %v", err)
		}
	}

	handler := LogListSources()
	resp := callTool(t, handler, map[string]any{"directory": dir})

	if isError(resp) {
		t.Fatalf("expected success, got error: %s", resp.ErrorMessage)
	}
	text := getText(resp)
	if !strings.Contains(text, "access.log") {
		t.Errorf("expected 'access.log' in output; got:\n%s", text)
	}
	if !strings.Contains(text, "error.log") {
		t.Errorf("expected 'error.log' in output; got:\n%s", text)
	}
	// Non-log file should not appear.
	if strings.Contains(text, "README.md") {
		t.Errorf("README.md should not appear in log listing; got:\n%s", text)
	}
	// Output should report 2 files found.
	if !strings.Contains(text, "Found 2 log file") {
		t.Errorf("expected 'Found 2 log file(s)'; got:\n%s", text)
	}
}

func TestLogListSources_EmptyDirectory(t *testing.T) {
	dir := t.TempDir()
	// No .log files — only a non-log file.
	if err := os.WriteFile(filepath.Join(dir, "notes.txt"), []byte("hi\n"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	handler := LogListSources()
	resp := callTool(t, handler, map[string]any{"directory": dir})

	if isError(resp) {
		t.Fatalf("expected success (empty result), got error: %s", resp.ErrorMessage)
	}
	text := getText(resp)
	if !strings.Contains(text, "No .log files") {
		t.Errorf("expected 'No .log files' message; got:\n%s", text)
	}
}

func TestLogListSources_MultipleLogFiles(t *testing.T) {
	dir := t.TempDir()
	names := []string{"a.log", "b.log", "c.log"}
	for _, name := range names {
		content := strings.Repeat("x", 1024) // 1 KB each
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
			t.Fatalf("WriteFile: %v", err)
		}
	}

	handler := LogListSources()
	resp := callTool(t, handler, map[string]any{"directory": dir})

	if isError(resp) {
		t.Fatalf("expected success, got error: %s", resp.ErrorMessage)
	}
	text := getText(resp)
	if !strings.Contains(text, "Found 3 log file") {
		t.Errorf("expected 'Found 3 log file(s)'; got:\n%s", text)
	}
	for _, name := range names {
		if !strings.Contains(text, name) {
			t.Errorf("expected %q in output; got:\n%s", name, text)
		}
	}
}
