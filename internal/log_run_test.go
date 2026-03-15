package internal

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/orchestra-mcp/plugin-devtools-log-viewer/internal/tools"
	pluginv1 "github.com/orchestra-mcp/gen-go/orchestra/plugin/v1"
	"google.golang.org/protobuf/types/known/structpb"
)

// setupRunner creates a ToolsPlugin and Runner for testing.
func setupRunner(t *testing.T) *tools.Runner {
	t.Helper()
	tp := NewToolsPlugin()
	t.Cleanup(tp.KillAll)
	return &tools.Runner{PM: tp}
}

// callRunnerTool is a helper that builds a ToolRequest and invokes the handler.
func callRunnerTool(t *testing.T, handler func(context.Context, *pluginv1.ToolRequest) (*pluginv1.ToolResponse, error), args map[string]any) *pluginv1.ToolResponse {
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

func getRunnerText(resp *pluginv1.ToolResponse) string {
	if resp == nil || resp.GetResult() == nil {
		return ""
	}
	if f := resp.GetResult().GetFields(); f != nil {
		if tf, ok := f["text"]; ok {
			return tf.GetStringValue()
		}
	}
	return ""
}

// extractID extracts the process ID from log_run output.
func extractID(text string) string {
	for _, line := range strings.Split(text, "\n") {
		if strings.Contains(line, "**ID:**") {
			parts := strings.Split(line, "**ID:** ")
			if len(parts) > 1 {
				return strings.TrimSpace(parts[1])
			}
		}
	}
	return ""
}

// ─── log_run ─────────────────────────────────────────────────────────────────

func TestLogRun_HappyPath(t *testing.T) {
	runner := setupRunner(t)
	resp := callRunnerTool(t, tools.LogRun(runner), map[string]any{
		"command":           "echo hello",
		"working_directory": t.TempDir(),
	})

	if !resp.Success {
		t.Fatalf("expected success, got error: %s", resp.ErrorMessage)
	}
	text := getRunnerText(resp)
	if !strings.Contains(text, "Process Started") {
		t.Errorf("expected 'Process Started' in output; got:\n%s", text)
	}
	if !strings.Contains(text, "proc-") {
		t.Errorf("expected process ID in output; got:\n%s", text)
	}
}

func TestLogRun_MissingCommand(t *testing.T) {
	runner := setupRunner(t)
	resp := callRunnerTool(t, tools.LogRun(runner), map[string]any{})

	if resp.Success {
		t.Fatalf("expected validation error when command is missing")
	}
	if resp.ErrorCode != "validation_error" {
		t.Errorf("expected 'validation_error', got %q", resp.ErrorCode)
	}
}

// ─── log_run_status ──────────────────────────────────────────────────────────

func TestLogRunStatus_Finished(t *testing.T) {
	runner := setupRunner(t)

	runResp := callRunnerTool(t, tools.LogRun(runner), map[string]any{
		"command":           "echo done",
		"working_directory": t.TempDir(),
	})
	id := extractID(getRunnerText(runResp))
	if id == "" {
		t.Fatal("failed to extract process ID")
	}

	time.Sleep(500 * time.Millisecond)

	resp := callRunnerTool(t, tools.LogRunStatus(runner), map[string]any{"id": id})
	if !resp.Success {
		t.Fatalf("expected success, got error: %s", resp.ErrorMessage)
	}
	text := getRunnerText(resp)
	if !strings.Contains(text, "finished") {
		t.Errorf("expected 'finished' status; got:\n%s", text)
	}
	if !strings.Contains(text, "Exit Code:** 0") {
		t.Errorf("expected exit code 0; got:\n%s", text)
	}
}

func TestLogRunStatus_NotFound(t *testing.T) {
	runner := setupRunner(t)
	resp := callRunnerTool(t, tools.LogRunStatus(runner), map[string]any{"id": "proc-nonexistent"})

	if resp.Success {
		t.Fatalf("expected not_found error")
	}
	if resp.ErrorCode != "not_found" {
		t.Errorf("expected 'not_found', got %q", resp.ErrorCode)
	}
}

// ─── log_run_output ──────────────────────────────────────────────────────────

func TestLogRunOutput_CapturesStdout(t *testing.T) {
	runner := setupRunner(t)

	runResp := callRunnerTool(t, tools.LogRun(runner), map[string]any{
		"command":           "echo line1 && echo line2 && echo line3",
		"working_directory": t.TempDir(),
	})
	id := extractID(getRunnerText(runResp))

	time.Sleep(500 * time.Millisecond)

	resp := callRunnerTool(t, tools.LogRunOutput(runner), map[string]any{"id": id})
	if !resp.Success {
		t.Fatalf("expected success, got error: %s", resp.ErrorMessage)
	}
	text := getRunnerText(resp)
	for _, want := range []string{"line1", "line2", "line3"} {
		if !strings.Contains(text, want) {
			t.Errorf("expected %q in output; got:\n%s", want, text)
		}
	}
}

func TestLogRunOutput_CapturesStderr(t *testing.T) {
	runner := setupRunner(t)

	runResp := callRunnerTool(t, tools.LogRun(runner), map[string]any{
		"command":           "echo stderr_msg >&2",
		"working_directory": t.TempDir(),
	})
	id := extractID(getRunnerText(runResp))

	time.Sleep(500 * time.Millisecond)

	resp := callRunnerTool(t, tools.LogRunOutput(runner), map[string]any{"id": id})
	text := getRunnerText(resp)
	if !strings.Contains(text, "stderr_msg") {
		t.Errorf("expected stderr to be captured; got:\n%s", text)
	}
}

func TestLogRunOutput_PatternFilter(t *testing.T) {
	runner := setupRunner(t)

	runResp := callRunnerTool(t, tools.LogRun(runner), map[string]any{
		"command":           "echo INFO_ok && echo ERROR_bad && echo INFO_good",
		"working_directory": t.TempDir(),
	})
	id := extractID(getRunnerText(runResp))

	time.Sleep(500 * time.Millisecond)

	resp := callRunnerTool(t, tools.LogRunOutput(runner), map[string]any{
		"id":      id,
		"pattern": "ERROR",
	})
	text := getRunnerText(resp)
	if !strings.Contains(text, "ERROR_bad") {
		t.Errorf("expected 'ERROR_bad' in filtered output; got:\n%s", text)
	}
	if strings.Contains(text, "INFO_ok") || strings.Contains(text, "INFO_good") {
		t.Errorf("INFO lines should be filtered out; got:\n%s", text)
	}
}

func TestLogRunOutput_InvalidRegex(t *testing.T) {
	runner := setupRunner(t)

	runResp := callRunnerTool(t, tools.LogRun(runner), map[string]any{
		"command":           "echo test",
		"working_directory": t.TempDir(),
	})
	id := extractID(getRunnerText(runResp))

	time.Sleep(200 * time.Millisecond)

	resp := callRunnerTool(t, tools.LogRunOutput(runner), map[string]any{
		"id":      id,
		"pattern": "[invalid",
	})
	if resp.Success {
		t.Fatalf("expected invalid_pattern error")
	}
	if resp.ErrorCode != "invalid_pattern" {
		t.Errorf("expected 'invalid_pattern', got %q", resp.ErrorCode)
	}
}

// ─── log_run_kill ────────────────────────────────────────────────────────────

func TestLogRunKill_RunningProcess(t *testing.T) {
	runner := setupRunner(t)

	runResp := callRunnerTool(t, tools.LogRun(runner), map[string]any{
		"command":           "sleep 30",
		"working_directory": t.TempDir(),
	})
	id := extractID(getRunnerText(runResp))

	time.Sleep(200 * time.Millisecond)

	resp := callRunnerTool(t, tools.LogRunKill(runner), map[string]any{"id": id})
	if !resp.Success {
		t.Fatalf("expected success, got error: %s", resp.ErrorMessage)
	}

	time.Sleep(200 * time.Millisecond)
	statusResp := callRunnerTool(t, tools.LogRunStatus(runner), map[string]any{"id": id})
	text := getRunnerText(statusResp)
	if strings.Contains(text, "**Status:** running") {
		t.Errorf("expected process to be killed; got:\n%s", text)
	}
}

func TestLogRunKill_NotFound(t *testing.T) {
	runner := setupRunner(t)
	resp := callRunnerTool(t, tools.LogRunKill(runner), map[string]any{"id": "proc-nonexistent"})

	if resp.Success {
		t.Fatalf("expected kill_error")
	}
	if resp.ErrorCode != "kill_error" {
		t.Errorf("expected 'kill_error', got %q", resp.ErrorCode)
	}
}

// ─── log_run_restart ─────────────────────────────────────────────────────────

func TestLogRunRestart_SameCommand(t *testing.T) {
	runner := setupRunner(t)

	runResp := callRunnerTool(t, tools.LogRun(runner), map[string]any{
		"command":           "sleep 30",
		"working_directory": t.TempDir(),
	})
	id := extractID(getRunnerText(runResp))

	time.Sleep(200 * time.Millisecond)

	resp := callRunnerTool(t, tools.LogRunRestart(runner), map[string]any{"id": id})
	if !resp.Success {
		t.Fatalf("expected success, got error: %s", resp.ErrorMessage)
	}
	text := getRunnerText(resp)
	if !strings.Contains(text, "Restarted") {
		t.Errorf("expected 'Restarted' in output; got:\n%s", text)
	}
	if !strings.Contains(text, id) {
		t.Errorf("expected same ID %s in output; got:\n%s", id, text)
	}
}

func TestLogRunRestart_NotFound(t *testing.T) {
	runner := setupRunner(t)
	resp := callRunnerTool(t, tools.LogRunRestart(runner), map[string]any{"id": "proc-nonexistent"})

	if resp.Success {
		t.Fatalf("expected restart_error")
	}
	if resp.ErrorCode != "restart_error" {
		t.Errorf("expected 'restart_error', got %q", resp.ErrorCode)
	}
}

// ─── log_run_list ────────────────────────────────────────────────────────────

func TestLogRunList_Empty(t *testing.T) {
	runner := setupRunner(t)
	resp := callRunnerTool(t, tools.LogRunList(runner), map[string]any{})

	if !resp.Success {
		t.Fatalf("expected success, got error: %s", resp.ErrorMessage)
	}
	text := getRunnerText(resp)
	if !strings.Contains(text, "No tracked processes") {
		t.Errorf("expected 'No tracked processes'; got:\n%s", text)
	}
}

func TestLogRunList_Multiple(t *testing.T) {
	runner := setupRunner(t)

	callRunnerTool(t, tools.LogRun(runner), map[string]any{
		"command":           "sleep 30",
		"working_directory": t.TempDir(),
	})
	callRunnerTool(t, tools.LogRun(runner), map[string]any{
		"command":           "sleep 30",
		"working_directory": t.TempDir(),
	})

	time.Sleep(200 * time.Millisecond)

	resp := callRunnerTool(t, tools.LogRunList(runner), map[string]any{})
	if !resp.Success {
		t.Fatalf("expected success, got error: %s", resp.ErrorMessage)
	}
	text := getRunnerText(resp)
	if !strings.Contains(text, "Managed Processes (2)") {
		t.Errorf("expected 2 processes; got:\n%s", text)
	}
}
