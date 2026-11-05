package aidialogue

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
)

// ErrTinyModelPanic is returned when the tiny model's CGo inference panics.
// The TieredBackend catches this and falls through to the primary model.
var ErrTinyModelPanic = errors.New("aidialogue: tiny model CGo panic")

// MaxTinyPanicsPerSession is the number of panics tolerated before the
// fast path is disabled for the session (ADR-0008 revised: "repeated
// panics disable the fast path").
const MaxTinyPanicsPerSession = 3

// TinyCGoBackend implements LLMBackend for a small GGUF model (<500M
// params, Q4_K_M, ~300MB disk) loaded in-process via CGo llama.cpp
// bindings. It is the "fast" tier in the dual-model architecture.
//
// CGo calls are wrapped in deferred recover() — a panic during inference
// is caught, the model is marked degraded for this query, and the
// TieredBackend falls through to the primary model. The fast path is
// disabled for the session after MaxTinyPanicsPerSession panics.
// Note: ADR-0008 specifies reload-after-panic; the current stub
// implementation permanently degrades after one panic. Implement reload
// when real CGo bindings land (needs a Loader interface exposed to
// TieredBackend).
//
// The actual CGo inference calls are stubbed — see generateStub below.
// Replace with real llama.cpp CGo bindings when the dependency lands.
type TinyCGoBackend struct {
	mu     sync.Mutex
	status ModelStatus

	// modelPath is the filesystem path to the GGUF file.
	modelPath string

	// Panic tracking (per-session).
	panicCount       int
	fastPathDisabled bool // true when panic threshold exceeded

	// TODO: CGo — replace with real llama.cpp handles.
	// model unsafe.Pointer // llama.cpp model handle
	// ctx   unsafe.Pointer // llama.cpp context
}

// NewTinyCGoBackend creates a tiny-model backend. The model is not loaded
// until Load is called (typically during main-menu warm-up).
func NewTinyCGoBackend(modelPath string) *TinyCGoBackend {
	return &TinyCGoBackend{
		status:    StatusUnloaded,
		modelPath: modelPath,
	}
}

// Load attempts to load the tiny GGUF model via CGo. On failure the
// backend enters StatusDegraded — the TieredBackend routes all queries
// to the primary model (invisible failover per AC-5).
//
// TODO: CGo — replace with:
//
//	model := C.llama_load_model_from_file(C.CString(tb.modelPath), &params)
//	ctx := C.llama_new_context(model, &ctxParams)
func (tb *TinyCGoBackend) Load() error {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	tb.status = StatusLoading

	// TODO: CGo — verify file exists and is readable.
	// For now, modelPath presence is sufficient for the stub.
	if tb.modelPath == "" {
		tb.status = StatusDegraded
		return ErrModelNotFound
	}

	// Stub: mark ready. Real CGo loading would go here.
	tb.status = StatusReady
	slog.Debug("tiny model loaded (stub)", "path", tb.modelPath)
	return nil
}

// Generate implements LLMBackend. It wraps CGo inference in recover() per
// ADR-0008 CGo safety contract. Single-flight: mutex-guarded.
func (tb *TinyCGoBackend) Generate(ctx context.Context, prompt string) (string, error) {
	tb.mu.Lock()

	if tb.status != StatusReady {
		tb.mu.Unlock()
		if tb.status == StatusGenerating {
			return "", ErrBusy
		}
		return "", ErrNotReady
	}
	if tb.fastPathDisabled {
		tb.mu.Unlock()
		return "", ErrNotReady
	}

	tb.status = StatusGenerating
	tb.mu.Unlock()

	// CGo safety: recover from panics in the CGo call chain.
	var response string
	var err error
	func() {
		defer func() {
			if r := recover(); r != nil {
				tb.mu.Lock()
				tb.panicCount++
				tb.status = StatusDegraded
				if tb.panicCount >= MaxTinyPanicsPerSession {
					tb.fastPathDisabled = true
				}
				tb.mu.Unlock()
				err = fmt.Errorf("%w: %v", ErrTinyModelPanic, r)
				slog.Error("tiny model CGo panic", "panic", r, "count", tb.panicCount)
			}
		}()

		// TODO: CGo — replace with real llama.cpp inference:
		//   response = llama_generate(tb.ctx, tb.model, prompt)
		response, err = tb.generateStub(ctx, prompt)
	}()

	tb.mu.Lock()
	if tb.status == StatusGenerating {
		tb.status = StatusReady
	}
	tb.mu.Unlock()

	return response, err
}

// Status implements LLMBackend.
func (tb *TinyCGoBackend) Status() ModelStatus {
	tb.mu.Lock()
	defer tb.mu.Unlock()
	return tb.status
}

// Shutdown implements LLMBackend. Frees the llama.cpp model and context.
//
// TODO: CGo — replace with:
//
//	C.llama_free(tb.ctx)
//	C.llama_free_model(tb.model)
func (tb *TinyCGoBackend) Shutdown() error {
	tb.mu.Lock()
	defer tb.mu.Unlock()
	tb.status = StatusUnloaded
	return nil
}

// ──────────────────────────────────────────────────────────────────────
// Stub generation (REMOVE when CGo bindings land)
// ──────────────────────────────────────────────────────────────────────

// generateStub produces simple in-character deflections for the stub
// tiny model. Replace with real CGo llama.cpp inference.
func (tb *TinyCGoBackend) generateStub(ctx context.Context, prompt string) (string, error) {
	if ctx.Err() != nil {
		return "", ctx.Err()
	}
	_ = prompt
	return "I'm afraid I don't have much to share on that particular matter, but I welcome your questions.", nil
}
