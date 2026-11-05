// Package aidialogue implements the AI Dialogue Engine's LLM inference
// backend (Story 001, TR-ai-001, TR-ai-005, ADR-0008): a swappable
// LLMBackend that drives a separate llama.cpp subprocess over
// newline-delimited JSON IPC on stdin/stdout.
//
// Control manifest rules honored here (Feature Layer, ADR-0008):
//   - LLM inference via separate llama.cpp process — stdin/stdout JSON IPC,
//     crash isolation
//   - LLMBackend interface: Generate(ctx, prompt) → (string, error) —
//     swappable
//   - Never in-process CGo LLM bindings for models >500M params (a 3B+
//     model crash must not take down the game); tiny models <500M may use
//     CGo with recover() guard per ADR-0008 revised 2026-08-11; never
//     HTTP for the local LLM
//   - Guardrails: model load ~15s during main menu (non-blocking), LLM
//     memory ~3-4GB in the separate process
//
// The package is pure Go — no Kaiju Engine APIs are required: inference is
// external to the engine (subprocess + pipes), so no kaijuengine.com imports
// exist here.
//
// Out of scope (neighbouring stories): prompt construction from KG data
// (Story 002), response post-processing (Story 003), per-NPC state machine
// and conversation memory (Story 004). Text-mode fallback rendering
// (quote-styled facts) belongs to Story 004/UI — this package only reports
// status and emits notification events that drive it.
package aidialogue

import (
	"context"
	"errors"
)

// LLMBackend is the interface for LLM inference (ADR-0008 Key Interfaces).
// Swappable — replace the process-based implementation with an in-process or
// cloud backend without changing dialogue logic.
type LLMBackend interface {
	// Generate sends a prompt and returns the generated text. The context
	// carries timeout/cancellation; a crashed or unavailable backend returns
	// an error so the caller (Story 004 state machine) falls back to text.
	Generate(ctx context.Context, prompt string) (string, error)

	// Status returns the current model status.
	Status() ModelStatus

	// Shutdown gracefully stops the LLM process (quit → SIGTERM → SIGKILL).
	Shutdown() error
}

// ModelStatus describes the state of the LLM subprocess (ADR-0008).
type ModelStatus int

const (
	// StatusUnloaded is the initial state: the model subprocess has not been
	// started.
	StatusUnloaded ModelStatus = iota
	// StatusLoading means the model is loading or warming up (~15s during the
	// main menu, non-blocking).
	StatusLoading
	// StatusReady means the model is loaded and ready for inference.
	StatusReady
	// StatusGenerating means inference is in progress.
	StatusGenerating
	// StatusDegraded means text-fallback mode is active for the session
	// (missing model file, failed capability probe, failed crash restart).
	// Session-stable — the mode does not flap (GDD CR-5).
	StatusDegraded
	// StatusError means an unrecoverable error (e.g., the backend binary
	// itself is missing).
	StatusError
)

// String returns a human-readable status name for logs and UI.
func (s ModelStatus) String() string {
	switch s {
	case StatusUnloaded:
		return "unloaded"
	case StatusLoading:
		return "loading"
	case StatusReady:
		return "ready"
	case StatusGenerating:
		return "generating"
	case StatusDegraded:
		return "degraded"
	case StatusError:
		return "error"
	default:
		return "unknown"
	}
}

// Sentinel errors shared across the package.
var (
	// ErrNotReady is returned by Generate when the backend is not in
	// StatusReady (loading, degraded, or error). The caller falls back to
	// text mode.
	ErrNotReady = errors.New("aidialogue: model not ready")
	// ErrBusy is returned by Generate when another request is already in
	// flight. The per-NPC state machine (Story 004) owns queueing — the
	// backend serves one request at a time.
	ErrBusy = errors.New("aidialogue: generation already in progress")
	// ErrModelCrashed is returned by Generate when the subprocess died during
	// generation. The current response falls back to text; the backend
	// auto-restarts once per session (GDD CR-5, AC-20).
	ErrModelCrashed = errors.New("aidialogue: model subprocess crashed")
	// ErrProcessExited is returned by the IPC layer when the subprocess
	// exited while a request was in flight.
	ErrProcessExited = errors.New("aidialogue: subprocess exited")
	// ErrModelNotFound is returned by FindModelFile when the model file does
	// not exist in any search directory (AC-15).
	ErrModelNotFound = errors.New("aidialogue: model file not found")
	// ErrChecksumMismatch is returned when the model file's SHA-256 does not
	// match Config.ExpectedSHA256 (Story 001: checksum validation before
	// loading).
	ErrChecksumMismatch = errors.New("aidialogue: model file checksum mismatch")
)
