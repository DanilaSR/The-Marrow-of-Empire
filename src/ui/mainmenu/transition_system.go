package mainmenu

// TransitionMode selects how a screen transition animates.
type TransitionMode int

const (
	// TransitionCrossfade is the 250ms fade used for all screen-to-screen
	// transitions (AC-7). Screen A fades out while screen B fades in.
	TransitionCrossfade TransitionMode = iota
	// TransitionHardCut is the instant cut used by the Loading state
	// (GDD transition rules: "The Loading state uses a hard cut to black").
	TransitionHardCut
)

// TransitionSystem animates screen transitions on an injectable clock so
// timing assertions are deterministic and tolerance-free (GDD measurement
// protocol: progress is exactly clamp(t_game / duration, 0, 1) per frame).
//
// Completion is time-based, not frame-based: a transition completes the
// moment the clock passes the end, even if a frame time spikes. Input
// blocking during the transition is the ScreenFlow's concern (it discards
// actions while Active()).
type TransitionSystem struct {
	now        func() float64 // injectable game-time clock, milliseconds
	durationMs float64
	startMs    float64
	mode       TransitionMode
	active     bool
	done       bool
}

// NewTransitionSystem creates a transition system reading time from the
// given clock function. The clock is the single source of time for all
// timing assertions; tests inject a fake clock.
func NewTransitionSystem(now func() float64) *TransitionSystem {
	return &TransitionSystem{
		now:        now,
		durationMs: ScreenCrossfadeMs,
	}
}

// SetCrossfadeMs overrides the crossfade duration (tuning knob,
// GDD `screen_crossfade_ms`, safe range 100-500ms).
func (t *TransitionSystem) SetCrossfadeMs(ms float64) {
	t.durationMs = ms
}

// CrossfadeMs returns the configured crossfade duration.
func (t *TransitionSystem) CrossfadeMs() float64 { return t.durationMs }

// Begin starts a transition. It returns false if a transition is already
// active — a second Begin cannot start (AC-7: rapid double-click on a
// transition-triggering button must not cause a double-transition).
func (t *TransitionSystem) Begin(mode TransitionMode) bool {
	if t.active {
		return false
	}
	t.active = true
	t.done = false
	t.mode = mode
	t.startMs = t.now()
	return true
}

// Active reports whether a transition is in progress. While active, the
// ScreenFlow discards all input (AC-7).
func (t *TransitionSystem) Active() bool { return t.active }

// Progress returns the crossfade progress in [0, 1]:
// clamp((now - start) / duration, 0, 1). A hard cut is always 1; an idle
// system reports 1 (fully shown). Exact at the unit level — no easing is
// applied to the timing itself.
func (t *TransitionSystem) Progress() float64 {
	if !t.active {
		return 1
	}
	if t.mode == TransitionHardCut {
		return 1
	}
	p := (t.now() - t.startMs) / t.durationMs
	if p <= 0 {
		return 0
	}
	if p >= 1 {
		return 1
	}
	return p
}

// Update advances the transition and reports completion exactly once
// (edge-triggered): the first call whose clock has passed the end returns
// true. Hard cuts complete on the first Update after Begin. A frame-time
// spike cannot stall a transition — completion depends only on the clock.
func (t *TransitionSystem) Update() bool {
	if !t.active || t.done {
		return false
	}
	if t.mode == TransitionHardCut || t.now()-t.startMs >= t.durationMs {
		t.active = false
		t.done = true
		return true
	}
	return false
}
