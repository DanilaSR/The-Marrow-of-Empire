// Concrete LLMBackend implementation tying together the IPC layer, the
// subprocess lifecycle, the capability probe, and the background warm-up
// (Story 001; TR-ai-003 graceful degradation: text fallback on model
// failure, auto-restart, degraded status indicator).
package aidialogue

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"
)

// Notification is a one-time, player-facing event emitted by the Backend
// (missing model, failed probe, failed restart). The UI layer renders it as
// the oxblood-knot glyph + plain-language settings notification (GDD CR-5);
// the backend's job ends at emission.
type Notification struct {
	// ID is a stable machine-readable identifier for UI matching.
	ID string
	// Message is the player-facing text (GDD CR-5 copy).
	Message string
}

// Notification IDs emitted at most once per session.
const (
	// NotifModelMissing: the model file was not found (AC-15).
	NotifModelMissing = "model-not-found"
	// NotifChecksumFailed: the model file's SHA-256 did not match the
	// expected digest.
	NotifChecksumFailed = "model-checksum-failed"
	// NotifProbeFailed: the capability probe failed its gate (AC-16).
	NotifProbeFailed = "probe-failed"
	// NotifWarmupFailed: the probe itself errored (subprocess died during
	// warm-up).
	NotifWarmupFailed = "warmup-failed"
	// NotifRestartFailed: a crash restart was refused (budget) or failed
	// (AC-20).
	NotifRestartFailed = "restart-failed"
	// NotifBinaryMissing: the backend binary itself could not be spawned.
	NotifBinaryMissing = "backend-binary-missing"
)

// textModeMessage is the one-time plain-language notification for degraded
// sessions (GDD CR-5).
const textModeMessage = "AI Dialogue unavailable — text mode active. Your hardware may not meet minimum requirements."

// Backend is the concrete LLMBackend (ADR-0008): it owns the subprocess,
// the IPC connection, the status state machine, the restart policy, and the
// warm-up sequence. Start it with Start and stop it with Shutdown. The
// backend is not safe for concurrent use of Start/Shutdown; Generate is
// single-flight (ErrBusy on overlap).
type Backend struct {
	cfg    Config
	logger *slog.Logger

	lifecycle context.Context
	cancel    context.CancelFunc

	mu         sync.Mutex
	status     ModelStatus
	proc       *process
	ipc        *conn
	policy     *restartPolicy
	modelPath  string
	restarting bool
	shutdown   bool
	notified   map[string]bool

	notifCh    chan Notification
	progressCh chan WarmupProgress
	wg         sync.WaitGroup
}

var _ LLMBackend = (*Backend)(nil)

// New creates a backend with the given config and logger. The logger may be
// nil (a discard logger is used). Nothing is spawned until Start.
func New(cfg Config, logger *slog.Logger) *Backend {
	if logger == nil {
		logger = slog.New(slog.DiscardHandler)
	}
	ctx, cancel := context.WithCancel(context.Background())
	return &Backend{
		cfg:        cfg,
		logger:     logger,
		lifecycle:  ctx,
		cancel:     cancel,
		status:     StatusUnloaded,
		policy:     newRestartPolicy(cfg.RestartMax, cfg.RestartCooldown),
		notified:   make(map[string]bool),
		notifCh:    make(chan Notification, 8),
		progressCh: make(chan WarmupProgress, 8),
	}
}

// Notifications returns the channel of one-time events for the UI layer.
func (b *Backend) Notifications() <-chan Notification { return b.notifCh }

// Progress returns the channel of warm-up progress for the main menu UI
// ("The old tongue wakes...", GDD CR-7 step 4).
func (b *Backend) Progress() <-chan WarmupProgress { return b.progressCh }

// Start resolves the model file, spawns the subprocess, and begins the
// background warm-up (probe + greeting pre-fill).
//
// Start never fails the game: every operational failure — missing model file
// (AC-15), slow capability probe (AC-16), missing binary — lands in a
// status (StatusDegraded or StatusError) and the game continues in
// text-fallback mode. The returned error is reserved for programmer errors
// (invalid config) and a cancelled ctx.
func (b *Backend) Start(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if b.cfg.BinaryPath == "" {
		return fmt.Errorf("aidialogue: config BinaryPath must not be empty")
	}

	modelPath, err := FindModelFile(b.cfg)
	if err != nil {
		b.setStatus(StatusDegraded)
		if errors.Is(err, ErrChecksumMismatch) {
			b.notifyOnce(NotifChecksumFailed, textModeMessage)
		} else {
			b.notifyOnce(NotifModelMissing, textModeMessage)
		}
		b.logger.Warn("llm backend degraded: model unavailable", "error", err)
		return nil // game continues in text mode (AC-15)
	}

	proc, err := spawnProcess(b.cfg, modelPath)
	if err != nil {
		b.setStatus(StatusError)
		b.notifyOnce(NotifBinaryMissing, textModeMessage)
		b.logger.Warn("llm backend error: spawn failed", "error", err)
		return nil // game continues in text mode
	}
	c := newConn(proc.stdin, proc.stdout)

	b.mu.Lock()
	b.proc = proc
	b.ipc = c
	b.modelPath = modelPath
	b.status = StatusLoading
	b.mu.Unlock()

	b.wg.Add(2)
	go b.monitor(proc)
	go b.runWarmup(c)

	b.logger.Info("llm backend started", "model", modelPath, "binary", b.cfg.BinaryPath)
	return nil
}

// Status implements LLMBackend.
func (b *Backend) Status() ModelStatus {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.status
}

// Generate implements LLMBackend: sends the prompt to the subprocess and
// returns the generated text. It is single-flight — a concurrent call
// returns ErrBusy (the Story 004 state machine owns queueing, depth 1, most
// recent wins). The context must carry a deadline; if it has none,
// Config.DefaultGenerateTimeout is applied.
//
// On subprocess death during generation, the current response falls back to
// text (the caller receives ErrModelCrashed, wrapping the transport error)
// and the backend auto-restarts once per session (AC-20, GDD CR-5). Callers
// retry after Status returns to StatusReady. Timeouts/cancellations leave
// the backend Ready — per GDD the next query retries normally.
func (b *Backend) Generate(ctx context.Context, prompt string) (string, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if _, hasDeadline := ctx.Deadline(); !hasDeadline {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, b.cfg.DefaultGenerateTimeout)
		defer cancel()
	}

	b.mu.Lock()
	switch b.status {
	case StatusReady:
	case StatusGenerating:
		b.mu.Unlock()
		return "", ErrBusy
	default:
		b.mu.Unlock()
		return "", ErrNotReady
	}
	b.status = StatusGenerating
	c := b.ipc
	proc := b.proc
	b.mu.Unlock()

	resp, err := c.request(ctx, &PromptRequest{
		Prompt:    prompt,
		MaxTokens: b.cfg.MaxTokens,
		Temp:      b.cfg.Temperature,
	})

	b.mu.Lock()
	if b.proc != proc {
		// A restart replaced (or is replacing) the subprocess; the restart
		// path owns the status.
		b.mu.Unlock()
		if err != nil {
			return "", fmt.Errorf("%w: %v", ErrModelCrashed, err)
		}
		return "", ErrModelCrashed
	}
	// The pipe can report the exit (EOF) a moment before cmd.Wait closes the
	// process's dead channel, so an ErrProcessExited failure is a crash even
	// while alive() is still true (AC-20).
	if !proc.alive() || errors.Is(err, ErrProcessExited) {
		b.status = StatusLoading // the restart path moves it on
		b.mu.Unlock()
		b.restart()
		if err != nil {
			return "", fmt.Errorf("%w: %v", ErrModelCrashed, err)
		}
		return "", ErrModelCrashed
	}
	b.status = StatusReady
	b.mu.Unlock()

	if err != nil {
		// Timeout or cancellation: the next query retries normally (per-NPC
		// timeout orchestration is Story 004).
		return "", err
	}
	if resp.Error != "" {
		return "", fmt.Errorf("aidialogue: subprocess error: %s", resp.Error)
	}
	return resp.Text, nil
}

// monitor waits for process death and applies the restart policy (Story
// 001: spawn/health-check/restart lifecycle). Restarts are only attempted
// for crashes while Ready/Generating (mid-session, AC-20); a death during
// warm-up is a load failure reported by the warm-up goroutine, and deaths
// during shutdown are ignored.
func (b *Backend) monitor(proc *process) {
	defer b.wg.Done()
	<-proc.deadCh()

	b.mu.Lock()
	shutdown := b.shutdown
	wasReady := b.status == StatusReady || b.status == StatusGenerating
	procIsCurrent := b.proc == proc
	b.mu.Unlock()
	if shutdown || !wasReady || !procIsCurrent {
		return
	}

	b.logger.Warn("llm subprocess died",
		"exit", proc.exitError(),
		"stderr_tail", proc.stderr.String())
	b.restart()
}

// restart applies the restart policy (Story 001: max 1 restart per session,
// 5-min cooldown) and spawns a replacement subprocess in the background.
// On success the backend returns to StatusReady and skips the re-probe — the
// startup probe is the session gate — so the next query uses the LLM
// (AC-20). On refusal or spawn failure it lands in session-stable
// StatusDegraded with a one-time notification.
func (b *Backend) restart() {
	b.mu.Lock()
	if b.shutdown || b.restarting {
		b.mu.Unlock()
		return
	}
	b.restarting = true
	b.mu.Unlock()
	defer func() {
		b.mu.Lock()
		b.restarting = false
		b.mu.Unlock()
	}()

	now := time.Now()
	b.mu.Lock()
	allowed := b.policy.allow(now)
	b.mu.Unlock()
	if !allowed {
		b.setStatus(StatusDegraded)
		b.notifyOnce(NotifRestartFailed, textModeMessage)
		b.logger.Warn("llm subprocess restart refused (restart budget or cooldown); text mode for session")
		return
	}

	proc, err := spawnProcess(b.cfg, b.modelPath)
	if err != nil {
		b.setStatus(StatusDegraded)
		b.notifyOnce(NotifRestartFailed, textModeMessage)
		b.logger.Warn("llm subprocess restart failed", "error", err)
		return
	}
	c := newConn(proc.stdin, proc.stdout)

	b.mu.Lock()
	if b.shutdown {
		// Shutdown began after spawnProcess completed — kill the fresh proc
		// so it is not reparented to init with the model resident.
		b.mu.Unlock()
		_ = c.quit()
		_ = c.closeStdin()
		_ = proc.terminateGracefully(b.cfg.TermGrace)
		return
	}
	b.policy.record(time.Now())
	b.proc = proc
	b.ipc = c
	b.status = StatusReady
	b.mu.Unlock()

	b.wg.Add(1)
	go b.monitor(proc)
	b.logger.Info("llm subprocess restarted")
}

// runWarmup executes the background warm-up (CR-7): capability probe, then
// greeting pre-fill. Runs on the lifecycle context so Shutdown aborts it.
func (b *Backend) runWarmup(c *conn) {
	defer b.wg.Done()
	result, err := warmup(b.lifecycle, b.progressCh, c, b.cfg)

	if b.lifecycle.Err() != nil {
		return // shutdown aborted warm-up; status left as-is
	}
	if err != nil && !result.Passed {
		b.setStatus(StatusDegraded)
		b.notifyOnce(NotifWarmupFailed, textModeMessage)
		b.logger.Warn("llm warm-up transport failure; text mode for session", "error", err)
		return
	}
	if !result.Passed {
		b.setStatus(StatusDegraded)
		b.notifyOnce(NotifProbeFailed, textModeMessage)
		b.logger.Warn("llm capability probe failed; text mode for session", "reason", result.FailReason)
		return
	}
	if err != nil {
		// Probe passed; the greeting pre-fill failed — warn-only (the probe
		// is the gate; pre-fill is a performance optimization).
		b.logger.Warn("llm greeting pre-fill failed", "error", err)
	}
	b.setStatus(StatusReady)
	b.logger.Info("llm backend ready",
		"decode_tok_s", result.DecodeTokPerSec,
		"prefill_tok_s", result.PrefillTokPerSec,
		"cold_ms", result.ColdStartMs,
		"cached_ms", result.CachedMs)
}

// Shutdown implements LLMBackend: sends the quit command, waits up to
// Config.QuitGrace for a clean exit, then interrupts (SIGTERM on unix), then
// SIGKILL after Config.TermGrace. Returns an error only if the process had
// to be killed. Idempotent.
func (b *Backend) Shutdown() error {
	b.mu.Lock()
	if b.shutdown {
		b.mu.Unlock()
		return nil
	}
	b.shutdown = true
	proc := b.proc
	c := b.ipc
	b.mu.Unlock()

	b.cancel() // abort warm-up

	if proc == nil {
		return nil // never spawned (unloaded/degraded at start)
	}

	if c != nil {
		_ = c.quit()
		_ = c.closeStdin()
	}
	select {
	case <-proc.deadCh():
		return nil // exited cleanly (or already dead) — nothing more to do
	case <-time.After(b.cfg.QuitGrace):
	}
	return proc.terminateGracefully(b.cfg.TermGrace)
}

// setStatus stores the status under the mutex.
func (b *Backend) setStatus(s ModelStatus) {
	b.mu.Lock()
	b.status = s
	b.mu.Unlock()
}

// notifyOnce emits a notification at most once per session (GDD CR-5: the
// one-time notice; the mode does not flap).
func (b *Backend) notifyOnce(id, message string) {
	b.mu.Lock()
	if b.notified[id] {
		b.mu.Unlock()
		return
	}
	b.notified[id] = true
	b.mu.Unlock()
	select {
	case b.notifCh <- Notification{ID: id, Message: message}:
	default:
	}
}
