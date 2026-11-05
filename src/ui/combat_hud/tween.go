// Paper-slide tween support for the combat HUD (TR-ui-004, ADR-0012
// motion language).
//
// The Kaiju CSS engine does not implement transition/animation/keyframes
// (those properties error out), so all combat HUD motion is driven per
// frame from Go — the same approach the title screen takes. Motion is
// restricted to the paper-slide vocabulary: translateY and border-color
// only, 200ms ease-out, no bounce, no scale, no spin, no flip. The desk
// never trembles: this UI renders on the UI orthographic camera, so world
// camera shake cannot move it.
package combat_hud

// paperSlideCurve is the art bible paper-slide easing curve
// cubic-bezier(0.22, 0.61, 0.36, 1) — the same ease-out used by the title
// screen's entrance motion.
var paperSlideCurve = [4]float64{0.22, 0.61, 0.36, 1}

// solvePaperSlide evaluates the paper-slide cubic bezier at progress x in
// [0, 1]. Bezier X is not invertible in closed form, so the curve
// parameter t is found by bisection (the curve is monotonic in x). The
// result is clamped to [0, 1].
func solvePaperSlide(x float64) float64 {
	if x <= 0 {
		return 0
	}
	if x >= 1 {
		return 1
	}
	lo, hi := 0.0, 1.0
	for i := 0; i < 32; i++ {
		t := (lo + hi) / 2
		if bezierX(paperSlideCurve[0], paperSlideCurve[2], t) < x {
			lo = t
		} else {
			hi = t
		}
	}
	t := (lo + hi) / 2
	return bezierY(paperSlideCurve[1], paperSlideCurve[3], t)
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

// Tween animates a single value from From to To over Duration seconds
// along the paper-slide ease-out curve. A non-positive duration completes
// on the first Update call (used by the reduced-motion gate).
type Tween struct {
	From, To float64
	Duration float64

	elapsed float64
	value   float64
	done    bool
}

// NewPaperSlideTween creates a tween from From to To over Duration seconds
// with the paper-slide ease-out curve.
func NewPaperSlideTween(from, to, duration float64) *Tween {
	return &Tween{
		From:     from,
		To:       to,
		Duration: duration,
		value:    from,
	}
}

// Done reports whether the tween has completed.
func (t *Tween) Done() bool { return t.done }

// Value returns the current interpolated value without advancing.
func (t *Tween) Value() float64 { return t.value }

// Update advances the tween by dt seconds; completed tweens stay done at
// their final value. A non-positive duration completes immediately on the
// first call.
func (t *Tween) Update(dt float64) {
	if t.done {
		return
	}
	if t.Duration <= 0 {
		t.done = true
		t.value = t.To
		return
	}
	t.elapsed += dt
	p := t.elapsed / t.Duration
	if p >= 1 {
		t.done = true
		t.value = t.To
		return
	}
	t.value = t.From + (t.To-t.From)*solvePaperSlide(p)
}
