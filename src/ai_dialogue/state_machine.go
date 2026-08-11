package aidialogue

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

// ConversationState is the per-NPC conversation state.
// Transitions: Unloaded → Loading → Ready ⇄ Generating → TextMode.
type ConversationState int

const (
	StateUnloaded ConversationState = iota
	StateLoading
	StateReady
	StateGenerating
	StateTextMode
)

// String returns a human-readable state name.
func (s ConversationState) String() string {
	switch s {
	case StateUnloaded:
		return "Unloaded"
	case StateLoading:
		return "Loading"
	case StateReady:
		return "Ready"
	case StateGenerating:
		return "Generating"
	case StateTextMode:
		return "TextMode"
	default:
		return "Unknown"
	}
}

const DefaultTimeoutThreshold = 25 * time.Second
const MaxTimeoutsBeforeStable = 3

// StateMachine manages the per-NPC conversation lifecycle: state transitions,
// degradation, timeout detection, and conversation memory. It wraps a shared
// LLMBackend and enforces the concurrency invariants from ADR-0008:
// one inference request at a time, queue depth ≤ 1, most recent question wins.
//
// All public methods are safe for concurrent use.
type StateMachine struct {
	state    ConversationState
	backend  LLMBackend
	npcID    uint64
	clock    Clock
	memory   *ConversationMemory
	fallback TextFallbackGenerator

	timeoutCounter  int
	sessionTextMode bool

	cancelFn        context.CancelFunc
	generationToken uint64 // incremented per Submit; stale completions become no-ops

	timeoutThreshold time.Duration

	mu sync.Mutex
}

type StateMachineOption func(*StateMachine)

func WithClock(c Clock) StateMachineOption {
	return func(sm *StateMachine) { sm.clock = c }
}

func WithTimeoutThreshold(d time.Duration) StateMachineOption {
	return func(sm *StateMachine) { sm.timeoutThreshold = d }
}

// NewStateMachine creates a per-NPC state machine wrapping the given backend.
func NewStateMachine(backend LLMBackend, npcID uint64, opts ...StateMachineOption) *StateMachine {
	sm := &StateMachine{
		backend:          backend,
		npcID:            npcID,
		clock:            RealClock{},
		memory:           &ConversationMemory{},
		timeoutThreshold: DefaultTimeoutThreshold,
	}

	for _, o := range opts {
		o(sm)
	}

	switch backend.Status() {
	case StatusUnloaded:
		sm.state = StateUnloaded
	case StatusLoading:
		sm.state = StateLoading
	case StatusDegraded, StatusError:
		sm.state = StateTextMode
		sm.sessionTextMode = true
	default:
		sm.state = StateReady
	}

	return sm
}

// ──────────────────────────────────────────────────────────────────────
// Public API
// ──────────────────────────────────────────────────────────────────────

// Submit sends a question to the LLM backend. If already generating, the
// current request is cancelled and the new question takes precedence
// (most recent wins per ADR-0008). The state machine owns the timeout
// deadline: context.WithTimeout is created here so timeout vs user-cancel
// are correctly classified.
func (sm *StateMachine) Submit(ctx context.Context, prompt string) (string, error) {
	sm.mu.Lock()

	if sm.sessionTextMode {
		sm.mu.Unlock()
		return sm.generateFallback(prompt), nil
	}

	// Cancel previous generation if still in flight (most recent wins).
	if sm.state == StateGenerating && sm.cancelFn != nil {
		sm.cancelFn()
		sm.cancelFn = nil
	}

	// Bump generation token so stale completions become no-ops.
	sm.generationToken++
	token := sm.generationToken

	// State machine owns the deadline — context.WithTimeout ensures
	// DeadlineExceeded is raised even if the backend doesn't self-timeout.
	genCtx, cancel := context.WithTimeout(ctx, sm.timeoutThreshold)
	sm.cancelFn = cancel
	sm.state = StateGenerating
	backend := sm.backend
	sm.mu.Unlock()

	// Blocking call outside the lock.
	response, err := backend.Generate(genCtx, prompt)
	cancel()

	sm.mu.Lock()
	defer sm.mu.Unlock()

	// ── Stale-generation guard ──
	// If another Submit arrived while we were generating, our token no
	// longer matches.  Return without touching shared state.
	if sm.generationToken != token {
		return "", fmt.Errorf("aidialogue: generation superseded for NPC %d", sm.npcID)
	}

	sm.cancelFn = nil

	// ── Classify the outcome ──
	switch {
	case err == nil:
		// Success.
		sm.state = StateReady
		sm.memory.RecordExchange(prompt, response)
		return response, nil

	case errors.Is(err, context.Canceled):
		// User-initiated cancel (Esc, new question, walk-away).
		sm.state = StateReady
		return "", fmt.Errorf("aidialogue: generation cancelled for NPC %d: %w", sm.npcID, err)

	case errors.Is(err, context.DeadlineExceeded):
		// Timeout — text fallback, increment counter.
		sm.timeoutCounter++
		if sm.timeoutCounter >= MaxTimeoutsBeforeStable {
			sm.sessionTextMode = true
		}
		sm.state = StateTextMode
		return sm.generateFallbackLocked(prompt), nil

	default:
		// Backend error (crash, not ready, etc.).
		if backend.Status() == StatusDegraded || backend.Status() == StatusError {
			sm.sessionTextMode = true
		}
		sm.state = StateTextMode
		return sm.generateFallbackLocked(prompt), nil
	}
}

// Cancel aborts the current generation (Esc, combat, walk-away, scene
// unload).  The response is discarded and the state returns to Ready.
// Safe to call when not generating (no-op).
func (sm *StateMachine) Cancel() {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if sm.state != StateGenerating {
		return
	}
	if sm.cancelFn != nil {
		sm.cancelFn()
		sm.cancelFn = nil
	}
	sm.state = StateReady
}

// Close ends the conversation: cancels any active generation, resets
// timeout counters, and clears conversation memory.
func (sm *StateMachine) Close() {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if sm.cancelFn != nil {
		sm.cancelFn()
		sm.cancelFn = nil
	}

	sm.timeoutCounter = 0
	if !sm.sessionTextMode {
		sm.state = StateReady
	}
	sm.memory = &ConversationMemory{}
}

// State returns the current conversation state.
func (sm *StateMachine) State() ConversationState {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	return sm.state
}

// IsTextMode reports whether text-fallback is active for this NPC.
func (sm *StateMachine) IsTextMode() bool {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	return sm.state == StateTextMode
}

// Memory returns a snapshot of the conversation memory for save serialization.
func (sm *StateMachine) Memory() ConversationMemory {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	return sm.memory.Snapshot()
}

// RestoreMemory replaces the conversation memory with a saved snapshot (AC-27).
func (sm *StateMachine) RestoreMemory(snapshot ConversationMemory) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.memory.Restore(snapshot)
}

// SetModelReady transitions from Loading → Ready when the backend reports
// it's ready.  Called by the warmup/initialization path (Story 001).
func (sm *StateMachine) SetModelReady() {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	if sm.state == StateLoading || sm.state == StateUnloaded {
		sm.state = StateReady
	}
}

// ──────────────────────────────────────────────────────────────────────
// Internal
// ──────────────────────────────────────────────────────────────────────

func (sm *StateMachine) generateFallback(prompt string) string {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	return sm.generateFallbackLocked(prompt)
}

// generateFallbackLocked MUST be called with sm.mu held.  It must not
// block (no I/O, no backend calls) — pure string formatting only.
func (sm *StateMachine) generateFallbackLocked(prompt string) string {
	_ = prompt
	return fmt.Sprintf("[Text Mode] NPC %d shares what they know.\n%s",
		sm.npcID, sm.memory.DisclosedSummary())
}
