// Unit tests for Story ENV-001: Game Clock & Light States.
//
// Evidence path: production/epics/living-environment/
// story-001-game-clock-light-states.md
//
// Tests map to the acceptance criteria (AC-1 .. AC-6) and verify
// TR-env-001 (24-hour clock, 10:1 ratio, 8:00 AM start) and TR-env-002
// (five light states, smooth boundary interpolation) per ADR-0007. Extra
// tests cover the edge cases: midnight wrap, zero/negative dt, and pause
// freezing transitions.
package env_test

import (
	"math"
	"testing"

	"marrow-of-empire/env"
)

// ambientAlmostEqual reports whether two ambient lights match within eps
// on every channel.
func ambientAlmostEqual(a, b env.AmbientLight, eps float64) bool {
	return math.Abs(a.Color.R-b.Color.R) <= eps &&
		math.Abs(a.Color.G-b.Color.G) <= eps &&
		math.Abs(a.Color.B-b.Color.B) <= eps &&
		math.Abs(a.Intensity-b.Intensity) <= eps
}

// halfBlend returns the exact midpoint of a transition (t = 0.5), where
// smooth-step(t) = t and the blend is a plain average.
func halfBlend(a, b env.AmbientLight) env.AmbientLight {
	return env.AmbientLight{
		Color: env.AmbientColor{
			R: (a.Color.R + b.Color.R) / 2,
			G: (a.Color.G + b.Color.G) / 2,
			B: (a.Color.B + b.Color.B) / 2,
		},
		Intensity: (a.Intensity + b.Intensity) / 2,
	}
}

// AC-1: GIVEN game start, WHEN the clock initializes, THEN game time is
// 8:00 AM (Day).
func TestClock_New_StartsAt8AMDay(t *testing.T) {
	// Arrange / Act
	c := env.NewGameClock()

	// Assert
	if c.Hour() != 8 || c.Minute() != 0 {
		t.Errorf("time = %02d:%02d, want 08:00", c.Hour(), c.Minute())
	}
	if c.TimeMinutes() != 480 {
		t.Errorf("TimeMinutes() = %v, want 480", c.TimeMinutes())
	}
	if c.CurrentState() != env.Day {
		t.Errorf("CurrentState() = %v, want Day", c.CurrentState())
	}
	if c.IsPaused() {
		t.Error("new clock must start unpaused")
	}
}

// AC-2: GIVEN 6 real seconds elapsed, WHEN the clock updates, THEN 1 game
// minute has advanced (10:1 ratio).
func TestClock_Update_SixSecondsOneGameMinute(t *testing.T) {
	// Arrange
	c := env.NewGameClock()

	// Act
	c.Update(6)

	// Assert
	if got := c.TimeMinutes(); got != 481 {
		t.Errorf("TimeMinutes() = %v, want 481", got)
	}
}

// GIVEN fractional real seconds, WHEN the clock updates, THEN time
// advances by real_seconds × 10 / 60 game minutes (0.6s → 0.1 min).
func TestClock_Update_FractionalSeconds(t *testing.T) {
	// Arrange
	c := env.NewGameClock()

	// Act
	c.Update(0.6)

	// Assert
	if got := c.TimeMinutes(); math.Abs(got-480.1) > 1e-9 {
		t.Errorf("TimeMinutes() = %v, want 480.1", got)
	}
}

// GIVEN two partial updates, WHEN accumulated across frames, THEN the
// total matches the sum (0.1 + 0.9 game minutes — no drift).
func TestClock_Update_AccumulatesAcrossFrames(t *testing.T) {
	// Arrange
	c := env.NewGameClock()

	// Act — 0.6s then 5.4s, together exactly 6 real seconds.
	c.Update(0.6)
	c.Update(5.4)

	// Assert
	if got := c.TimeMinutes(); math.Abs(got-481) > 1e-9 {
		t.Errorf("TimeMinutes() = %v, want 481", got)
	}
}

// AC-3: GIVEN game time 4:00 AM, WHEN the clock advances, THEN the light
// state transitions from Deep Night to Dawn with smooth interpolation
// over 2-5 game minutes. The transition window is centered on 4:00 and
// spans TransitionMinutes (3.0 default): [3:58:30, 4:01:30].
func TestClock_DeepNightToDawn_TransitionsSmoothly(t *testing.T) {
	// Arrange
	c := env.NewGameClock()

	// Act / Assert — before the window, Deep Night is fully active.
	c.SetTime(238) // 3:58:00
	if c.CurrentState() != env.DeepNight {
		t.Errorf("CurrentState() at 3:58 = %v, want DeepNight", c.CurrentState())
	}
	if !ambientAlmostEqual(c.Ambient(), env.DeepNight.Ambient(), 1e-9) {
		t.Error("Ambient() at 3:58 must be pure Deep Night (no blend yet)")
	}

	// Inside the window but before the boundary hour, the state is still
	// Deep Night while the ambient has begun blending toward Dawn.
	c.SetTime(239) // 3:59:00 — t = 1/6, smooth-step ≈ 0.074
	if c.CurrentState() != env.DeepNight {
		t.Errorf("CurrentState() at 3:59 = %v, want DeepNight", c.CurrentState())
	}
	if ambientAlmostEqual(c.Ambient(), env.DeepNight.Ambient(), 1e-9) {
		t.Error("Ambient() at 3:59 must have begun blending toward Dawn")
	}

	// At 4:00 the state has switched to Dawn and the ambient is midway
	// between Deep Night and Dawn — the smooth transition is in progress
	// (smooth-step(0.5) = 0.5).
	c.SetTime(240) // 4:00
	if c.CurrentState() != env.Dawn {
		t.Errorf("CurrentState() at 4:00 = %v, want Dawn", c.CurrentState())
	}
	if !ambientAlmostEqual(c.Ambient(), halfBlend(env.DeepNight.Ambient(), env.Dawn.Ambient()), 1e-9) {
		t.Errorf("Ambient() at 4:00 = %+v, want 50%% blend of DeepNight→Dawn", c.Ambient())
	}

	// After the window the transition completes and Dawn is fully active.
	c.SetTime(242) // 4:02:00
	if !ambientAlmostEqual(c.Ambient(), env.Dawn.Ambient(), 1e-9) {
		t.Error("Ambient() at 4:02 must be pure Dawn (transition complete)")
	}
}

// AC-4: GIVEN game time 17:00, WHEN the clock advances, THEN Dusk begins
// (amber hour) with smooth transition from Day.
func TestClock_DayToDusk_17Hours(t *testing.T) {
	// Arrange
	c := env.NewGameClock()
	c.SetTime(1020) // 17:00

	// Act
	state := c.CurrentState()

	// Assert — Dusk is active and the ambient is midway between Day and
	// Dusk (amber hour begins, blended).
	if state != env.Dusk {
		t.Errorf("CurrentState() at 17:00 = %v, want Dusk", state)
	}
	if !ambientAlmostEqual(c.Ambient(), halfBlend(env.Day.Ambient(), env.Dusk.Ambient()), 1e-9) {
		t.Errorf("Ambient() at 17:00 = %+v, want 50%% blend of Day→Dusk", c.Ambient())
	}

	// Act — advance through the window: 9 real seconds = 1.5 game
	// minutes → 17:01:30, the window end.
	c.Update(9)

	// Assert — transition complete, pure Dusk amber.
	if !ambientAlmostEqual(c.Ambient(), env.Dusk.Ambient(), 1e-9) {
		t.Error("Ambient() at 17:01:30 must be pure Dusk (transition complete)")
	}
}

// GIVEN the clock advancing across 17:00, WHEN real seconds pass, THEN
// the Update loop itself drives the Day → Dusk state change with the
// ambient blend underway.
func TestClock_Advance_CrossesDuskBoundary(t *testing.T) {
	// Arrange — 16:58:00, just before the Day→Dusk transition window
	// (starts 16:58:30).
	c := env.NewGameClock()
	c.SetTime(1018)

	// Act — 12.1 real seconds ≈ 2.02 game minutes, crossing 17:00.
	c.Update(12.1)

	// Assert — the clock drove Day → Dusk.
	if c.CurrentState() != env.Dusk {
		t.Errorf("CurrentState() after crossing 17:00 = %v, want Dusk", c.CurrentState())
	}
	// Assert — the ambient is blending from Day toward amber Dusk: no
	// longer pure Day, not yet pure Dusk.
	if ambientAlmostEqual(c.Ambient(), env.Day.Ambient(), 1e-9) {
		t.Error("Ambient() must no longer be pure Day after crossing 17:00")
	}
	if ambientAlmostEqual(c.Ambient(), env.Dusk.Ambient(), 1e-9) {
		t.Error("Ambient() must not yet be pure Dusk — transition still in progress")
	}
}

// AC-5: GIVEN game time 20:00, WHEN the clock advances, THEN Night begins
// (cool indigo with warm life-pockets; the pockets are localized lights,
// out of scope — the ambient is cool indigo).
func TestClock_DuskToNight_20Hours(t *testing.T) {
	// Arrange
	c := env.NewGameClock()
	c.SetTime(1200) // 20:00

	// Act
	state := c.CurrentState()

	// Assert — Night is active, ambient midway between Dusk amber and
	// Night indigo.
	if state != env.Night {
		t.Errorf("CurrentState() at 20:00 = %v, want Night", state)
	}
	if !ambientAlmostEqual(c.Ambient(), halfBlend(env.Dusk.Ambient(), env.Night.Ambient()), 1e-9) {
		t.Errorf("Ambient() at 20:00 = %+v, want 50%% blend of Dusk→Night", c.Ambient())
	}

	// Act — advance through the window: 9 real seconds = 1.5 game
	// minutes → 20:01:30, the window end.
	c.Update(9)

	// Assert — pure Night.
	if !ambientAlmostEqual(c.Ambient(), env.Night.Ambient(), 1e-9) {
		t.Error("Ambient() at 20:01:30 must be pure Night (transition complete)")
	}
}

// AC-6: GIVEN conversation active, WHEN the clock would advance, THEN
// time is paused during conversation.
func TestClock_Pause_FreezesTimeDuringConversation(t *testing.T) {
	// Arrange — a conversation starts.
	c := env.NewGameClock()
	c.Pause()

	// Act — a long span of real time passes while paused.
	c.Update(600) // 10 real minutes = 100 game minutes if unpaused

	// Assert — time did not advance.
	if c.TimeMinutes() != 480 {
		t.Errorf("TimeMinutes() = %v, want 480 (frozen)", c.TimeMinutes())
	}
	if !c.IsPaused() {
		t.Error("IsPaused() = false, want true")
	}

	// Act — the conversation ends.
	c.Resume()
	c.Update(6)

	// Assert — time advances again.
	if got := c.TimeMinutes(); got != 481 {
		t.Errorf("TimeMinutes() after resume = %v, want 481", got)
	}
}

// GIVEN the clock paused mid-transition, WHEN time would advance, THEN
// the ambient blend is frozen at the paused value — transitions derive
// from time alone, so pausing freezes them implicitly.
func TestClock_Pause_FreezesTransitions(t *testing.T) {
	// Arrange — land at 4:00, mid DeepNight→Dawn blend.
	c := env.NewGameClock()
	c.SetTime(240)
	c.Pause()

	// Act — the pause spans the whole would-be transition.
	c.Update(30) // 5 real seconds = 5 game minutes unpaused

	// Assert — still halfway blended, not completed.
	if !ambientAlmostEqual(c.Ambient(), halfBlend(env.DeepNight.Ambient(), env.Dawn.Ambient()), 1e-9) {
		t.Error("Ambient() must be frozen mid-blend while paused")
	}
	if c.CurrentState() != env.Dawn {
		t.Errorf("CurrentState() = %v, want Dawn (frozen at 4:00)", c.CurrentState())
	}
}

// GIVEN the clock near midnight, WHEN 6 real seconds pass, THEN the clock
// wraps to 0:00 and the state remains Night — Night spans midnight
// (20:00-2:00), so no boundary exists at 0:00.
func TestClock_Update_WrapsAtMidnight(t *testing.T) {
	// Arrange — 23:59:00.
	c := env.NewGameClock()
	c.SetTime(1439)

	// Act — 12 real seconds = 2 game minutes, crossing midnight.
	c.Update(12)

	// Assert — wrapped into the new day at ~0:01 (float rounding at the
	// wrap boundary is tolerated).
	if got := c.TimeMinutes(); math.Abs(got-1) > 1e-9 {
		t.Errorf("TimeMinutes() = %v, want ≈1 (wrapped past midnight)", got)
	}
	if c.Hour() != 0 {
		t.Errorf("Hour() = %d, want 0 (wrapped into the new day)", c.Hour())
	}
	if c.CurrentState() != env.Night {
		t.Errorf("CurrentState() at 0:00 = %v, want Night", c.CurrentState())
	}

	// Act / Assert — Night continues through midnight with no boundary
	// blend at 0:00.
	c.SetTime(30) // 0:30
	if c.CurrentState() != env.Night {
		t.Errorf("CurrentState() at 0:30 = %v, want Night", c.CurrentState())
	}
	if !ambientAlmostEqual(c.Ambient(), env.Night.Ambient(), 1e-9) {
		t.Error("Ambient() at 0:30 must be pure Night (no midnight boundary)")
	}
}

// GIVEN zero or negative dt, WHEN the clock updates, THEN time does not
// change — time never runs backward.
func TestClock_Update_ZeroAndNegativeDt_NoOp(t *testing.T) {
	// Arrange
	c := env.NewGameClock()

	// Act / Assert
	c.Update(0)
	if got := c.TimeMinutes(); got != 480 {
		t.Errorf("Update(0): TimeMinutes() = %v, want 480", got)
	}
	c.Update(-6)
	if got := c.TimeMinutes(); got != 480 {
		t.Errorf("Update(-6): TimeMinutes() = %v, want 480", got)
	}
}

// GIVEN the full 2.4 real hours of a game day, WHEN the clock updates,
// THEN exactly 1440 game minutes elapse and the clock returns to its
// 8:00 AM start (10:1 ratio, ADR-0007 full cycle).
func TestClock_Update_FullDayCycle(t *testing.T) {
	// Arrange
	c := env.NewGameClock()

	// Act — 2.4 real hours in one update.
	c.Update(2.4 * 3600) // 8640 real seconds = 1440 game minutes

	// Assert — back to the start of the day (float rounding at the wrap
	// boundary is tolerated).
	if got := c.TimeMinutes(); math.Abs(got-480) > 1e-9 {
		t.Errorf("TimeMinutes() = %v, want 480 (back to 8:00 AM)", got)
	}
}

// GIVEN a time with minutes, WHEN Hour and Minute are queried, THEN the
// components are correct (e.g. 8:37 AM = hour 8, minute 37).
func TestClock_HourMinute_Components(t *testing.T) {
	// Arrange
	c := env.NewGameClock()
	c.SetTime(8*60 + 37) // 8:37

	// Act / Assert
	if c.Hour() != 8 || c.Minute() != 37 {
		t.Errorf("Hour()/Minute() = %d/%d, want 8/37", c.Hour(), c.Minute())
	}
}

// GIVEN SetTime with out-of-range minutes, WHEN set, THEN the value wraps
// into [0, 1440) (save/load restore may carry any float).
func TestClock_SetTime_WrapsOutOfRange(t *testing.T) {
	// Arrange
	c := env.NewGameClock()

	// Act — 25:00 wraps to 1:00.
	c.SetTime(1500)
	if got := c.TimeMinutes(); got != 60 {
		t.Errorf("SetTime(1500): TimeMinutes() = %v, want 60", got)
	}

	// Act — -0:30 wraps to 23:30.
	c.SetTime(-30)
	if got := c.TimeMinutes(); got != 1410 {
		t.Errorf("SetTime(-30): TimeMinutes() = %v, want 1410", got)
	}
}

// TR-env-002: GIVEN each state boundary hour, WHEN LightStateAt is
// queried, THEN the state table maps exactly per ADR-0007: Night 20:00-
// 2:00, Deep Night 2:00-4:00, Dawn 4:00-7:00, Day 7:00-17:00, Dusk
// 17:00-20:00.
func TestLightStateAt_StateTable(t *testing.T) {
	cases := []struct {
		minutes float64
		want    env.LightState
	}{
		{0, env.Night},       // 0:00
		{119.9, env.Night},   // 1:59:54
		{120, env.DeepNight}, // 2:00
		{239.9, env.DeepNight},
		{240, env.Dawn}, // 4:00
		{419.9, env.Dawn},
		{420, env.Day}, // 7:00
		{1019.9, env.Day},
		{1020, env.Dusk}, // 17:00
		{1199.9, env.Dusk},
		{1200, env.Night}, // 20:00
		{1439.9, env.Night},
	}
	for _, tc := range cases {
		if got := env.LightStateAt(tc.minutes); got != tc.want {
			t.Errorf("LightStateAt(%v) = %v, want %v", tc.minutes, got, tc.want)
		}
	}
}

// GIVEN the tuning constant, THEN the transition duration lies within the
// ADR-0007 range of 2-5 game minutes.
func TestTransitionMinutes_WithinAdrRange(t *testing.T) {
	if env.TransitionMinutes < 2 || env.TransitionMinutes > 5 {
		t.Errorf("TransitionMinutes = %v, want within [2, 5] per ADR-0007", env.TransitionMinutes)
	}
}

// GIVEN interpolation factors 0 and 1, WHEN Interpolate is applied, THEN
// the result equals the source and destination lights (endpoint pinning
// of the smooth-step curve).
func TestInterpolate_Endpoints(t *testing.T) {
	a := env.Dawn.Ambient()
	b := env.Dusk.Ambient()
	if !ambientAlmostEqual(env.Interpolate(a, b, 0), a, 1e-12) {
		t.Error("Interpolate(..., 0) must equal the source ambient")
	}
	if !ambientAlmostEqual(env.Interpolate(a, b, 1), b, 1e-12) {
		t.Error("Interpolate(..., 1) must equal the destination ambient")
	}
}

// GIVEN factor 0.5, WHEN Interpolate is applied, THEN the result is the
// exact average of both ambients (smooth-step(0.5) = 0.5).
func TestInterpolate_Midpoint(t *testing.T) {
	a := env.Dawn.Ambient()
	b := env.Dusk.Ambient()
	if !ambientAlmostEqual(env.Interpolate(a, b, 0.5), halfBlend(a, b), 1e-12) {
		t.Error("Interpolate(..., 0.5) must be the average of both ambients")
	}
}

// GIVEN each light state, WHEN String is queried, THEN the display name
// matches for logging and debugging.
func TestLightState_String(t *testing.T) {
	want := map[env.LightState]string{
		env.Dawn:      "Dawn",
		env.Day:       "Day",
		env.Dusk:      "Dusk",
		env.Night:     "Night",
		env.DeepNight: "DeepNight",
	}
	for s, name := range want {
		if got := s.String(); got != name {
			t.Errorf("%v.String() = %q, want %q", s, got, name)
		}
	}
}
