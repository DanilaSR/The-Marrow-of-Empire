// Capability probe (GDD CR-1, Formulas § Latency Budget; Story 001
// Implementation Notes): a ~150-token benchmark prompt run at startup that
// measures prefill and decode speed and gates LLM mode on three conditions.
package aidialogue

import (
	"context"
	"fmt"
	"time"
)

// ProbeResult carries the measured speeds, the modeled totals, and the gate
// verdict.
type ProbeResult struct {
	// DecodeTokPerSec is the measured decode speed (tokens/second).
	DecodeTokPerSec float64
	// PrefillTokPerSec is the measured prefill speed (tokens/second).
	PrefillTokPerSec float64
	// ColdStartMs is the modeled cold-start total for a max-length response
	// (prefill of the probe prompt + decode of ProbeResponseTokens).
	ColdStartMs int64
	// CachedMs is the modeled cached total: decode of ProbeResponseTokens
	// with the prompt prefix cached.
	CachedMs int64
	// Passed reports whether all three gates passed.
	Passed bool
	// FailReason names the first gate that failed (empty when Passed).
	FailReason string
}

// runProbe executes the two-request benchmark over conn:
//
//  1. the probe prompt with max_tokens=1 — dominated by prefill time,
//  2. the same prompt with max_tokens=ProbeDecodeTokens — prefill (possibly
//     prompt-cached) plus a short decode run.
//
// The decode rate is derived from the duration difference, which cancels the
// prefill component whether or not the backend caches the prompt; the
// prefill rate comes from the 1-token run. Durations are wall-clock measured
// on the game side, independent of the backend's self-reported DurationMs.
//
// The gate (all three, per Story 001): decode ≥ MinDecodeTokPerSec,
// cold-start total ≤ MaxColdStartMs, cached total ≤ MaxCachedMs. If the
// probe fails, the engine operates in text-fallback mode for the session
// (AC-16).
func runProbe(ctx context.Context, c *conn, cfg Config) (ProbeResult, error) {
	res := ProbeResult{}

	start := time.Now()
	_, err := c.request(ctx, &PromptRequest{Prompt: cfg.ProbePrompt, MaxTokens: 1, Temp: cfg.Temperature})
	if err != nil {
		return res, fmt.Errorf("aidialogue: probe run 1: %w", err)
	}
	run1 := time.Since(start)

	start = time.Now()
	_, err = c.request(ctx, &PromptRequest{Prompt: cfg.ProbePrompt, MaxTokens: cfg.ProbeDecodeTokens, Temp: cfg.Temperature})
	if err != nil {
		return res, fmt.Errorf("aidialogue: probe run 2: %w", err)
	}
	run2 := time.Since(start)

	decodeDelta := run2 - run1
	if decodeDelta <= 0 {
		return res, fmt.Errorf("aidialogue: probe decode delta %s is not positive", decodeDelta)
	}
	decodedTokens := float64(cfg.ProbeDecodeTokens - 1)
	res.DecodeTokPerSec = decodedTokens / decodeDelta.Seconds()

	prefillSec := run1.Seconds() - 1/res.DecodeTokPerSec
	if prefillSec <= 0 {
		res.PrefillTokPerSec = 0
		res.ColdStartMs = -1
		res.FailReason = "prefill time unmeasurable"
		return res, nil
	}
	res.PrefillTokPerSec = float64(cfg.ProbePromptTokens) / prefillSec
	res.ColdStartMs = msFromSecs(float64(cfg.ProbePromptTokens)/res.PrefillTokPerSec + float64(cfg.ProbeResponseTokens)/res.DecodeTokPerSec)
	res.CachedMs = msFromSecs(float64(cfg.ProbeResponseTokens) / res.DecodeTokPerSec)

	switch {
	case res.DecodeTokPerSec < cfg.MinDecodeTokPerSec:
		res.FailReason = fmt.Sprintf("decode %.1f tok/s below minimum %.1f tok/s", res.DecodeTokPerSec, cfg.MinDecodeTokPerSec)
	case res.ColdStartMs > cfg.MaxColdStartMs:
		res.FailReason = fmt.Sprintf("cold-start total %dms above limit %dms", res.ColdStartMs, cfg.MaxColdStartMs)
	case res.CachedMs > cfg.MaxCachedMs:
		res.FailReason = fmt.Sprintf("cached total %dms above limit %dms", res.CachedMs, cfg.MaxCachedMs)
	default:
		res.Passed = true
	}
	return res, nil
}

func msFromSecs(s float64) int64 { return int64(s * 1000) }
