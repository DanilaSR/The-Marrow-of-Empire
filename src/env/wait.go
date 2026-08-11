// Wait / pass-time mechanic per ADR-0007 (Wait / Pass Time) and TR-env-003,
// Story ENV-002: the player at a rest point fast-forwards game time to a
// target hour behind a fade-to-black transition. Rest-point heal rules
// (inn: full, camp: 50%) are health-system concerns (ENV-002 AC-3/AC-4)
// and out of scope here; this file implements the time fast-forward itself
// (AC-2) and the combat/conversation block (AC-5).
//
// The transition is a four-phase timeline driven by Update's real dt:
// FadingOut (500ms, time frozen) → FadedOut (time advances at 100×) →
// FadingIn (500ms, time already at target) → NoFade (complete). The clock
// stays a pure data structure as in ENV-001: FadePhase/FadeProgress are
// presentation data consumed by a future render layer — the clock never
// renders.
//
// Story: ENV-002 (production/epics/living-environment/
// story-002-wait-pass-time.md).
package env

import (
	"fmt"
	"math"
)

// Wait tuning constants per ADR-0007 wait parameters (ENV-002
// Implementation Notes: "Wait speed 100× ... Fade to black 500ms, time
// advances, fade in 500ms").
const (
	// FastForwardGameMinutesPerRealSecond is the rate game time advances
	// during a wait's FadedOut phase: 100 game minutes per real second
	// (ENV-002 AC-2: 12 game hours = 720 game minutes in 7.2 real
	// seconds).
	//
	// NOTE: ADR-0007 and the ENV-002 notes also state "Wait speed 100×
	// (1 game hour = 36 real seconds)" — 60× slower than this constant —
	// while the ADR's performance section claims a ~500ms full-day skip,
	// which contradicts both. The explicit 7.2-second expectation of
	// ENV-002 AC-2 is implemented here; this constant is the single
	// retune point if the lead confirms the ADR rate instead.
	FastForwardGameMinutesPerRealSecond = 100.0

	// FastForwardFadeSeconds is the duration of the fade-to-black at a
	// wait's start and the fade-in at its end (ENV-002 Implementation
	// Notes: 500ms each).
	FastForwardFadeSeconds = 0.5

	// fastForwardSnapEpsilon is the float tolerance (game minutes) under
	// which the wait counts as complete and snaps to the target hour —
	// arrival always lands exactly on targetHour:00 (ENV-002 AC-2).
	fastForwardSnapEpsilon = 1e-9
)

// FadePhase identifies the current phase of a wait fast-forward transition
// (ENV-002 AC-2: fade to black, time advances, fade in).
type FadePhase int

// Fade phases in timeline order.
const (
	// NoFade: no wait transition in progress; FadeProgress is 0.
	NoFade FadePhase = iota
	// FadingOut: screen fading to black (first 500ms); game time is
	// frozen.
	FadingOut
	// FadedOut: screen fully black; game time advances at
	// FastForwardGameMinutesPerRealSecond toward the target hour.
	FadedOut
	// FadingIn: target hour reached; screen fading back in (last 500ms).
	FadingIn
)

// String returns the phase name for logging and debugging.
func (p FadePhase) String() string {
	switch p {
	case NoFade:
		return "NoFade"
	case FadingOut:
		return "FadingOut"
	case FadedOut:
		return "FadedOut"
	case FadingIn:
		return "FadingIn"
	default:
		return fmt.Sprintf("FadePhase(%d)", int(p))
	}
}

// FastForward starts a wait fast-forward to targetHour (0-23) per ENV-002:
// the clock enters FadingOut, then advances at 100× behind black, then
// fades in, landing exactly on targetHour:00. It returns the game minutes
// that will elapse during the wait, rounded to the nearest minute, or 0
// when the wait cannot start: the clock is paused (combat/conversation —
// ENV-002 AC-5 "wait is blocked"), a wait is already in progress,
// targetHour is outside 0-23, or the current time is already exactly at
// that hour (wait presets are Dawn/Noon/Dusk/Midnight, AC-1; a
// zero-length or full-day wait is not supported).
//
// A target earlier than the current hour wraps past midnight ("Wait until
// Dawn" at 22:00 targets 6:00 the next day, AC-2). Completion snaps the
// clock to the target hour — time never overshoots into the next hour —
// so the light state lands exactly on the waited-for state (AC-2: "the
// light state reaches Dawn (6:00 AM)").
func (c *GameClock) FastForward(targetHour int) int {
	if c.paused || c.fastForwarding || targetHour < 0 || targetHour > 23 {
		return 0
	}
	elapsed := c.minutesUntilHour(targetHour)
	if elapsed == 0 {
		return 0
	}
	c.fastForwarding = true
	c.targetHour = targetHour
	c.remainingMinutes = elapsed
	c.fadePhase = FadingOut
	c.fadeProgress = 0
	return int(math.Round(elapsed))
}

// IsFastForwarding reports whether a wait fast-forward transition is in
// progress — true from FastForward until the fade-in completes.
func (c *GameClock) IsFastForwarding() bool {
	return c.fastForwarding
}

// FadePhase returns the current wait transition phase (NoFade when idle).
func (c *GameClock) FadePhase() FadePhase {
	return c.fadePhase
}

// FadeProgress returns the screen blackness of the current fade in [0, 1]
// — 0 clear, 1 fully black — eased with the same smooth-step curve as the
// light-state transitions (light.go): FadingOut goes 0 → 1, FadedOut
// holds 1, FadingIn goes 1 → 0, NoFade is 0. A presentation layer draws
// a black overlay at this alpha and advances nothing itself; the clock
// never renders (ENV-001 principle).
func (c *GameClock) FadeProgress() float64 {
	switch c.fadePhase {
	case FadingOut:
		return smoothStep(c.fadeProgress)
	case FadedOut:
		return 1
	case FadingIn:
		return 1 - smoothStep(c.fadeProgress)
	default: // NoFade
		return 0
	}
}

// minutesUntilHour returns the game minutes from the current time to the
// next occurrence of hour (0-23), wrapping past midnight when the target
// is earlier than the current time. Returns 0 when the current time is
// already exactly at that hour.
func (c *GameClock) minutesUntilHour(hour int) float64 {
	target := float64(hour * 60)
	if target == c.timeMinutes {
		return 0
	}
	if target > c.timeMinutes {
		return target - c.timeMinutes
	}
	return MinutesPerDay - c.timeMinutes + target
}

// updateFastForward advances the wait transition by dt real seconds,
// stepping fade-out → advance → fade-in and carrying leftover dt across
// phase boundaries. Game time advances only in FadedOut and only up to
// the target hour, which is hit exactly (snap within float tolerance;
// never overshoots). dt beyond the transition's end is discarded — a
// wait never spills into normal-rate time.
func (c *GameClock) updateFastForward(dt float64) {
	remaining := dt
	for remaining > 0 {
		switch c.fadePhase {
		case FadingOut:
			need := (1 - c.fadeProgress) * FastForwardFadeSeconds
			if remaining < need {
				c.fadeProgress += remaining / FastForwardFadeSeconds
				return
			}
			remaining -= need
			c.fadePhase = FadedOut
			c.fadeProgress = 1
		case FadedOut:
			advance := remaining * FastForwardGameMinutesPerRealSecond
			if advance < c.remainingMinutes-fastForwardSnapEpsilon {
				c.timeMinutes += advance
				c.remainingMinutes -= advance
				return
			}
			remaining -= c.remainingMinutes / FastForwardGameMinutesPerRealSecond
			c.timeMinutes = float64(c.targetHour * 60)
			c.remainingMinutes = 0
			c.fadePhase = FadingIn
			c.fadeProgress = 0
		case FadingIn:
			need := (1 - c.fadeProgress) * FastForwardFadeSeconds
			if remaining < need {
				c.fadeProgress += remaining / FastForwardFadeSeconds
				return
			}
			c.fastForwarding = false
			c.fadePhase = NoFade
			c.fadeProgress = 0
			return // leftover dt is discarded
		}
	}
}
