package audio

import (
	"kaijuengine.com/matrix"
)

// Camera is the read-only camera surface the audio listener follows. The
// game's isometric camera (marrow-of-empire/camera.Isometric) satisfies it.
type Camera interface {
	// Position returns the camera position in world space.
	Position() matrix.Vec3
	// Focus returns the world-space point the camera is centered on.
	Focus() matrix.Vec3
}

// Listener is the spatial audio listener state (Story AUDIO-001 AC-2). The
// listener anchors to the camera's focus point — the center of the scene —
// so panning moves it with the camera while zoom changes leave it untouched
// (the QA edge case: "camera zoom changes 4u/10u/20u — listener position
// unchanged", which holds because zoom alters the ortho half-height and the
// camera's standoff position, never its focus).
type Listener struct {
	position matrix.Vec3 // world-space anchor (camera focus)
	at       matrix.Vec3 // forward (look) direction, unit length
	up       matrix.Vec3 // up vector, unit length
}

// Position returns the listener position in world space.
func (l Listener) Position() matrix.Vec3 { return l.position }

// At returns the listener's forward (look) direction.
func (l Listener) At() matrix.Vec3 { return l.at }

// Up returns the listener's up vector.
func (l Listener) Up() matrix.Vec3 { return l.up }
