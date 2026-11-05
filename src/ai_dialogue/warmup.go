// Background model warm-up (GDD CR-7, Story 001 Implementation Notes):
// after the subprocess spawns, the warm-up runs the capability probe and
// pre-fills a generic greeting prompt — this pages the ~2.1 GB of mmap'd
// weight tensors into memory so the first real query does not pay
// page-fault costs during prefill. Progress is reported on a channel for
// the main menu ("The old tongue wakes..."). Target: <12s NVMe desktop,
// <20s SATA SSD (AC-25, manual QA).
package aidialogue

import "context"

// WarmupProgress reports warm-up progress for the main menu UI. Steps
// (CR-7): 1 = model load (subprocess spawn), 2 = capability probe, 3 =
// greeting pre-fill.
type WarmupProgress struct {
	Step  int
	Total int
	Label string
	Pct   int
}

// warmupSteps mirrors CR-7's three steps.
const warmupSteps = 3

// warmup runs the capability probe and the greeting pre-fill over conn.
// The spawn itself is step 1 and is reported as the first progress event.
//
// Failure semantics: a probe failure or probe transport error returns the
// result with Passed=false and/or a non-nil error — the caller enters
// text-fallback mode. A greeting pre-fill error is returned with
// Passed=true: the probe is the gate, and pre-fill is a performance
// optimization, so the caller treats it as warn-only (documented deviation:
// CR-7 step 3 does not re-gate a machine already proven capable).
func warmup(ctx context.Context, progress chan<- WarmupProgress, c *conn, cfg Config) (ProbeResult, error) {
	emit := func(step int, label string, pct int) {
		if progress != nil {
			select {
			case progress <- WarmupProgress{Step: step, Total: warmupSteps, Label: label, Pct: pct}:
			case <-ctx.Done():
			default:
			}
		}
	}

	emit(1, "Loading the old tongue", 25)
	emit(2, "Measuring the tongue", 60)
	result, err := runProbe(ctx, c, cfg)
	if err != nil {
		return result, err
	}
	if !result.Passed {
		return result, nil
	}

	emit(3, "Waking the voice", 90)
	_, err = c.request(ctx, &PromptRequest{
		Prompt:    "Hello. I am awake. Speak.",
		MaxTokens: 8,
		Temp:      cfg.Temperature,
	})
	if err != nil {
		return result, err // probe passed — caller logs warn-only
	}
	emit(3, "The voice is ready", 100)
	return result, nil
}
