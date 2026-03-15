package internal

import (
	"bufio"
	"context"
	"crypto/rand"
	"fmt"
	"os"
	"os/exec"
	"sync"
	"syscall"
	"time"
)

// ProcessStatus describes the state of a managed process.
type ProcessStatus string

const (
	StatusRunning  ProcessStatus = "running"
	StatusFinished ProcessStatus = "finished"
	StatusFailed   ProcessStatus = "failed"
)

// ManagedProcess tracks a background command's lifecycle and output.
type ManagedProcess struct {
	ID        string
	Command   string
	WorkDir   string
	Cmd       *exec.Cmd
	PID       int
	ExitCode  int
	StartedAt time.Time
	EndedAt   time.Time
	Output    *RingBuffer
	Err       error

	mu     sync.RWMutex
	status ProcessStatus
	cancel context.CancelFunc
}

// IsRunning returns true if the process has not yet exited.
func (p *ManagedProcess) IsRunning() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.status == StatusRunning
}

// Status returns the current process status.
func (p *ManagedProcess) Status() ProcessStatus {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.status
}

// UptimeSeconds returns wall-clock time since the process started.
func (p *ManagedProcess) UptimeSeconds() float64 {
	p.mu.RLock()
	defer p.mu.RUnlock()
	end := time.Now()
	if !p.EndedAt.IsZero() {
		end = p.EndedAt
	}
	return end.Sub(p.StartedAt).Seconds()
}

// ErrorString returns the error message if the process failed.
func (p *ManagedProcess) ErrorString() string {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if p.Err != nil {
		return p.Err.Error()
	}
	return ""
}

// Kill sends SIGTERM to the process group, waits 3 seconds, then SIGKILL.
func (p *ManagedProcess) Kill() error {
	p.mu.RLock()
	if p.status != StatusRunning || p.Cmd == nil || p.Cmd.Process == nil {
		p.mu.RUnlock()
		return nil
	}
	pid := p.Cmd.Process.Pid
	p.mu.RUnlock()

	// SIGTERM to the process group (negative PID).
	_ = syscall.Kill(-pid, syscall.SIGTERM)

	done := make(chan struct{})
	go func() {
		_, _ = p.Cmd.Process.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-time.After(3 * time.Second):
		_ = syscall.Kill(-pid, syscall.SIGKILL)
		<-done
		return nil
	}
}

// GenerateProcessID creates a short random ID like "proc-a1b2c3d4".
func GenerateProcessID() string {
	b := make([]byte, 4)
	_, _ = rand.Read(b)
	return fmt.Sprintf("proc-%x", b)
}

// StartProcess launches a command in the background with combined stdout+stderr
// captured into a ring buffer.
func StartProcess(ctx context.Context, id, command, workDir string, bufSize int) *ManagedProcess {
	if bufSize <= 0 {
		bufSize = 1000
	}

	childCtx, cancel := context.WithCancel(ctx)

	cmd := exec.CommandContext(childCtx, "sh", "-c", command)
	cmd.Dir = workDir
	cmd.Env = os.Environ()
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	mp := &ManagedProcess{
		ID:        id,
		Command:   command,
		WorkDir:   workDir,
		Cmd:       cmd,
		status:    StatusRunning,
		StartedAt: time.Now(),
		Output:    NewRingBuffer(bufSize),
		cancel:    cancel,
	}

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		mp.mu.Lock()
		mp.status = StatusFailed
		mp.Err = fmt.Errorf("stdout pipe: %w", err)
		mp.EndedAt = time.Now()
		mp.ExitCode = -1
		mp.mu.Unlock()
		cancel()
		return mp
	}
	cmd.Stderr = cmd.Stdout // merge stderr into stdout pipe

	if err := cmd.Start(); err != nil {
		mp.mu.Lock()
		mp.status = StatusFailed
		mp.Err = err
		mp.EndedAt = time.Now()
		mp.ExitCode = -1
		mp.mu.Unlock()
		cancel()
		return mp
	}

	mp.PID = cmd.Process.Pid

	// Scanner goroutine: reads lines and writes them to the ring buffer.
	go func() {
		scanner := bufio.NewScanner(stdoutPipe)
		scanner.Buffer(make([]byte, 0, 256*1024), 256*1024)
		for scanner.Scan() {
			mp.Output.Write(scanner.Text())
		}
	}()

	// Waiter goroutine: updates status when the process exits.
	go func() {
		waitErr := cmd.Wait()
		mp.mu.Lock()
		mp.EndedAt = time.Now()
		if waitErr != nil {
			mp.status = StatusFailed
			mp.Err = waitErr
			if exitErr, ok := waitErr.(*exec.ExitError); ok {
				mp.ExitCode = exitErr.ExitCode()
			} else {
				mp.ExitCode = -1
			}
		} else {
			mp.status = StatusFinished
			mp.ExitCode = 0
		}
		mp.mu.Unlock()
	}()

	return mp
}
