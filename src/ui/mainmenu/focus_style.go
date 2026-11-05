package mainmenu

import "math"

// Color is an opaque sRGB color used by the focus ring style system. The
// palette below mirrors the desk register from the art bible / title screen
// stylesheet (assets/ui/title_screen.html); values are data-driven so the
// art director can tune them without code changes.
type Color struct {
	R, G, B uint8
}

// Desk-register palette. The focus ring candidates are ordered by visual
// preference: ink first (AC-4 "ink outline"), then oxblood (ADR-0012
// "oxblood/umber focus ring"), then parchment for dark backgrounds.
var (
	// ColorParchment is the desk panel background (#E3D5B0).
	ColorParchment = Color{0xE3, 0xD5, 0xB0}
	// ColorInk is the ink-dark text/outline color (#1C1813).
	ColorInk = Color{0x1C, 0x18, 0x13}
	// ColorOxblood is the destructive/ring accent (#7A2E28).
	ColorOxblood = Color{0x7A, 0x2E, 0x28}
	// ColorDarkStone is the save-slot card background (GDD: "dark stone
	// card", selected slot styling). Used by Story 004; included now so the
	// ≥3:1 ring guarantee is verified against it.
	ColorDarkStone = Color{0x2B, 0x24, 0x20}
	// ColorAmber is the accent/fill color (#D9A441). It is NOT a ring
	// candidate: amber-on-parchment is ~1.5:1 (control manifest forbids
	// amber-on-parchment text).
	ColorAmber = Color{0xD9, 0xA4, 0x41}
	// ColorIvory is the ivory text color for destructive fills (#F5EEDC).
	// Ivory-on-oxblood is ~8:1, passing the >= 4.5:1 WCAG AA requirement
	// for the destructive button text (Story 006 AC-4).
	ColorIvory = Color{0xF5, 0xEE, 0xDC}
	// ColorDimOverlay is the pause dim quad color: black, with the alpha
	// animated by PauseMenu.DimAlpha (Story 006 AC-1b, world dims to 30%).
	ColorDimOverlay = Color{0x00, 0x00, 0x00}
)

// ringCandidates are the ring colors tried in order of visual preference.
// Ink passes ≥3:1 on parchment; parchment passes on dark stone and black
// overlays; oxblood passes on parchment only. The list is data-driven —
// palette edits here propagate to the ring guarantee automatically.
var ringCandidates = []Color{ColorInk, ColorOxblood, ColorParchment}

// RelativeLuminance returns the WCAG 2.x relative luminance of c.
func RelativeLuminance(c Color) float64 {
	r := linearize(c.R)
	g := linearize(c.G)
	b := linearize(c.B)
	return 0.2126*r + 0.7152*g + 0.0722*b
}

// linearize converts an sRGB channel value to linear light.
func linearize(v uint8) float64 {
	s := float64(v) / 255.0
	if s <= 0.04045 {
		return s / 12.92
	}
	return math.Pow((s+0.055)/1.055, 2.4)
}

// ContrastRatio returns the WCAG contrast ratio between two colors
// ((L1+0.05)/(L2+0.05) with L1 ≥ L2), in the range [1, 21].
func ContrastRatio(a, b Color) float64 {
	la, lb := RelativeLuminance(a), RelativeLuminance(b)
	if la < lb {
		la, lb = lb, la
	}
	return (la + 0.05) / (lb + 0.05)
}

// DPIScaleForHeight returns the height-based DPI scale for a viewport
// height (ADR-0012: reference 1920x1080, height-based scaling).
func DPIScaleForHeight(height float64) float64 {
	return height / ReferenceHeightPx
}

// FocusStyle resolves the focus ring treatment (color + width) for a
// focusable element. The ring color is selected per background so the
// ≥3:1 contrast requirement holds against every menu background — a single
// fixed color cannot (oxblood is ~6.4:1 on parchment but ~1.6:1 on dark
// stone).
type FocusStyle struct {
	// DPIScale is the current height-based DPI scale (1.0 at 1080p).
	DPIScale float64
}

// NewFocusStyle creates a focus style at the given DPI scale.
func NewFocusStyle(dpiScale float64) *FocusStyle {
	return &FocusStyle{DPIScale: dpiScale}
}

// RingWidth returns the ring width in pixels: 2px at reference DPI,
// DPI-scaled up, never below MinRingWidthPx (AC-4: 2px width, DPI-scaled).
func (s *FocusStyle) RingWidth() float64 {
	return math.Max(MinRingWidthPx, FocusRingWidthPx*s.DPIScale)
}

// FocusRingFor returns the ring color for the given background: the first
// palette candidate achieving ≥3:1 contrast. If no candidate passes (not
// expected with the desk palette), the highest-contrast candidate is
// returned so the ring is never invisible.
func (s *FocusStyle) FocusRingFor(bg Color) Color {
	for _, c := range ringCandidates {
		if ContrastRatio(c, bg) >= MinFocusContrast {
			return c
		}
	}
	best, bestRatio := ringCandidates[0], 0.0
	for _, c := range ringCandidates {
		if r := ContrastRatio(c, bg); r > bestRatio {
			best, bestRatio = c, r
		}
	}
	return best
}
