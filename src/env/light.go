// Light states and ambient light interpolation per ADR-0007, TR-env-002,
// and the Living Environment GDD CR-2: five discrete light states with
// smooth interpolation at boundaries (2-5 game minutes per transition).
//
// Everything here is pure: given game minutes since midnight, LightStateAt
// returns the active state and AmbientAt returns the interpolated ambient
// light. The clock (clock.go) owns time and pause; this file owns the
// time→light mapping, so a future render system can consume AmbientLight
// without any engine dependency (ENV-001: rendering is out of scope).
//
// Story: ENV-001 (production/epics/living-environment/
// story-001-game-clock-light-states.md).
package env

import "fmt"

// LightState identifies one of the five light states per ADR-0007 and
// TR-env-002.
type LightState int

// Light states in canonical order. State hours per ADR-0007 and the GDD
// CR-2 table: Dawn 4:00-7:00, Day 7:00-17:00, Dusk 17:00-20:00, Night
// 20:00-2:00 (spans midnight), Deep Night 2:00-4:00.
const (
	Dawn LightState = iota
	Day
	Dusk
	Night
	DeepNight
	// NumLightStates is the number of light states (five).
	NumLightStates
)

// String returns the state name ("Dawn", "Day", "Dusk", "Night",
// "DeepNight") for logging and debugging.
func (s LightState) String() string {
	switch s {
	case Dawn:
		return "Dawn"
	case Day:
		return "Day"
	case Dusk:
		return "Dusk"
	case Night:
		return "Night"
	case DeepNight:
		return "DeepNight"
	default:
		return fmt.Sprintf("LightState(%d)", int(s))
	}
}

// TransitionMinutes is the duration of a light-state boundary transition
// in game minutes. ADR-0007 and the GDD CR-2 specify 2-5 game minutes
// (12-30 real seconds); 3.0 is the default within that range. Transitions
// are centered on the boundary hour: a transition at boundary B spans
// [B - T/2, B + T/2] game minutes. A test guards that the value stays
// within the ADR range.
const TransitionMinutes = 3.0

// boundary is one light-state transition: at `at` game minutes the state
// changes from `from` to `to`, and the ambient light blends smoothly
// across the centered window [at - TransitionMinutes/2, at +
// TransitionMinutes/2].
type boundary struct {
	at   float64
	from LightState
	to   LightState
}

// stateBoundaries lists every light-state transition in one 24-hour
// cycle per the ADR-0007 state table (TR-env-002). Night spans midnight
// (20:00-2:00), so no boundary exists at 0:00. With TransitionMinutes ≤ 5
// the windows are disjoint and none crosses midnight.
var stateBoundaries = []boundary{
	{at: 120, from: Night, to: DeepNight}, // 2:00 — the coldest hour begins
	{at: 240, from: DeepNight, to: Dawn},  // 4:00 — dawn breaks (ENV-001 AC-3)
	{at: 420, from: Dawn, to: Day},        // 7:00 — full day
	{at: 1020, from: Day, to: Dusk},       // 17:00 — amber hour begins (AC-4)
	{at: 1200, from: Dusk, to: Night},     // 20:00 — cool indigo night (AC-5)
}

// LightStateAt returns the light state active at the given game minutes
// since midnight per the ADR-0007 state table (TR-env-002): Night
// 0:00-2:00 and 20:00-24:00 (spans midnight), Deep Night 2:00-4:00, Dawn
// 4:00-7:00, Day 7:00-17:00, Dusk 17:00-20:00. Any value is wrapped into
// [0, 1440) first.
//
// This pure function is also reusable by ENV-003 NPC schedules (ADR-0007:
// "NPC is active when game time falls within any entry").
func LightStateAt(minutes float64) LightState {
	switch m := wrapMinutes(minutes); {
	case m < 120:
		return Night
	case m < 240:
		return DeepNight
	case m < 420:
		return Dawn
	case m < 1020:
		return Day
	case m < 1200:
		return Dusk
	default:
		return Night
	}
}

// AmbientColor is an RGB color with each channel in [0, 1] (linear, as a
// future render system consumes it).
type AmbientColor struct {
	R, G, B float64
}

// AmbientLight is the ambient light at one moment of game time: an RGB
// color and an intensity scalar. A future presentation system consumes
// AmbientLight to drive sky color, tint, and shadow strength — the light
// logic itself never renders (ENV-001: presentation is out of scope).
type AmbientLight struct {
	Color     AmbientColor
	Intensity float64
}

// Ambient returns the static ambient light of the state — the light the
// world settles into once the boundary transition into this state has
// completed.
//
// The values are PLACEHOLDERS pending art direction (GDD CR-2 light
// character, art bible Section 3 mood targets): Dawn "warm gold rising",
// Day "full sunlight, warm amber baseline", Dusk "amber hour — deepest
// gold", Night "cool indigo", Deep Night "coldest, darkest". Warm
// life-pockets (lanterns, windows, campfires) are localized lights, not
// ambient, and are out of scope here.
func (s LightState) Ambient() AmbientLight {
	switch s {
	case Dawn:
		return AmbientLight{Color: AmbientColor{0.92, 0.72, 0.55}, Intensity: 0.50}
	case Day:
		return AmbientLight{Color: AmbientColor{1.00, 0.97, 0.90}, Intensity: 1.00}
	case Dusk:
		return AmbientLight{Color: AmbientColor{1.00, 0.62, 0.28}, Intensity: 0.55}
	case Night:
		return AmbientLight{Color: AmbientColor{0.30, 0.35, 0.62}, Intensity: 0.30}
	default: // DeepNight
		return AmbientLight{Color: AmbientColor{0.12, 0.14, 0.24}, Intensity: 0.12}
	}
}

// smoothStep applies the Hermite smooth-step curve 3t² - 2t³ to t clamped
// to [0, 1]: both endpoints have zero derivative, giving a seamless
// ease-in/ease-out between light states (GDD CR-2 "smooth
// interpolation").
func smoothStep(t float64) float64 {
	if t <= 0 {
		return 0
	}
	if t >= 1 {
		return 1
	}
	return t * t * (3 - 2*t)
}

// Interpolate blends two ambient lights by factor t in [0, 1] after
// applying the smooth-step curve — the transition interpolation used at
// light-state boundaries (GDD CR-2). t outside [0, 1] is clamped to the
// endpoints.
func Interpolate(a, b AmbientLight, t float64) AmbientLight {
	u := smoothStep(t)
	return AmbientLight{
		Color: AmbientColor{
			R: a.Color.R + (b.Color.R-a.Color.R)*u,
			G: a.Color.G + (b.Color.G-a.Color.G)*u,
			B: a.Color.B + (b.Color.B-a.Color.B)*u,
		},
		Intensity: a.Intensity + (b.Intensity-a.Intensity)*u,
	}
}

// AmbientAt returns the ambient light at the given game minutes since
// midnight. Within TransitionMinutes centered on a boundary hour the
// light blends from the outgoing state's ambient to the incoming state's
// via a smooth-step curve; outside transition windows it equals the
// active state's static ambient (GDD CR-2, TR-env-002, ENV-001 AC-3).
//
// The blend is derived from time alone, so pausing the clock freezes
// transitions implicitly (ENV-001 AC-6), and the result is deterministic
// and allocation-free (a single pass over the five boundaries; <0.1ms
// per ADR-0007 Performance Implications).
func AmbientAt(minutes float64) AmbientLight {
	m := wrapMinutes(minutes)
	half := TransitionMinutes / 2
	for _, b := range stateBoundaries {
		if m >= b.at-half && m <= b.at+half {
			t := (m - (b.at - half)) / TransitionMinutes
			return Interpolate(b.from.Ambient(), b.to.Ambient(), t)
		}
	}
	return LightStateAt(m).Ambient()
}
