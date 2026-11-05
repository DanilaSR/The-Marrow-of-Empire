// Integration tests for Story 001: LLM Backend Interface & IPC Subprocess.
//
// Evidence path: production/epics/ai-dialogue-engine/story-001-llm-backend-ipc.md
// Required evidence: this file must exist and pass (Story Type: Integration).
//
// Acceptance criteria covered (1:1 function mapping):
//
//	AC-15  test_model_file_missing_enters_degraded_mode  -> TestModelFileMissingEntersDegradedMode
//	AC-16  test_probe_below_10toks_enters_degraded_mode  -> TestProbeBelow10ToksEntersDegradedMode
//	AC-20  test_subprocess_crash_restarts_once           -> TestSubprocessCrashRestartsOnce
//
// The tests use a stub LLM binary (stub/main.go, built in TestMain) — no
// real model file or llama.cpp is required. The stub simulates prefill and
// decode latency (STUB_TOKEN_MS controls the token rate), deliberate crashes
// (a prompt containing "CRASH"), and clean exit on the "quit" command.
//
// Test evidence standards: deterministic (no real-time-dependent
// assertions), isolated (each test sets up and tears down its own state),
// no external APIs (the stub is local).
package aidialogue_test

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"marrow-of-empire/ai_dialogue"
)

// stubBin is the path of the compiled stub binary.
var stubBin string

// TestMain compiles the stub LLM binary once for the whole suite. go test
// runs with the working directory set to this package, so "./stub" resolves.
func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "ai-dialogue-stub-*")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(dir)
	stubBin = filepath.Join(dir, "llm-stub")
	build := exec.Command("go", "build", "-o", stubBin, "./stub")
	out, err := build.CombinedOutput()
	if err != nil {
		panic(fmt.Sprintf("build stub: %v\n%s", err, out))
	}
	os.Exit(m.Run())
}

// newTestBackend creates a backend configured for the stub binary and a
// model file in dir (any bytes — the stub ignores model content).
func newTestBackend(t *testing.T, dir string, mutate func(*aidialogue.Config)) *aidialogue.Backend {
	t.Helper()
	cfg := aidialogue.DefaultConfig()
	cfg.BinaryPath = stubBin
	cfg.ModelDirs = []string{dir}
	cfg.ModelName = "test-model.gguf"
	cfg.MaxTokens = 30 // keeps each stub response ~0.6s (token rate 20ms/token)
	cfg.RestartCooldown = 50 * time.Millisecond
	cfg.DefaultGenerateTimeout = 5 * time.Second
	if mutate != nil {
		mutate(&cfg)
	}
	return aidialogue.New(cfg, slog.New(slog.DiscardHandler))
}

// writeModelFile creates a dummy model file in dir.
func writeModelFile(t *testing.T, dir string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	path := filepath.Join(dir, "test-model.gguf")
	if err := os.WriteFile(path, []byte("dummy model bytes"), 0o644); err != nil {
		t.Fatalf("write model: %v", err)
	}
}

// waitStatus polls until backend.Status() == want or the deadline passes.
func waitStatus(t *testing.T, b *aidialogue.Backend, want aidialogue.ModelStatus) {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		if b.Status() == want {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("status %s, want %s", b.Status(), want)
}

// nextNotification reads one notification with the given ID within 5s.
func nextNotification(t *testing.T, b *aidialogue.Backend, id string) {
	t.Helper()
	deadline := time.After(5 * time.Second)
	for {
		select {
		case n := <-b.Notifications():
			if n.ID == id {
				return
			}
		case <-deadline:
			t.Fatalf("no notification %q within 5s", id)
		}
	}
}

// expectNoNotification asserts that no notification arrives within 200ms
// (verifies one-time emission).
func expectNoNotification(t *testing.T, b *aidialogue.Backend) {
	t.Helper()
	select {
	case n := <-b.Notifications():
		t.Fatalf("unexpected notification %q", n.ID)
	case <-time.After(200 * time.Millisecond):
	}
}

// TestModelFileMissingEntersDegradedMode covers AC-15: the model file is
// configured to a nonexistent path → StatusDegraded, a one-time notification
// is emitted, and subsequent queries return an error (the caller falls back
// to quote-styled fact text — rendering is Story 004's job).
func TestModelFileMissingEntersDegradedMode(t *testing.T) {
	dir := t.TempDir() // empty — no model file anywhere
	backend := newTestBackend(t, dir, nil)

	if err := backend.Start(context.Background()); err != nil {
		t.Fatalf("Start returned error: %v", err)
	}
	waitStatus(t, backend, aidialogue.StatusDegraded)
	nextNotification(t, backend, aidialogue.NotifModelMissing)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if _, err := backend.Generate(ctx, "Who tends the lanterns?"); !errors.Is(err, aidialogue.ErrNotReady) {
		t.Fatalf("Generate error = %v, want ErrNotReady", err)
	}

	// One-time notification: a second Start must not re-emit it.
	if err := backend.Start(context.Background()); err != nil {
		t.Fatalf("second Start returned error: %v", err)
	}
	expectNoNotification(t, backend)

	if err := backend.Shutdown(); err != nil {
		t.Fatalf("Shutdown: %v", err)
	}
}

// TestProbeBelow10ToksEntersDegradedMode covers AC-16: the capability probe
// returns ~6.7 tok/s (below the 10 tok/s minimum) → StatusDegraded and a
// one-time notification. STUB_TOKEN_MS=150 makes the stub decode at
// 7 / (7×0.15s) ≈ 6.7 tok/s — the measured rate is derived from the
// duration difference between the two probe runs, so the decode gate fails.
func TestProbeBelow10ToksEntersDegradedMode(t *testing.T) {
	dir := t.TempDir()
	writeModelFile(t, dir)
	t.Setenv("STUB_TOKEN_MS", "150") // ≈ 6.7 tok/s < 10

	backend := newTestBackend(t, dir, nil)
	if err := backend.Start(context.Background()); err != nil {
		t.Fatalf("Start returned error: %v", err)
	}
	waitStatus(t, backend, aidialogue.StatusDegraded)
	nextNotification(t, backend, aidialogue.NotifProbeFailed)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if _, err := backend.Generate(ctx, "Who tends the lanterns?"); !errors.Is(err, aidialogue.ErrNotReady) {
		t.Fatalf("Generate error = %v, want ErrNotReady", err)
	}
	expectNoNotification(t, backend) // one-time

	if err := backend.Shutdown(); err != nil {
		t.Fatalf("Shutdown: %v", err)
	}
}

// TestSubprocessCrashRestartsOnce covers AC-20: the model subprocess crashes
// during generation → the current response falls back to text (Generate
// returns ErrModelCrashed), the subprocess is restarted once, and if the
// restart succeeds the next query uses the LLM. A second crash is not
// restarted (max 1 restart per session) → session-stable StatusDegraded with
// a one-time notification.
func TestSubprocessCrashRestartsOnce(t *testing.T) {
	dir := t.TempDir()
	writeModelFile(t, dir)
	backend := newTestBackend(t, dir, nil)

	if err := backend.Start(context.Background()); err != nil {
		t.Fatalf("Start returned error: %v", err)
	}
	waitStatus(t, backend, aidialogue.StatusReady)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Normal generation works.
	text, err := backend.Generate(ctx, "Who tends the lanterns?")
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if text == "" {
		t.Fatal("empty response")
	}

	// Crash during generation → the current response falls back to text.
	if _, err := backend.Generate(ctx, "CRASH"); !errors.Is(err, aidialogue.ErrModelCrashed) {
		t.Fatalf("Generate after crash = %v, want ErrModelCrashed", err)
	}

	// Auto-restart once → Ready again.
	waitStatus(t, backend, aidialogue.StatusReady)

	// The next query uses the LLM again.
	text, err = backend.Generate(ctx, "What news from the coast?")
	if err != nil {
		t.Fatalf("Generate after restart: %v", err)
	}
	if text == "" {
		t.Fatal("empty response after restart")
	}

	// Second crash → restart refused (max 1 per session) → session Degraded.
	if _, err := backend.Generate(ctx, "CRASH"); !errors.Is(err, aidialogue.ErrModelCrashed) {
		t.Fatalf("Generate after second crash = %v, want ErrModelCrashed", err)
	}
	waitStatus(t, backend, aidialogue.StatusDegraded)
	nextNotification(t, backend, aidialogue.NotifRestartFailed)
	expectNoNotification(t, backend) // one-time

	if err := backend.Shutdown(); err != nil {
		t.Fatalf("Shutdown: %v", err)
	}
}

// TestGracefulShutdown covers the shutdown path (Story 001 Implementation
// Notes): the quit command reaches the subprocess, which exits cleanly;
// Shutdown is idempotent.
func TestGracefulShutdown(t *testing.T) {
	dir := t.TempDir()
	writeModelFile(t, dir)
	backend := newTestBackend(t, dir, nil)

	if err := backend.Start(context.Background()); err != nil {
		t.Fatalf("Start returned error: %v", err)
	}
	waitStatus(t, backend, aidialogue.StatusReady)

	if err := backend.Shutdown(); err != nil {
		t.Fatalf("Shutdown: %v", err)
	}
	if err := backend.Shutdown(); err != nil {
		t.Fatalf("second Shutdown (idempotency): %v", err)
	}
}
