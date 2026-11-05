// Package mainmenu implements the navigation accessibility and screen
// transition foundation for "The Marrow of Empire" main menu & settings
// system (Story 001, ADR-0012).
//
// The package is deliberately split into a pure Go core and a thin engine
// adapter:
//
//   - screen_flow.go        — the 7-state screen-flow state machine
//   - focus_manager.go      — per-screen focusable registry and traversal
//   - transition_system.go  — crossfade/hard-cut transitions (injectable clock)
//   - input_router.go       — keyboard/gamepad/mouse → common navigation actions
//   - binding_registry.go   — reserved-key protection (Escape, F9)
//   - focus_style.go        — desk-register palette, WCAG contrast, ring style
//   - engine_input.go       — the only file coupled to the Kaiju engine (hid)
//
// The pure core never touches engine types, so every rule in ADR-0012's
// accessibility contract (tab order, focus ring ≥3:1, reserved keys,
// 250ms crossfade) is unit-testable with an injectable clock.
package mainmenu

// ScreenCrossfadeMs is the crossfade duration for all screen-to-screen
// transitions (GDD `screen_crossfade_ms`). Tuning knob, safe range 100-500ms.
const ScreenCrossfadeMs = 250.0

// FocusRingWidthPx is the focus ring width at the reference DPI
// (ADR-0012: 2px, DPI-scaled). Never rendered below MinRingWidthPx.
const FocusRingWidthPx = 2.0

// MinRingWidthPx floors the ring width so scaling never thins it below the
// 2px accessibility requirement (matters at 1280x720, the minimum
// supported resolution).
const MinRingWidthPx = 2.0

// ReferenceHeightPx is the reference viewport height for height-based
// scaling (ADR-0012: reference 1920x1080).
const ReferenceHeightPx = 1080.0

// MinFocusContrast is the minimum WCAG contrast ratio for the focus ring
// against every menu background (AC-4, ≥3:1).
const MinFocusContrast = 3.0
