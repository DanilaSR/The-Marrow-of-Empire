// AP meter model (Story 002, AC-1/AC-5): the engine-free state behind the
// bottom-left Action Point bar. The meter owns the animated fill fraction
// and the numeric readout; the hudRenderer (action_render.go) translates
// them into document operations per frame.
//
// Timing per the story implementation notes: refill on turn start is
// 200ms (AC-5, with a brief amber pulse at completion), depletion on AP
// spend is 150ms (AC-1). Kaiju CSS has no transitions, so both are
// Go-driven tweens along the paper-slide ease-out curve (tween.go).
//
// The combat system emits no AP change event yet — AP spending lands with
// the targeting/attack stories — so the HUD wires only the turn-start
// refill today; SetAP (the depletion path) is rendered and tested now and
// the future EventAPChanged wiring will route into it.
package combat_hud

import (
	"fmt"
	"math"
)

// AP animation timings per the Story 002 implementation notes.
const (
	// apRefillDuration animates the fill from its current level to full on
	// turn start (AC-5: 200ms).
	apRefillDuration = 0.2 // seconds
	// apDepleteDuration animates the fill down when AP is spent (AC-1:
	// 150ms — the player sees the drain, not a teleport).
	apDepleteDuration = 0.15 // seconds
	// apPulseDuration is the amber brightness spike that follows a refill
	// completion (AC-5: brief, 150ms).
	apPulseDuration = 0.15 // seconds
)

// APMeter holds the active combatant's Action Points for display: the
// settled values, the animated fill fraction [0,1], and the refill pulse.
// It is not safe for concurrent use (same contract as CombatHUD).
type APMeter struct {
	current int
	max     int

	// fill is the animated fill fraction driving the bar width; it lags
	// behind current during transitions and equals current/max at rest.
	fill float64
	// tween advances fill toward its target (refill or depletion).
	tween *Tween
	// pulse drives the post-refill amber glow: strength = sin(pi * value),
	// so the glow brightens then dims over apPulseDuration.
	pulse *Tween

	reducedMotion bool
}

// NewAPMeter creates an empty meter (0 / 0, no fill).
func NewAPMeter() *APMeter { return &APMeter{} }

// OnTurnStarted applies a turn change (AC-5): the combat contract resets
// the pool to the combatant's full budget before TurnStarted fires, so
// current is the fresh turn budget and max the pool ceiling. The fill
// animates from its previous level (the last turn's remaining AP) to full
// over apRefillDuration, and the amber pulse fires when the fill
// completes. A meter already full skips both animations.
func (m *APMeter) OnTurnStarted(current, max int) {
	if current < 0 {
		current = 0
	}
	if max < 0 {
		max = 0
	}
	if current > max {
		current = max
	}
	m.current = current
	m.max = max
	m.pulse = nil
	target := 1.0
	if max == 0 {
		target = 0
	}
	if m.reducedMotion || m.fill >= target {
		// Reduced motion: jump straight to full. Already full: nothing to
		// animate, and no pulse for a bar that never moved.
		m.fill = target
		m.tween = nil
		return
	}
	m.tween = NewPaperSlideTween(m.fill, target, apRefillDuration)
}

// SetAP applies an AP spend (AC-1): the fill depletes to current/max over
// apDepleteDuration and the numeric readout counts down. Clamps to
// [0, max]. This is the hook the future EventAPChanged wiring calls; no
// live caller exists yet (see the package comment).
func (m *APMeter) SetAP(current int) {
	if current < 0 {
		current = 0
	}
	if current > m.max {
		current = m.max
	}
	m.current = current
	target := m.fraction(current)
	if m.reducedMotion || m.fill == target {
		m.fill = target
		m.tween = nil
		return
	}
	m.tween = NewPaperSlideTween(m.fill, target, apDepleteDuration)
}

// Update advances the fill and pulse tweens by dt seconds. Completed
// tweens stay at their final value; the pulse fires once when a refill
// to full completes (depletion never pulses — AC-1: the amber glow is the
// post-refill reward, not a drain effect).
func (m *APMeter) Update(dt float64) {
	if m.tween != nil {
		m.tween.Update(dt)
		m.fill = m.tween.Value()
		if m.tween.Done() {
			refill := m.tween.To >= 1.0 // refill targets the full bar; depletion never does
			m.tween = nil
			if !m.reducedMotion && m.pulse == nil && refill {
				m.pulse = NewPaperSlideTween(0, 1, apPulseDuration)
			}
		}
	}
	if m.pulse != nil {
		m.pulse.Update(dt)
		if m.pulse.Done() {
			m.pulse = nil
		}
	}
}

// Reset clears the meter (combat start and combat end): 0 / 0, no fill,
// no animations.
func (m *APMeter) Reset() {
	m.current, m.max = 0, 0
	m.fill = 0
	m.tween, m.pulse = nil, nil
}

// SetReducedMotion enables or disables the reduced-motion gate: while on,
// fill transitions complete instantly and the pulse is skipped
// (accessibility: "Combat HUD slide-in -> 0ms").
func (m *APMeter) SetReducedMotion(on bool) {
	m.reducedMotion = on
	if on {
		m.pulse = nil
	}
}

// Current returns the settled AP value (the target of any in-flight
// animation; the readout shows the animated value during transitions).
func (m *APMeter) Current() int { return m.current }

// Max returns the per-turn AP budget.
func (m *APMeter) Max() int { return m.max }

// FillFraction returns the animated fill fraction in [0,1] — the bar
// width as a percentage.
func (m *APMeter) FillFraction() float64 { return m.fill }

// PulseStrength returns the current amber pulse intensity in [0,1], 0
// outside the post-refill spike window.
func (m *APMeter) PulseStrength() float64 {
	if m.pulse == nil {
		return 0
	}
	return math.Sin(math.Pi * m.pulse.Value())
}

// DisplayText returns the numeric readout, "current / max" in combat face
// with tabular figures (prevents layout shift as digits change). The
// current value is the animated one, so the readout counts up on refill
// and down on depletion (AC-1, AC-5).
func (m *APMeter) DisplayText() string {
	return fmt.Sprintf("%d / %d", m.displayedCurrent(), m.max)
}

// displayedCurrent derives the animated readout value from the fill
// fraction, rounded to a whole AP point.
func (m *APMeter) displayedCurrent() int {
	return int(math.Round(m.fill * float64(m.max)))
}

// fraction converts an AP value to its fill fraction of the budget.
func (m *APMeter) fraction(v int) float64 {
	if m.max <= 0 {
		return 0
	}
	return float64(v) / float64(m.max)
}
