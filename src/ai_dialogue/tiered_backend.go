package aidialogue

import (
	"context"
	"fmt"
)

// TieredBackend composes two LLMBackend implementations behind the
// LLMBackend interface per ADR-0008 revised Decision (dual-model).
// Simple queries route to the fast (tiny CGo) backend; complex queries
// route to the full (IPC primary) backend. Transparent to callers —
// the AI Dialogue Engine sees one LLMBackend.
type TieredBackend struct {
	fast       LLMBackend // TinyCGoBackend (<500M params, in-process)
	full       LLMBackend // IPCLlamaBackend (3B params, separate process)
	classifier QueryClassifier
}

// NewTieredBackend creates a dual-model backend. Both backends must be
// independently initialized before use. If fast is nil, all queries
// route to full (invisible failover per AC-5). full must not be nil.
func NewTieredBackend(fast, full LLMBackend) *TieredBackend {
	return &TieredBackend{
		fast: fast,
		full: full,
	}
}

// Generate routes the prompt to the appropriate backend. Simple queries
// go to the tiny CGo model; complex queries go to the IPC primary.
// If the tiny model fails (error or degraded), the query silently falls
// through to the primary model (AC-4, AC-5).
func (tb *TieredBackend) Generate(ctx context.Context, prompt string) (string, error) {
	// Try fast backend for simple queries.
	if tb.shouldTryFast(prompt) {
		resp, err := tb.fast.Generate(ctx, prompt)
		if err == nil {
			return resp, nil
		}
		// Fast backend failed — fall through to full silently.
		// This handles: CGo panic (AC-4), model not loaded (AC-5),
		// and transient inference errors.
	}

	return tb.full.Generate(ctx, prompt)
}

// Status returns the best available tier status. Ready if either
// backend is ready; Degraded only if both are degraded; Loading otherwise.
func (tb *TieredBackend) Status() ModelStatus {
	fullStatus := tb.full.Status()
	fastStatus := ModelStatus(StatusUnloaded)
	if tb.fast != nil {
		fastStatus = tb.fast.Status()
	}

	// Ready if either tier is operational.
	if fullStatus == StatusReady || fastStatus == StatusReady {
		return StatusReady
	}

	// Generating if either tier is actively generating.
	if fullStatus == StatusGenerating || fastStatus == StatusGenerating {
		return StatusGenerating
	}

	// Degraded only when both tiers are degraded (AC-5: invisible failover
	// means primary alone is still Ready, not Degraded).
	if fullStatus == StatusDegraded && fastStatus == StatusDegraded {
		return StatusDegraded
	}

	// At least one tier is loading or unloaded.
	return StatusLoading
}

// Shutdown stops both backends. Errors from the fast backend are logged
// but do not prevent the full backend from shutting down.
func (tb *TieredBackend) Shutdown() error {
	var errs []error
	if tb.fast != nil {
		if err := tb.fast.Shutdown(); err != nil {
			errs = append(errs, fmt.Errorf("fast backend shutdown: %w", err))
		}
	}
	if err := tb.full.Shutdown(); err != nil {
		errs = append(errs, fmt.Errorf("full backend shutdown: %w", err))
	}
	if len(errs) > 0 {
		return fmt.Errorf("tiered backend shutdown: %v", errs)
	}
	return nil
}

// shouldTryFast reports whether the prompt is a candidate for the tiny
// model. The fast backend must be ready AND the query must be simple.
func (tb *TieredBackend) shouldTryFast(prompt string) bool {
	if tb.fast == nil {
		return false
	}
	if tb.fast.Status() != StatusReady {
		return false
	}
	// Classify without topic-matcher data — the classifier defaults to
	// complex (route to primary) when uncertain, which is the safe default.
	return tb.classifier.IsSimple(prompt, ClassifyResult{})
}

// ShouldTryFast is the public equivalent that accepts topic-matcher
// metadata from the Question System (Story Q-001). Callers that have
// MatchResult data should use this for more accurate routing.
func (tb *TieredBackend) ShouldTryFast(prompt string, cr ClassifyResult) bool {
	if tb.fast == nil {
		return false
	}
	if tb.fast.Status() != StatusReady {
		return false
	}
	return tb.classifier.IsSimple(prompt, cr)
}
