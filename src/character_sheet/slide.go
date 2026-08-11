// This file provides the Go-side paper-slide motion for the character
// sheet. Kaiju's CSS engine has no transition/animation support (verified in
// kaiju/src/engine/ui/markup/css — the title screen documents the same
// limitation), so sheet motion is driven per frame from Go, following the
// title screen's tween pattern.
package character_sheet

// slideTween is a single paper-slide motion: a translateX slide with an
// accompanying overlay opacity fade. Motion follows the art bible desk
// register (ADR-0012, TR-ui-004): translate and opacity only — no bounce,
// no scale, no spin.
type slideTween struct {
	from, to float64 // translateX range in px
	duration float64 // seconds
	elapsed  float64 // seconds elapsed
	entering bool    // entering: ease-out fade in; exiting: ease-in fade out
	progress float64 // eased progress in [0, 1]
	value    float64 // current translateX value in px
	done     bool
}

// newSlide creates a slide from translateX from to translateX to over
// duration seconds. entering selects the easing curve: ease-out for
// entrances, ease-in for exits.
func newSlide(from, to, duration float64, entering bool) *slideTween {
	return &slideTween{
		from:     from,
		to:       to,
		duration: duration,
		entering: entering,
	}
}

// advance progresses the slide by dt seconds. The easing curves are cubic:
// ease-out 1-(1-p)^3 for entrances, ease-in p^3 for exits. A slide whose
// duration has elapsed snaps to its end state and is marked done.
func (t *slideTween) advance(dt float64) {
	if t.done {
		return
	}
	t.elapsed += dt
	p := t.elapsed / t.duration
	if p >= 1 {
		p = 1
		t.done = true
	}
	if t.entering {
		t.progress = 1 - (1-p)*(1-p)*(1-p)
	} else {
		t.progress = p * p * p
	}
	t.value = t.from + (t.to-t.from)*t.progress
}

// complete snaps the slide to its final state. Used by the reduced-motion
// gate and when a slide is interrupted (e.g. Close during the entrance).
func (t *slideTween) complete() {
	t.done = true
	t.progress = 1
	t.value = t.to
}

// opacity returns the overlay opacity for the current progress: 0 → 1 while
// entering, 1 → 0 while exiting.
func (t *slideTween) opacity() float64 {
	if t.entering {
		return t.progress
	}
	return 1 - t.progress
}
