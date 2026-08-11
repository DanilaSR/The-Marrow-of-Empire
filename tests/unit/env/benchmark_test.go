// Performance benchmarks for Story ENV-001: Game Clock & Light States.
//
// Evidence path: production/epics/living-environment/
// story-001-game-clock-light-states.md
//
// Each benchmark maps to one ADR-0007 performance guardrail (control
// manifest Core layer): clock update < 0.01ms/frame and light state
// interpolation < 0.1ms during transitions.
//
// Run: go test -bench=. -benchtime=1000x ./tests/unit/env/
//
// TestPerformanceTargets runs the benchmarks internally and asserts against
// the ADR-0007 budgets. Run with -count=1 for stable single-run results;
// CI may set relaxed budgets.
package env_test

import (
	"sort"
	"testing"
	"time"

	"marrow-of-empire/env"
)

// percentile returns the p-th percentile (0-100) from sorted float64s.
func percentile(sorted []float64, p int) float64 {
	if len(sorted) == 0 {
		return 0
	}
	idx := (p * len(sorted)) / 100
	if idx >= len(sorted) {
		idx = len(sorted) - 1
	}
	return sorted[idx]
}

// ---------------------------------------------------------------------------
// Guardrail 1: Clock update < 0.01ms/frame — ADR-0007
// ---------------------------------------------------------------------------

// BenchmarkGameClockUpdate times one frame's worth of clock advance at the
// 60fps frame budget: a multiply, add, and modulo — no allocations.
func BenchmarkGameClockUpdate(b *testing.B) {
	c := env.NewGameClock()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c.Update(1.0 / 60.0) // one 60fps frame in real seconds
	}
}

func TestGameClockUpdatePerformance(t *testing.T) {
	c := env.NewGameClock()

	// Warm up.
	c.Update(1.0 / 60.0)

	const n = 100_000
	times := make([]float64, 0, n)
	for i := 0; i < n; i++ {
		start := time.Now()
		c.Update(1.0 / 60.0)
		times = append(times, float64(time.Since(start)))
	}

	sort.Float64s(times)
	p95 := percentile(times, 95)

	// ADR-0007 guardrail: clock update < 0.01ms = 10,000ns
	const budgetNs = 10_000
	if p95 > budgetNs {
		t.Errorf("FAIL: GameClock.Update p95 %.0fns exceeds budget %dns (%.3fms > 0.01ms)",
			p95, budgetNs, p95/1_000_000)
	} else {
		t.Logf("PASS: GameClock.Update p95 %.0fns — budget 0.01ms", p95)
	}
}

// ---------------------------------------------------------------------------
// Guardrail 2: Light state interpolation < 0.1ms during transitions —
// ADR-0007
// ---------------------------------------------------------------------------

// BenchmarkAmbientAt times ambient light sampling at the boundary hours
// where interpolation is active (the worst case: a full pass over the
// boundary table plus the smooth-step blend).
func BenchmarkAmbientAt(b *testing.B) {
	// Boundary hours where the blend window is active: 4:00, 7:00, 17:00,
	// 20:00 (transition midpoints; see env.TransitionMinutes).
	boundaries := []float64{240, 420, 1020, 1200}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = env.AmbientAt(boundaries[i%len(boundaries)])
	}
}

func TestLightInterpolationPerformance(t *testing.T) {
	samples := []float64{240, 420, 1020, 1200} // mid-transition samples

	// Warm up.
	for _, m := range samples {
		env.AmbientAt(m)
	}

	const n = 100_000
	times := make([]float64, 0, n)
	for i := 0; i < n; i++ {
		start := time.Now()
		_ = env.AmbientAt(samples[i%len(samples)])
		times = append(times, float64(time.Since(start)))
	}

	sort.Float64s(times)
	p95 := percentile(times, 95)

	// ADR-0007 guardrail: light state interpolation < 0.1ms = 100,000ns
	const budgetNs = 100_000
	if p95 > budgetNs {
		t.Errorf("FAIL: AmbientAt p95 %.0fns exceeds budget %dns (%.3fms > 0.1ms)",
			p95, budgetNs, p95/1_000_000)
	} else {
		t.Logf("PASS: AmbientAt p95 %.0fns — budget 0.1ms", p95)
	}
}

// ---------------------------------------------------------------------------
// Aggregated performance gate — run via go test -run TestPerformanceTargets
// ---------------------------------------------------------------------------

func TestPerformanceTargets(t *testing.T) {
	t.Run("GameClockUpdate", TestGameClockUpdatePerformance)
	t.Run("LightInterpolation", TestLightInterpolationPerformance)
}
