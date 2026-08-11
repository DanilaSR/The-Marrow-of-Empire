// Unit tests for Story ENV-002: Wait / Pass-Time Mechanic.
//
// Evidence path: production/epics/living-environment/
// story-002-wait-pass-time.md
//
// Tests map to the acceptance criteria per ADR-0007 (Wait / Pass Time):
// AC-2 (fast-forward to a target hour at 100× with fade-to-black
// transition, landing exactly on the waited-for light state), AC-5 (wait
// blocked during combat, suspended while paused), plus the midnight wrap
// and light-state behavior of the fast-forward. All time advance is
// driven by explicit Update calls — no timers, no sleeps — so every test
// is deterministic.
package env_test

import (
	"testing"

	"marrow-of-empire/env"
)

// AC-2: GIVEN the player waits from 8:00 to 20:00, WHEN the fast-forward
// runs, THEN 12 game hours (720 minutes) elapse in 7.2 real seconds of
// advance at 100×, wrapped in 500ms fade-out and fade-in, and the clock
// lands exactly on 20:00 — Night.
func TestFastForward_8To20_Elapses720Minutes(t *testing.T) {
	// Arrange — new game: 8:00 AM.
	c := env.NewGameClock()

	// Act — start the wait.
	elapsed := c.FastForward(20)

	// Assert — the wait is precomputed at 12 game hours and the
	// transition begins with a fade-out, time not yet advancing.
	if elapsed != 720 {
		t.Errorf("FastForward(20) = %d, want 720 (12 game hours)", elapsed)
	}
	if !c.IsFastForwarding() {
		t.Error("IsFastForwarding() = false, want true after starting a wait")
	}
	if c.FadePhase() != env.FadingOut {
		t.Errorf("FadePhase() = %v, want FadingOut", c.FadePhase())
	}
	if c.FadeProgress() != 0 {
		t.Errorf("FadeProgress() = %v, want 0 (screen clear)", c.FadeProgress())
	}

	// Act — 250ms into the fade-out.
	c.Update(0.25)

	// Assert — the eased progress is mid-fade (smooth-step(0.5) = 0.5).
	if got := c.FadeProgress(); got != 0.5 {
		t.Errorf("FadeProgress() mid-fade = %v, want 0.5 (smooth-step eased)", got)
	}

	// Act — the rest of the fade-out (250ms): fully black.
	c.Update(0.25)

	// Assert — FadedOut, and game time stayed frozen at 8:00.
	if c.FadePhase() != env.FadedOut {
		t.Errorf("FadePhase() after fade-out = %v, want FadedOut", c.FadePhase())
	}
	if c.FadeProgress() != 1 {
		t.Errorf("FadeProgress() after fade-out = %v, want 1 (black)", c.FadeProgress())
	}
	if got := c.TimeMinutes(); got != 480 {
		t.Errorf("TimeMinutes() during fade = %v, want 480 (frozen)", got)
	}

	// Act — 1 real second at 100× = 100 game minutes (rate check).
	c.Update(1)

	// Assert — 9:40, still fast-forwarding behind black.
	if got := c.TimeMinutes(); got != 580 {
		t.Errorf("TimeMinutes() after 1s = %v, want 580 (100 game minutes at 100×)", got)
	}
	if c.FadePhase() != env.FadedOut {
		t.Errorf("FadePhase() mid-advance = %v, want FadedOut", c.FadePhase())
	}

	// Act — the remaining 6.2 real seconds reach 20:00 exactly (7.2s
	// total advance for 720 game minutes).
	c.Update(6.2)

	// Assert — snapped to 20:00, Night, still black as the fade-in
	// begins.
	if got := c.TimeMinutes(); got != 1200 {
		t.Errorf("TimeMinutes() = %v, want 1200 (20:00)", got)
	}
	if c.Hour() != 20 {
		t.Errorf("Hour() = %d, want 20", c.Hour())
	}
	if c.CurrentState() != env.Night {
		t.Errorf("CurrentState() at 20:00 = %v, want Night", c.CurrentState())
	}
	if c.FadePhase() != env.FadingIn {
		t.Errorf("FadePhase() after advance = %v, want FadingIn", c.FadePhase())
	}
	if c.FadeProgress() != 1 {
		t.Errorf("FadeProgress() at fade-in start = %v, want 1 (still black)", c.FadeProgress())
	}

	// Act — halfway through the fade-in.
	c.Update(0.25)

	// Assert — the screen is half cleared (smooth-step eased).
	if got := c.FadeProgress(); got != 0.5 {
		t.Errorf("FadeProgress() mid-fade-in = %v, want 0.5", got)
	}

	// Act — the rest of the fade-in completes the wait.
	c.Update(0.25)

	// Assert — idle again, time unchanged at 20:00.
	if c.IsFastForwarding() {
		t.Error("IsFastForwarding() = true, want false after the transition")
	}
	if c.FadePhase() != env.NoFade || c.FadeProgress() != 0 {
		t.Errorf("FadePhase()/FadeProgress() = %v/%v, want NoFade/0", c.FadePhase(), c.FadeProgress())
	}
	if got := c.TimeMinutes(); got != 1200 {
		t.Errorf("TimeMinutes() after completion = %v, want 1200", got)
	}
}

// AC-2: GIVEN "Wait until Dawn" selected at 22:00, WHEN the fast-forward
// runs, THEN time wraps past midnight and the light state reaches Dawn
// (6:00 AM) exactly.
func TestFastForward_WrapsPastMidnightToDawn(t *testing.T) {
	// Arrange — 22:00.
	c := env.NewGameClock()
	c.SetTime(22 * 60)

	// Act — wait until Dawn (6:00).
	elapsed := c.FastForward(6)

	// Assert — 8 game hours elapse, wrapping past midnight.
	if elapsed != 480 {
		t.Errorf("FastForward(6) at 22:00 = %d, want 480 (8 game hours)", elapsed)
	}

	// Act — fade-out, then 4.8 real seconds at 100× = 480 game minutes.
	c.Update(env.FastForwardFadeSeconds)
	c.Update(4.8)

	// Assert — landed exactly on 6:00 AM, Dawn.
	if got := c.TimeMinutes(); got != 360 {
		t.Errorf("TimeMinutes() = %v, want 360 (6:00)", got)
	}
	if c.Hour() != 6 {
		t.Errorf("Hour() = %d, want 6", c.Hour())
	}
	if c.CurrentState() != env.Dawn {
		t.Errorf("CurrentState() = %v, want Dawn (AC-2: light state reaches Dawn)", c.CurrentState())
	}

	// Act / Assert — fade-in completes the wait.
	c.Update(env.FastForwardFadeSeconds)
	if c.IsFastForwarding() {
		t.Error("IsFastForwarding() = true, want false after completion")
	}

	// GIVEN a wait crossing midnight itself (23:30 → 1:00), WHEN it
	// runs, THEN the clock wraps into the new day.
	c2 := env.NewGameClock()
	c2.SetTime(23*60 + 30) // 23:30
	if got := c2.FastForward(1); got != 90 {
		t.Errorf("FastForward(1) at 23:30 = %d, want 90 (wraps past 0:00)", got)
	}
	c2.Update(env.FastForwardFadeSeconds)
	c2.Update(0.9) // 90 game minutes at 100×
	if got := c2.TimeMinutes(); got != 60 {
		t.Errorf("TimeMinutes() = %v, want 60 (1:00, wrapped into the new day)", got)
	}
	if c2.Hour() != 1 {
		t.Errorf("Hour() = %d, want 1", c2.Hour())
	}
}

// AC-5: GIVEN combat or conversation active, WHEN the player attempts to
// wait, THEN the wait is blocked (FastForward returns 0 and nothing
// starts); and GIVEN a wait in progress, WHEN combat pauses the clock,
// THEN the fast-forward suspends and resumes where it left off.
func TestFastForward_Paused_BlockedAndSuspended(t *testing.T) {
	// Arrange — combat active when the wait is attempted.
	c := env.NewGameClock()
	c.Pause()

	// Act — attempt the wait.
	elapsed := c.FastForward(20)

	// Assert — blocked: nothing starts, time unchanged.
	if elapsed != 0 {
		t.Errorf("FastForward(20) while paused = %d, want 0 (blocked)", elapsed)
	}
	if c.IsFastForwarding() {
		t.Error("IsFastForwarding() = true, want false — a blocked wait must not start")
	}
	if got := c.TimeMinutes(); got != 480 {
		t.Errorf("TimeMinutes() = %v, want 480 (unchanged)", got)
	}

	// Arrange — combat ends, the wait starts and advances 2 real seconds.
	c.Resume()
	if got := c.FastForward(20); got != 720 {
		t.Errorf("FastForward(20) = %d, want 720", got)
	}
	c.Update(env.FastForwardFadeSeconds) // fade-out
	c.Update(2)                          // +200 game minutes → 11:20

	// Act — combat begins mid-wait: the clock pauses.
	c.Pause()
	c.Update(100) // a long frozen span

	// Assert — the wait suspends: time, phase, and progress are frozen.
	if got := c.TimeMinutes(); got != 680 {
		t.Errorf("TimeMinutes() while paused mid-wait = %v, want 680 (frozen at 11:20)", got)
	}
	if !c.IsFastForwarding() {
		t.Error("IsFastForwarding() = false, want true — the wait suspends, not cancels")
	}
	if c.FadePhase() != env.FadedOut {
		t.Errorf("FadePhase() = %v, want FadedOut", c.FadePhase())
	}

	// Act — combat ends: the wait resumes and completes at 20:00.
	c.Resume()
	c.Update(5.2) // remaining 520 game minutes at 100×

	// Assert — landed exactly on the target.
	if got := c.TimeMinutes(); got != 1200 {
		t.Errorf("TimeMinutes() after resume = %v, want 1200 (20:00)", got)
	}
	c.Update(env.FastForwardFadeSeconds)
	if c.IsFastForwarding() {
		t.Error("IsFastForwarding() = true, want false after completion")
	}
}

// GIVEN a fast-forward crossing light-state boundaries, WHEN time
// advances at 100×, THEN the ADR-0007 state table and boundary blends
// apply exactly as during normal time — the wait crosses Day → Dusk →
// Night with the ambient interpolation active at the boundary hours.
func TestFastForward_RespectsLightStateTransitions(t *testing.T) {
	// Arrange — 8:00 (Day) waiting until 20:00 (Night).
	c := env.NewGameClock()
	c.FastForward(20)
	c.Update(env.FastForwardFadeSeconds)

	// Act — 5.5 real seconds = 550 game minutes → 17:10, inside Dusk.
	c.Update(5.5)

	// Assert — the Day→Dusk crossing happened mid-wait and the boundary
	// blend (window ends 17:01:30) has completed: pure Dusk.
	if c.CurrentState() != env.Dusk {
		t.Errorf("CurrentState() = %v, want Dusk at 17:10", c.CurrentState())
	}
	if !ambientAlmostEqual(c.Ambient(), env.Dusk.Ambient(), 1e-9) {
		t.Error("Ambient() at 17:10 must be pure Dusk (Day→Dusk blend completed)")
	}

	// Act — 1.5 more seconds → 19:40, still Dusk.
	c.Update(1.5)

	// Assert — pure Dusk persists.
	if c.CurrentState() != env.Dusk {
		t.Errorf("CurrentState() = %v, want Dusk at 19:40", c.CurrentState())
	}
	if !ambientAlmostEqual(c.Ambient(), env.Dusk.Ambient(), 1e-9) {
		t.Error("Ambient() at 19:40 must be pure Dusk")
	}

	// Act — 0.2 more seconds reaches 20:00, the target hour; the clock
	// snaps exactly onto the Dusk→Night boundary.
	c.Update(0.2)

	// Assert — the state has switched to Night while the ambient is
	// exactly at the boundary blend midpoint — the transition window is
	// respected even at the arrival instant.
	if c.CurrentState() != env.Night {
		t.Errorf("CurrentState() = %v, want Night at 20:00", c.CurrentState())
	}
	if !ambientAlmostEqual(c.Ambient(), halfBlend(env.Dusk.Ambient(), env.Night.Ambient()), 1e-9) {
		t.Error("Ambient() at 20:00 must be the 50% Dusk→Night blend (boundary respected)")
	}

	// Act / Assert — the wait finishes with the fade-in.
	c.Update(env.FastForwardFadeSeconds)
	if c.IsFastForwarding() {
		t.Error("IsFastForwarding() = true, want false after completion")
	}
}
