// Subprocess lifecycle for the LLM backend (TR-ai-005, Story 001
// Implementation Notes): spawn via os/exec, health monitoring via cmd.Wait
// in a goroutine, restart budget, and graceful shutdown (quit → SIGTERM →
// SIGKILL).
package aidialogue

import (
	"bytes"
	"fmt"
	"io"
	"os/exec"
	"strconv"
	"sync"
	"syscall"
	"time"
)

// process wraps an exec.Cmd running the LLM companion binary. The process
// is owned by the Backend, which replaces it on restart.
type process struct {
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout io.ReadCloser
	stderr *errCapture

	mu      sync.Mutex
	exitErr error
	dead    chan struct{} // closed when cmd.Wait returns
}

// spawnProcess starts the LLM binary with the GGUF model and engine
// arguments. The model file must already exist (FindModelFile succeeded).
func spawnProcess(cfg Config, modelPath string) (*process, error) {
	if cfg.BinaryPath == "" {
		return nil, fmt.Errorf("aidialogue: BinaryPath is empty")
	}
	args := []string{
		"--model", modelPath,
		"--ctx-size", strconv.Itoa(cfg.ContextSize),
		"--threads", strconv.Itoa(cfg.Threads),
	}
	args = append(args, cfg.ExtraArgs...)

	cmd := exec.Command(cfg.BinaryPath, args...)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("aidialogue: stdin pipe: %w", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("aidialogue: stdout pipe: %w", err)
	}
	capture := newErrCapture(8 << 10)
	cmd.Stderr = capture

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("aidialogue: spawn %s: %w", cfg.BinaryPath, err)
	}

	p := &process{
		cmd:    cmd,
		stdin:  stdin,
		stdout: stdout,
		stderr: capture,
		dead:   make(chan struct{}),
	}
	go func() {
		err := cmd.Wait()
		p.mu.Lock()
		p.exitErr = err
		p.mu.Unlock()
		close(p.dead)
	}()
	return p, nil
}

// deadCh is closed when the process exits (health monitor: detect exit via
// cmd.Wait in a goroutine, Story 001).
func (p *process) deadCh() <-chan struct{} { return p.dead }

// exitError returns the cmd.Wait error once the process has exited (nil for
// a clean exit).
func (p *process) exitError() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.exitErr
}

// alive reports whether the process is still running (non-blocking).
func (p *process) alive() bool {
	select {
	case <-p.dead:
		return false
	default:
		return true
	}
}

// terminateGracefully sends an interrupt (SIGTERM on unix), waits up to
// grace, then SIGKILL. Returns an error only when the process had to be
// killed — a clean exit is not an error.
func (p *process) terminateGracefully(grace time.Duration) error {
	if !p.alive() {
		return nil
	}
	_ = p.cmd.Process.Signal(syscall.SIGTERM)
	select {
	case <-p.dead:
		return nil
	case <-time.After(grace):
		_ = p.cmd.Process.Kill()
		<-p.dead
		return fmt.Errorf("aidialogue: subprocess killed after grace period")
	}
}

// errCapture is a bounded stderr buffer for the subprocess: the tail of the
// captured output is attached to diagnostics when the process dies
// unexpectedly.
type errCapture struct {
	mu  sync.Mutex
	buf bytes.Buffer
	max int
}

func newErrCapture(max int) *errCapture { return &errCapture{max: max} }

func (e *errCapture) Write(p []byte) (int, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.buf.Len() < e.max {
		remaining := e.max - e.buf.Len()
		if len(p) > remaining {
			p = p[len(p)-remaining:] // keep the tail
		}
		_, _ = e.buf.Write(p)
	}
	return len(p), nil
}

func (e *errCapture) String() string {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.buf.String()
}

// restartPolicy enforces the session restart budget (Story 001: max 1
// restart per session with a 5-min cooldown; GDD CR-5: "No more than 1
// restart per 5 minutes").
type restartPolicy struct {
	max      int
	cooldown time.Duration

	mu    sync.Mutex
	count int
	last  time.Time
}

func newRestartPolicy(max int, cooldown time.Duration) *restartPolicy {
	return &restartPolicy{max: max, cooldown: cooldown}
}

// allow reports whether a restart may be attempted now.
func (r *restartPolicy) allow(now time.Time) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.count >= r.max {
		return false
	}
	return r.last.IsZero() || now.Sub(r.last) >= r.cooldown
}

// record counts a successful restart. Returns the new restart count.
func (r *restartPolicy) record(now time.Time) int {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.count++
	r.last = now
	return r.count
}
