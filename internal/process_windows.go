//go:build windows

package internal

import (
	"bufio"
	"context"
	"crypto/rand"
	"fmt"
	"os"
	"os/exec"
	"sync"
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

func (p *ManagedProcess) IsRunning() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.status == StatusRunning
}

func (p *ManagedProcess) Status() ProcessStatus {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.status
}

func (p *ManagedProcess) UptimeSeconds() float64 {
	p.mu.RLock()
	defer p.mu.RUnlock()
	end := time.Now()
	if !p.EndedAt.IsZero() {
		end = p.EndedAt
	}
	return end.Sub(p.StartedAt).Seconds()
}

func (p *ManagedProcess) ErrorString() string {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if p.Err != nil {
		return p.Err.Error()
	}
	return ""
}

// Kill terminates the process on Windows.
func (p *ManagedProcess) Kill() error {
	p.mu.RLock()
	if p.status != StatusRunning || p.Cmd == nil || p.Cmd.Process == nil {
		p.mu.RUnlock()
		return nil
	}
	p.mu.RUnlock()
	return p.Cmd.Process.Kill()
}

func GenerateProcessID() string {
	b := make([]byte, 4)
	_, _ = rand.Read(b)
	return fmt.Sprintf("proc-%x", b)
}

func StartProcess(ctx context.Context, id, command, workDir string, bufSize int) *ManagedProcess {
	if bufSize <= 0 {
		bufSize = 1000
	}

	childCtx, cancel := context.WithCancel(ctx)

	cmd := exec.CommandContext(childCtx, "cmd", "/C", command)
	cmd.Dir = workDir
	cmd.Env = os.Environ()

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
	cmd.Stderr = cmd.Stdout

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

	go func() {
		scanner := bufio.NewScanner(stdoutPipe)
		scanner.Buffer(make([]byte, 0, 256*1024), 256*1024)
		for scanner.Scan() {
			mp.Output.Write(scanner.Text())
		}
	}()

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
