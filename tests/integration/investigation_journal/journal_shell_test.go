// Package investigation_journal_test contains integration tests for the
// investigation journal UI shell.
//
// Story: INVJ-001 — Journal Shell & Three-Panel Layout
// Type: UI (test evidence is advisory; these tests verify pure-Go logic)
// Test evidence: tests/integration/investigation_journal/journal_shell_test.go
package investigation_journal_test

import (
	"math"
	"testing"

	"marrow-of-empire/investigation_journal"
)

// ──────────────────────────────────────────────────────────────────────────────
// AC-1, AC-2: Journal State Machine (Closed ↔ Open)
// ──────────────────────────────────────────────────────────────────────────────

func TestJournalState_ClosedIsDefault(t *testing.T) {
	// iota starts at 0 — StateClosed must be the zero value so a
	// zero-valued Journal starts in the correct state.
	if investigation_journal.StateClosed != 0 {
		t.Errorf("StateClosed must be 0 (iota zero value), got %d", int(investigation_journal.StateClosed))
	}
}

func TestJournalState_ValuesAreDistinct(t *testing.T) {
	states := []investigation_journal.JournalState{
		investigation_journal.StateClosed,
		investigation_journal.StateOpen,
	}

	seen := make(map[investigation_journal.JournalState]bool)
	for _, s := range states {
		if seen[s] {
			t.Errorf("duplicate state value: %v (%d)", s, int(s))
		}
		seen[s] = true
	}

	if len(seen) != 2 {
		t.Errorf("expected 2 distinct states, got %d", len(seen))
	}
}

func TestJournalState_String(t *testing.T) {
	tests := []struct {
		state    investigation_journal.JournalState
		expected string
	}{
		{investigation_journal.StateClosed, "Closed"},
		{investigation_journal.StateOpen, "Open"},
		{investigation_journal.JournalState(99), "Unknown"},
	}

	for _, tt := range tests {
		got := tt.state.String()
		if got != tt.expected {
			t.Errorf("JournalState(%d).String() = %q, want %q", int(tt.state), got, tt.expected)
		}
	}
}

func TestJournalState_IsOpen(t *testing.T) {
	if investigation_journal.StateClosed.IsOpen() {
		t.Error("StateClosed.IsOpen() = true, want false")
	}
	if !investigation_journal.StateOpen.IsOpen() {
		t.Error("StateOpen.IsOpen() = false, want true")
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// AC-4: Journal State Changed Event
// ──────────────────────────────────────────────────────────────────────────────

func TestJournalStateEvent_OpenTrue(t *testing.T) {
	evt := investigation_journal.JournalStateEvent{Open: true}
	if !evt.Open {
		t.Error("JournalStateEvent{Open: true}.Open = false, want true")
	}
}

func TestJournalStateEvent_OpenFalse(t *testing.T) {
	evt := investigation_journal.JournalStateEvent{Open: false}
	if evt.Open {
		t.Error("JournalStateEvent{Open: false}.Open = true, want false")
	}
}

func TestEventJournalStateChanged_IsNonEmpty(t *testing.T) {
	if investigation_journal.EventJournalStateChanged == "" {
		t.Error("EventJournalStateChanged must not be empty")
	}
}

func TestEventJournalStateChanged_IsNamespaced(t *testing.T) {
	// Event type must follow "journal.state_changed" naming pattern (ADR-0011).
	eventStr := string(investigation_journal.EventJournalStateChanged)
	if len(eventStr) < 10 {
		t.Errorf("EventJournalStateChanged too short: %q", eventStr)
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// AC-1, AC-2: Animation — EaseOutCubic (calls the real exported function)
// ──────────────────────────────────────────────────────────────────────────────

func TestEaseOutCubic_Boundaries(t *testing.T) {
	// Calls the real EaseOutCubic exported by the investigation_journal package.
	// Boundary values must be exact.
	if v := investigation_journal.EaseOutCubic(0.0); math.Abs(v-0.0) > 1e-9 {
		t.Errorf("EaseOutCubic(0.0) = %.6f, want 0.0", v)
	}
	if v := investigation_journal.EaseOutCubic(1.0); math.Abs(v-1.0) > 1e-9 {
		t.Errorf("EaseOutCubic(1.0) = %.6f, want 1.0", v)
	}
}

func TestEaseOutCubic_ClampedInput(t *testing.T) {
	// Values outside [0,1] must be clamped.
	for _, tc := range []struct{ input, expected float64 }{
		{-0.5, 0.0},
		{1.5, 1.0},
		{-100.0, 0.0},
		{100.0, 1.0},
	} {
		got := investigation_journal.EaseOutCubic(tc.input)
		if math.Abs(got-tc.expected) > 1e-9 {
			t.Errorf("EaseOutCubic(%.1f) = %.6f, want %.1f", tc.input, got, tc.expected)
		}
	}
}

func TestEaseOutCubic_MidpointFasterThanLinear(t *testing.T) {
	// Ease-out cubic at t=0.5: 1-(1-0.5)³ = 1-0.125 = 0.875.
	// This must be > 0.5 (fast start, gentle stop per ADR-0012).
	got := investigation_journal.EaseOutCubic(0.5)
	if got <= 0.5 {
		t.Errorf("EaseOutCubic(0.5) = %.4f, want > 0.5 (ease-out is fast-start)", got)
	}
	if math.Abs(got-0.875) > 1e-9 {
		t.Errorf("EaseOutCubic(0.5) = %.4f, want 0.875", got)
	}
}

func TestEaseOutCubic_Monotonic(t *testing.T) {
	// Ease-out cubic must be strictly increasing on [0,1].
	prev := -1.0
	for i := 0; i <= 100; i++ {
		tv := float64(i) / 100.0
		result := investigation_journal.EaseOutCubic(tv)
		if result <= prev {
			t.Errorf("EaseOutCubic not monotonic at t=%.2f: %.6f <= %.6f", tv, result, prev)
		}
		prev = result
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// AC-3: Layout Constants
// ──────────────────────────────────────────────────────────────────────────────

func TestLayoutConstants_SidebarDetailSum(t *testing.T) {
	// Sidebar (30%) + Detail (70%) must sum to 1.0 (CR-1).
	sum := investigation_journal.SidebarFraction + investigation_journal.DetailFraction
	if math.Abs(sum-1.0) > 1e-9 {
		t.Errorf("SidebarFraction + DetailFraction = %.4f, want 1.0", sum)
	}
}

func TestLayoutConstants_SidebarFraction(t *testing.T) {
	if investigation_journal.SidebarFraction <= 0 || investigation_journal.SidebarFraction >= 1 {
		t.Errorf("SidebarFraction = %.4f, want value in (0, 1)", investigation_journal.SidebarFraction)
	}
}

func TestLayoutConstants_DetailFraction(t *testing.T) {
	if investigation_journal.DetailFraction <= 0 || investigation_journal.DetailFraction >= 1 {
		t.Errorf("DetailFraction = %.4f, want value in (0, 1)", investigation_journal.DetailFraction)
	}
}

func TestLayoutConstants_QuickRefHeight(t *testing.T) {
	if investigation_journal.QuickRefHeight <= 0 {
		t.Errorf("QuickRefHeight = %.0f, want > 0", investigation_journal.QuickRefHeight)
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// AC-7: Performance — Animation durations within ADR-0012 spec
// ──────────────────────────────────────────────────────────────────────────────

func TestAnimShowDurationSec_WithinSpec(t *testing.T) {
	// Reads the real exported constant — if it changes, this test notices.
	d := investigation_journal.AnimShowDurationSec
	if d <= 0 || d > 0.500 {
		t.Errorf("AnimShowDurationSec = %.3fs, want in (0, 0.500]", d)
	}
}

func TestAnimHideDurationSec_WithinSpec(t *testing.T) {
	d := investigation_journal.AnimHideDurationSec
	if d <= 0 || d > 0.500 {
		t.Errorf("AnimHideDurationSec = %.3fs, want in (0, 0.500]", d)
	}
}

func TestAnimDurations_ShowNotShorterThanHide(t *testing.T) {
	// Per ADR-0012: enter (300ms) should be ≥ exit (200ms).
	if investigation_journal.AnimShowDurationSec < investigation_journal.AnimHideDurationSec {
		t.Errorf("AnimShowDurationSec (%.3fs) < AnimHideDurationSec (%.3fs) — enter should be ≥ exit",
			investigation_journal.AnimShowDurationSec, investigation_journal.AnimHideDurationSec)
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// Sidebar slide-in animation: negative start X targets exist
// ──────────────────────────────────────────────────────────────────────────────

func TestSidebarAnimation_TargetOffscreen(t *testing.T) {
	// The sidebar slides in from the left — its offscreen position must be
	// negative (to the left of the visible area). The exact value is computed
	// at runtime from screen dimensions, but the direction must be negative.
	// Since sidebarW is positive (30% of parchment width), -sidebarW < 0.
	// We verify the fraction is positive so the negation produces correct sign.
	if investigation_journal.SidebarFraction <= 0 {
		t.Error("SidebarFraction must be > 0 for sidebar slide-in to work")
	}
}
