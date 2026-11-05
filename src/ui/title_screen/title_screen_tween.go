// Package title_screen implements "The Marrow of Empire" title screen UI.
//
// This file provides the Go-side tween system. The Kaiju CSS engine does not
// implement transition/animation/keyframes (those properties error out), so
// all title screen motion is driven per frame from Go code. Motion follows
// the art bible desk register: translateY and opacity only — no bounce, no
// scale, no 3D transforms.
package title_screen

import (
	"kaijuengine.com/matrix"
)

// Easing identifies one of the three easing curves from the title screen
// visual design brief Section 4.
type Easing int

const (
	// EaseOut is cubic-bezier(0.22, 0.61, 0.36, 1). Used for all entries.
	EaseOut Easing = iota
	// EaseIn is cubic-bezier(0.55, 0.06, 0.68, 0.19). Used for all exits.
	EaseIn
	// EaseInOut is cubic-bezier(0.65, 0, 0.35, 1). Used for crossfades only.
	EaseInOut
)

// bezierCurves maps each easing to its cubic-bezier control points [x1, y1, x2, y2].
var bezierCurves = [3][4]float64{
	EaseOut:   {0.22, 0.61, 0.36, 1},
	EaseIn:    {0.55, 0.06, 0.68, 0.19},
	EaseInOut: {0.65, 0, 0.35, 1},
}

// curve returns the easing function for the given easing, evaluating the
// cubic-bezier at progress p in [0, 1].
func curve(e Easing) func(p float64) float64 {
	cp := bezierCurves[e]
	return func(p float64) float64 {
		return solveCubicBezier(cp[0], cp[1], cp[2], cp[3], p)
	}
}

// solveCubicBezier evaluates the cubic bezier defined by control points
// (x1, y1) and (x2, y2) at progress x in [0, 1]. Bezier X is not invertible
// in closed form, so the curve parameter t is found by bisection (the easing
// curves used here are monotonic in x). The result is clamped to [0, 1].
func solveCubicBezier(x1, y1, x2, y2, x float64) float64 {
	if x <= 0 {
		return 0
	}
	if x >= 1 {
		return 1
	}
	lo, hi := 0.0, 1.0
	for i := 0; i < 32; i++ {
		t := (lo + hi) / 2
		if bezierX(x1, x2, t) < x {
			lo = t
		} else {
			hi = t
		}
	}
	t := (lo + hi) / 2
	return bezierY(y1, y2, t)
}

// bezierX evaluates the X component of a cubic bezier.
func bezierX(x1, x2, t float64) float64 {
	mt := 1 - t
	return 3*mt*mt*t*x1 + 3*mt*t*t*x2 + t*t*t
}

// bezierY evaluates the Y component of a cubic bezier.
func bezierY(y1, y2, t float64) float64 {
	mt := 1 - t
	return 3*mt*mt*t*y1 + 3*mt*t*t*y2 + t*t*t
}

// Tween is a single value animation from From to To over Duration seconds,
// using one of the desk-register easing curves.
type Tween struct {
	From, To matrix.Float
	Duration float64
	Delay    float64

	Easing Easing
	Atmospheric bool

	curve   func(p float64) float64
	elapsed float64
	value   matrix.Float
	done    bool
}

// NewTween creates a tween from From to To over Duration seconds with the
// given easing. A non-positive duration completes on the first Update call.
func NewTween(from, to matrix.Float, duration float64, easing Easing) *Tween {
	return &Tween{
		From:     from,
		To:       to,
		Duration: duration,
		Easing:   easing,
		curve:    curve(easing),
		value:    from,
	}
}

// WithDelay returns the tween with a delay before it starts advancing.
func (t *Tween) WithDelay(d float64) *Tween {
	t.Delay = d
	return t
}

// SetAtmospheric marks the tween as atmospheric (retained under reduced motion).
func (t *Tween) SetAtmospheric() *Tween {
	t.Atmospheric = true
	return t
}

// Done reports whether the tween has completed.
func (t *Tween) Done() bool { return t.done }

// Value returns the current interpolated value without advancing the tween.
func (t *Tween) Value() matrix.Float { return t.value }

// Update advances the tween by dt seconds and returns the current value and
// whether the tween completed on this call.
func (t *Tween) Update(dt float64) (matrix.Float, bool) {
	if t.done {
		return t.value, true
	}
	if t.Duration <= 0 {
		t.done = true
		t.value = t.To
		return t.value, true
	}
	t.elapsed += dt
	if t.elapsed < t.Delay {
		return t.value, false
	}
	p := (t.elapsed - t.Delay) / t.Duration
	if p >= 1 {
		t.done = true
		t.value = t.To
		return t.value, true
	}
	eased := t.curve(p)
	t.value = t.From + (t.To-t.From)*matrix.Float(eased)
	return t.value, false
}

// TweenRunner owns the screen's active tweens and the reduced-motion gate.
type TweenRunner struct {
	reducedMotion bool
	tweens        []*Tween
}

// NewTweenRunner creates an empty tween runner.
func NewTweenRunner() *TweenRunner { return &TweenRunner{} }

// SetReducedMotion enables or disables the reduced-motion gate.
func (r *TweenRunner) SetReducedMotion(on bool) {
	if r.reducedMotion == on {
		return
	}
	r.reducedMotion = on
	if on {
		for _, t := range r.tweens {
			if !t.Atmospheric {
				t.Duration = 0
				t.Delay = 0
				t.done = true
				t.value = t.To
			}
		}
	}
}

// ReducedMotion reports whether the gate is enabled.
func (r *TweenRunner) ReducedMotion() bool { return r.reducedMotion }

// Play starts a tween. Under reduced motion, non-atmospheric tweens complete instantly.
func (r *TweenRunner) Play(t *Tween) {
	t.value = t.From
	t.done = false
	t.elapsed = 0
	if r.reducedMotion && !t.Atmospheric {
		t.Duration = 0
		t.Delay = 0
	}
	r.tweens = append(r.tweens, t)
}

// Update advances every active tween by dt seconds and drops completed ones.
func (r *TweenRunner) Update(dt float64) {
	if len(r.tweens) == 0 {
		return
	}
	alive := r.tweens[:0]
	for _, t := range r.tweens {
		t.Update(dt)
		if !t.Done() {
			alive = append(alive, t)
		}
	}
	r.tweens = alive
}

// Clear cancels all tweens without completing them.
func (r *TweenRunner) Clear() {
	r.tweens = r.tweens[:0]
}
