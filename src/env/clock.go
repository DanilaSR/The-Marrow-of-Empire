// Package env implements the living environment core systems per ADR-0007
// (Game Clock & Environment Systems) and TR-env-001/002: the continuous
// 24-hour game clock at 10:1 compression and the five light states with
// smooth boundary interpolation.
//
// The package is pure Go math with no Kaiju engine dependency, so it can
// be unit-tested without the engine running (tests/unit/env). The clock
// is a pure data structure — it holds time and pause state only and never
// renders; a future presentation system consumes LightState and
// AmbientLight to drive sky, tint, and shadow color (ENV-001: rendering
// is out of scope).
//
// Story: ENV-001 (production/epics/living-environment/
// story-001-game-clock-light-states.md).
package env

import "math"

// Clock constants per ADR-0007 clock parameters and TR-env-001.
const (
	// MinutesPerDay is the number of game minutes in a 24-hour day.
	MinutesPerDay = 1440

	// StartMinutes is the new-game start time: 8:00 AM = 8 × 60
	// (ADR-0007: "Start time (new game) 8:00 AM (Day)").
	StartMinutes = 8 * 60
)

// gameMinutesPerRealSecond is the 10:1 compression ratio per ADR-0007:
// 1 real minute = 10 game minutes, so time advances at 10/60 game minutes
// per real second. Exactly 6 real seconds advance 1 game minute (ENV-001
// AC-2, Implementation Notes: game_minutes = real_seconds × 10 / 60).
const gameMinutesPerRealSecond = 10.0 / 60.0

// GameClock is the continuous 24-hour game clock per ADR-0007 and
// TR-env-001: 10:1 compression (2.4 real hours per full game day),
// starting at 8:00 AM (Day) on a new game, advancing only during
// exploration. Time pauses during conversation, menus, journal, and
// combat (ENV-001 AC-6; control manifest Core rule: "Never advance time
// during combat, conversation, menus, or journal").
//
// The clock is a pure data structure: it tracks game time and pause state
// only. The game loop calls Update with real elapsed seconds; consumers
// read time via TimeMinutes/Hour/Minute and light via CurrentState/
// Ambient. Pause is a plain boolean flag set by game systems via
// Pause/Resume (ENV-004 combat integration is a later story; if combat
// and conversation ever pause concurrently, the Pause/Resume API allows
// upgrading to a ref-counted flag without changing call sites).
//
// Story: ENV-001 (production/epics/living-environment/
// story-001-game-clock-light-states.md).
type GameClock struct {
	// timeMinutes is game minutes since midnight, wrapped into [0, 1440).
	timeMinutes float64
	// paused freezes Update; set by game systems (conversation, menu,
	// journal, combat) via Pause/Resume.
	paused bool
}

// NewGameClock returns a clock at the new-game start time 8:00 AM (Day)
// per ADR-0007 clock parameters, unpaused (ENV-001 AC-1).
func NewGameClock() *GameClock {
	return &GameClock{timeMinutes: StartMinutes}
}

// Update advances the clock by dt real seconds at the 10:1 ratio
// (game_minutes = real_seconds × 10 / 60, ADR-0007): 6 real seconds
// equal 1 game minute (ENV-001 AC-2). Time advances only while not
// paused; a paused clock discards dt entirely (ENV-001 AC-6). At a full
// day the time wraps to 0:00 — Night spans midnight (20:00-2:00), so no
// light-state boundary exists at 0:00. dt ≤ 0 is a no-op: time never
// runs backward.
//
// Performance: a single multiply, add, and modulo — no allocations
// (<0.01ms/frame per ADR-0007 Performance Implications).
func (c *GameClock) Update(dt float64) {
	if c.paused || dt <= 0 {
		return
	}
	c.timeMinutes = math.Mod(c.timeMinutes+dt*gameMinutesPerRealSecond, MinutesPerDay)
}

// Pause freezes the clock: Update discards dt while paused. Game systems
// call Pause when conversation, menu, journal, or combat begins (ADR-0007
// Time Pause). Pause is idempotent.
func (c *GameClock) Pause() {
	c.paused = true
}

// Resume unfreezes the clock after the pausing system ends. Resume is
// idempotent.
func (c *GameClock) Resume() {
	c.paused = false
}

// IsPaused reports whether the clock is currently frozen.
func (c *GameClock) IsPaused() bool {
	return c.paused
}

// SetTime sets the game time in game minutes since midnight, wrapping any
// value into [0, 1440) (negative values wrap into the previous day). Used
// by save/load restore (ADR-0002 full-state serialization) and by test
// fixtures; pause state is unaffected.
func (c *GameClock) SetTime(minutes float64) {
	c.timeMinutes = wrapMinutes(minutes)
}

// TimeMinutes returns game minutes since midnight in [0, 1440) — the
// canonical game-time unit (float64 minutes), matching the
// Event.Timestamp convention in entity/event_bus.go ("game time in
// minutes").
func (c *GameClock) TimeMinutes() float64 {
	return c.timeMinutes
}

// Hour returns the game hour of the current time, 0-23.
func (c *GameClock) Hour() int {
	return int(c.timeMinutes) / 60
}

// Minute returns the game minute of the current time, 0-59.
func (c *GameClock) Minute() int {
	return int(c.timeMinutes) % 60
}

// CurrentState returns the light state active at the current game time
// (ENV-001 AC-3/4/5; see LightStateAt).
func (c *GameClock) CurrentState() LightState {
	return LightStateAt(c.timeMinutes)
}

// Ambient returns the ambient light color/intensity at the current game
// time, smoothly interpolated across light-state boundaries (ENV-001
// AC-3; see AmbientAt). A future presentation system consumes this to
// drive rendering — the clock itself never renders.
func (c *GameClock) Ambient() AmbientLight {
	return AmbientAt(c.timeMinutes)
}

// wrapMinutes wraps game minutes into [0, 1440), the canonical clock
// range. Negative values wrap backward into the previous day.
func wrapMinutes(minutes float64) float64 {
	m := math.Mod(minutes, MinutesPerDay)
	if m < 0 {
		m += MinutesPerDay
	}
	return m
}
