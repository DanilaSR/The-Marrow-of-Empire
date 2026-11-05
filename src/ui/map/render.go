// Renderer half of the local map: it translates the RegionMapData and
// viewport models into Kaiju document operations — the parchment overlay,
// the world dim, the region linework (ink segments cloned from the hidden
// template), the player indicator, and the fixed parchment furniture
// (region name, hint). All layout motion (slide, dim, surface offset) is
// applied per frame from Go.
//
// All methods are no-ops when the document is nil (headless mode: tests
// exercise the state machine, tween math, and viewport clamping without a
// rendering context).
package local_map

import (
	"fmt"
	"log/slog"
	"math"

	"kaijuengine.com/engine/ui"
	"kaijuengine.com/engine/ui/markup/document"
)

// Ink linework thicknesses (art bible Section 6: ink-on-parchment). The
// classes line-boundary/line-path/line-structure own the ink colors; the
// thickness is applied inline per segment.
const (
	boundaryThickness  = 3.0 // px — region boundaries
	pathThickness      = 2.0 // px — roads and paths
	structureThickness = 2.0 // px — building footprints and walls
)

// Player indicator geometry (AC-4): an 8px ink dot with a 3x12px ink nib
// whose center sits nibDist px from the dot center along the facing
// direction.
const (
	dotSize   = 8.0
	nibWidth  = 3.0
	nibHeight = 12.0
	nibDist   = 9.0
)

// Reference label font sizes (AC-7: body text 16px at reference height,
// never below 12px effective). Multiplied by HeightScaleFactor Go-side.
const (
	regionNameFontSize = 16.0
	hintFontSize       = 12.0
)

// "World" button geometry (Story 003 navigation: zoom out to the world
// map). The tab sits in the top-right corner of the parchment; the rect
// is mirrored in local_map.go (worldButtonHit) and the markup CSS — Go
// positions it inline from these constants so the hit-test never drifts
// from the rendered tab.
const (
	worldButtonWidth  = 72.0 // px
	worldButtonHeight = 28.0 // px
	worldButtonMargin = 20.0 // px from the top-right corner
)

// Marker geometry (Story 002, pen-and-ink icons: single-stroke, 2px at
// 24px, flat fills for palette colors only — TR-ui-003). The sizes are
// mirrored in assets/ui/map.html CSS (the CSS comments note the pairing);
// the constants drive the positioning math here. Marker labels are fixed
// 12px — always at the AC-7 effective minimum, so they need no
// height-based scaling.
const (
	markerIconBox    = 10.0 // pin marker icon box (px) — .marker-npc/.marker-location
	threadDotSize    = 8.0  // amber dot diameter — .marker-thread-dot
	threadStarSize   = 12.0 // amber star box — .marker-thread-star
	starArmLength    = 6.0  // star spoke length (px) — .marker-star-arm height
	starArmThickness = 2.0  // star spoke weight (px) — .marker-star-arm width
)

// mapRenderer owns the bound markup elements and the cloned linework
// segments.
type mapRenderer struct {
	doc *document.Document

	root     *document.Element // map-root: full-screen overlay (translateY slide)
	dim      *document.Element // map-dim: soot world-dim layer (opacity)
	surface  *document.Element // map-surface: pan/scroll viewport content (left/top)
	player   *document.Element // map-player: player dot container
	nib      *document.Element // map-player-nib: facing indicator
	template *document.Element // line-template: hidden clone source for segments

	// Marker clone sources (Story 002): hidden templates duplicated once
	// per marker in rebuildMarkers.
	npcTemplate   *document.Element // npc-marker-template: NPC pin + label
	threadDotTpl  *document.Element // thread-dot-template: amber dot
	threadStarTpl *document.Element // thread-star-template: amber star
	locationTpl   *document.Element // location-marker-template: pin + label + ink X arms

	regionNameElm *document.Element // map-region-name (font-size scaling)
	regionNameLbl *ui.Label         // map-region-name text
	hintLabelElm  *document.Element // map-hint-label (font-size scaling)

	segments []*document.Element // active linework segments
	markers  []*document.Element // active marker elements
}

// newMapRenderer creates the renderer. A nil document puts the renderer in
// headless mode (all operations no-op).
func newMapRenderer(doc *document.Document) *mapRenderer {
	return &mapRenderer{doc: doc}
}

// online reports whether the renderer has a document to operate on.
func (r *mapRenderer) online() bool { return r.doc != nil }

// bind resolves the markup ids and parks the overlay: root hidden, dim at
// full brightness. Open() slides it in.
func (r *mapRenderer) bind() {
	if !r.online() {
		return
	}
	r.root = r.element("map-root")
	r.dim = r.element("map-dim")
	r.surface = r.element("map-surface")
	r.player = r.element("map-player")
	r.nib = r.element("map-player-nib")
	r.template = r.element("line-template")
	r.npcTemplate = r.element("npc-marker-template")
	r.threadDotTpl = r.element("thread-dot-template")
	r.threadStarTpl = r.element("thread-star-template")
	r.locationTpl = r.element("location-marker-template")
	r.regionNameElm = r.element("map-region-name")
	r.hintLabelElm = r.element("map-hint-label")
	if r.regionNameElm != nil {
		r.regionNameLbl = r.regionNameElm.FindElementLabelById("map-region-name")
	}
	// AC-3 accessibility: the amber thread star relies on exactly five
	// .marker-star-arm children in the template — fewer arms produce an
	// incomplete shape; the dot/star distinction depends on this contract
	// (never hue alone, control manifest). Warn loudly on mismatch.
	if r.threadStarTpl != nil {
		arms := 0
		for _, c := range r.threadStarTpl.Children {
			if c.HasClass("marker-star-arm") {
				arms++
			}
		}
		if arms != 5 {
			slog.Warn("local map: thread-star template has wrong arm count", "expected", 5, "found", arms)
		}
	}
	if r.root != nil {
		r.doc.SetElementStyleProperty(r.root, "display", "none")
	}
	// Clone sources park off-screen: the line template and the four marker
	// templates. Clones are shown Go-side (display: block + classes).
	for _, tpl := range []*document.Element{r.template, r.npcTemplate, r.threadDotTpl, r.threadStarTpl, r.locationTpl} {
		if tpl != nil {
			r.doc.SetElementStyleProperty(tpl, "display", "none")
		}
	}
	r.applyDim(0)
}

// element resolves a markup id, logging a warning when missing.
func (r *mapRenderer) element(id string) *document.Element {
	elm, ok := r.doc.GetElementById(id)
	if !ok {
		// Bound ids are part of the map markup contract; a missing one is
		// a content bug, so warn loudly rather than fail silently.
		slog.Warn("local map: markup element not found", "id", id)
		return nil
	}
	return elm
}

// Overlay visibility ------------------------------------------------------

// show displays the overlay (the Open sequence parks it off-screen
// immediately after, so it cannot flash).
func (r *mapRenderer) show() {
	if !r.online() || r.root == nil {
		return
	}
	r.doc.SetElementStyleProperty(r.root, "display", "flex")
}

// hide removes the overlay from the screen (after the exit slide).
func (r *mapRenderer) hide() {
	if !r.online() || r.root == nil {
		return
	}
	r.doc.SetElementStyleProperty(r.root, "display", "none")
}

// applySlide positions the overlay via translateY (paper-slide motion; the
// values come from the map's Go-driven tweens). translateY positive = up
// in Kaiju's layout, so the map parks below the screen edge at negative
// values and slides up into view.
func (r *mapRenderer) applySlide(offsetY float64) {
	if !r.online() || r.root == nil {
		return
	}
	r.doc.SetElementStyleProperty(r.root, "translateY", fmt.Sprintf("%.0fpx", offsetY))
}

// applyDim sets the world-dim layer opacity (0 = world at full brightness,
// worldDimOpacity = world at 30%, AC-1). Driven per frame by the dim tween.
func (r *mapRenderer) applyDim(opacity float64) {
	if !r.online() || r.dim == nil {
		return
	}
	r.doc.SetElementStyleProperty(r.dim, "opacity", fmt.Sprintf("%.3f", opacity))
}

// applySurface positions the region surface inside the parchment — the
// pan/scroll viewport offset (AC-5).
func (r *mapRenderer) applySurface(offsetX, offsetY float64) {
	if !r.online() || r.surface == nil {
		return
	}
	r.doc.SetElementStyleProperty(r.surface, "left", fmt.Sprintf("%.1fpx", offsetX))
	r.doc.SetElementStyleProperty(r.surface, "top", fmt.Sprintf("%.1fpx", offsetY))
}

// Region linework (AC-3) --------------------------------------------------

// rebuildRegion replaces the linework with the region's segments: each
// boundary/path/structure is cloned from the hidden template, classified
// (ink weight + color), and placed on the surface. Also writes the
// surface's pixel extent.
func (r *mapRenderer) rebuildRegion(data RegionMapData, scale float64) {
	if !r.online() {
		return
	}
	for _, elm := range r.segments {
		r.doc.RemoveElement(elm)
	}
	r.segments = nil

	if r.surface != nil {
		r.doc.SetElementStyleProperty(r.surface, "width", fmt.Sprintf("%.0fpx", data.Width*scale))
		r.doc.SetElementStyleProperty(r.surface, "height", fmt.Sprintf("%.0fpx", data.Height*scale))
	}

	surfaceH := data.Height * scale
	count := len(data.Boundaries) + len(data.Paths) + len(data.Structures)
	if r.template == nil || count == 0 {
		return
	}
	clones := r.doc.DuplicateElementRepeat(r.template, count)
	i := 0
	add := func(segments []LineSegment, class string, thickness float64) {
		for _, s := range segments {
			elm := clones[i]
			i++
			r.doc.SetElementIdWithoutApplyStyles(elm, fmt.Sprintf("line-%d", i))
			r.doc.SetElementClassesWithoutApply(elm, "line-segment", class)
			r.doc.SetElementStylePropertyWithoutApply(elm, "display", "block")
			r.styleSegment(elm, s, scale, surfaceH, thickness)
			r.segments = append(r.segments, elm)
		}
	}
	add(data.Boundaries, "line-boundary", boundaryThickness)
	add(data.Paths, "line-path", pathThickness)
	add(data.Structures, "line-structure", structureThickness)
	r.doc.ApplyStyles()
}

// styleSegment places one line segment on the surface: a thin panel from
// the segment's midpoint (rotate spins around the element center), sized
// to the segment length, rotated to the segment's angle. World y-up maps
// to surface y-down, so the midpoint's y flips.
func (r *mapRenderer) styleSegment(elm *document.Element, s LineSegment, scale, surfaceH, thickness float64) {
	dx := (s.X2 - s.X1) * scale
	dy := (s.Y2 - s.Y1) * scale
	length := math.Hypot(dx, dy)
	if length < 1.0 {
		return // degenerate segment — leave the clone empty
	}
	angleDeg := math.Atan2(dy, dx) * 180 / math.Pi
	midX := (s.X1 + s.X2) / 2 * scale
	midY := surfaceH - (s.Y1+s.Y2)/2*scale

	r.doc.SetElementStylePropertyWithoutApply(elm, "left", fmt.Sprintf("%.1fpx", midX-length/2))
	r.doc.SetElementStylePropertyWithoutApply(elm, "top", fmt.Sprintf("%.1fpx", midY-thickness/2))
	r.doc.SetElementStylePropertyWithoutApply(elm, "width", fmt.Sprintf("%.1fpx", length))
	r.doc.SetElementStylePropertyWithoutApply(elm, "height", fmt.Sprintf("%.1fpx", thickness))
	r.doc.SetElementStylePropertyWithoutApply(elm, "rotate", fmt.Sprintf("%.1fdeg", angleDeg))
}

// Player indicator (AC-4) -------------------------------------------------

// applyPlayer places the indicator at the player's surface position with
// the facing nib offset along the facing direction. px/py are surface
// pixels (y-down).
func (r *mapRenderer) applyPlayer(px, py, facing float64) {
	if !r.online() {
		return
	}
	if r.player != nil {
		r.doc.SetElementStyleProperty(r.player, "left", fmt.Sprintf("%.1fpx", px-dotSize/2))
		r.doc.SetElementStyleProperty(r.player, "top", fmt.Sprintf("%.1fpx", py-dotSize/2))
	}
	if r.nib != nil {
		dx, dy := FacingNibOffset(facing, nibDist)
		r.doc.SetElementStyleProperty(r.nib, "left", fmt.Sprintf("%.1fpx", dotSize/2+dx-nibWidth/2))
		r.doc.SetElementStyleProperty(r.nib, "top", fmt.Sprintf("%.1fpx", dotSize/2+dy-nibHeight/2))
	}
}

// FacingNibOffset returns the facing nib's offset (px) from the player dot
// center, along the facing direction at distance dist. Facing is radians
// in the math convention (0 = +x / east, positive = counter-clockwise);
// the returned offset is in screen coordinates (y-down), hence the y flip.
func FacingNibOffset(facing, dist float64) (float64, float64) {
	return math.Cos(facing) * dist, -math.Sin(facing) * dist
}

// Markers (Story 002) ------------------------------------------------------

// rebuildMarkers replaces the marker layer with the snapshot's markers:
// NPC pins (AC-1), amber thread dots and stars (AC-2/AC-3), and location
// pins with optional ink X overlays (AC-4/AC-5). Each marker is a clone of
// its template, positioned on the surface at its world coordinates
// converted to surface pixels (north up). The map trusts the snapshot —
// markers from other regions never appear (AC-6).
func (r *mapRenderer) rebuildMarkers(snap MarkerSnapshot, scale, surfaceH float64) {
	if !r.online() {
		return
	}
	for _, elm := range r.markers {
		r.doc.RemoveElement(elm)
	}
	r.markers = nil

	r.buildNPCMarkers(snap.NPCs, scale, surfaceH)
	r.buildThreadMarkers(snap.Threads, scale, surfaceH)
	r.buildLocationMarkers(snap.Locations, scale, surfaceH)
	r.doc.ApplyStyles()
}

// cloneTemplate duplicates the hidden template count times without
// applying styles (the caller applies once at the end of rebuildMarkers,
// the same batching the linework uses).
func (r *mapRenderer) cloneTemplate(tpl *document.Element, count int) []*document.Element {
	if tpl == nil || count <= 0 {
		return nil
	}
	return r.doc.DuplicateElementRepeatWithoutApplyStyles(tpl, count)
}

// placeMarker positions one cloned marker element on the surface: unique
// id, marker classes (dropping the template class), visible, and centered
// on the marker point (px, py in surface pixels).
func (r *mapRenderer) placeMarker(elm *document.Element, kind string, i int, px, py, box float64) {
	r.doc.SetElementIdWithoutApplyStyles(elm, fmt.Sprintf("marker-%s-%d", kind, i))
	r.doc.SetElementClassesWithoutApply(elm, "marker", "marker-"+kind)
	r.doc.SetElementStylePropertyWithoutApply(elm, "display", "block")
	r.doc.SetElementStylePropertyWithoutApply(elm, "left", fmt.Sprintf("%.1fpx", px-box/2))
	r.doc.SetElementStylePropertyWithoutApply(elm, "top", fmt.Sprintf("%.1fpx", py-box/2))
	r.markers = append(r.markers, elm)
}

// markerLabel returns the name label UI of a cloned pin marker (the
// template's .marker-label child), or nil when missing — a missing inner
// label must never crash the map.
func (r *mapRenderer) markerLabel(elm *document.Element) *ui.Label {
	for _, c := range elm.Children {
		if c.HasClass("marker-label") {
			return c.InnerLabel()
		}
	}
	return nil
}

// buildNPCMarkers clones one named pin per discovered NPC (AC-1). The
// label shows the NPC's display name, truncated at 24 runes; an unnamed
// NPC shows "???" (MarkerDisplayName).
func (r *mapRenderer) buildNPCMarkers(npcs []NPCMarkerData, scale, surfaceH float64) {
	clones := r.cloneTemplate(r.npcTemplate, len(npcs))
	if len(clones) == 0 {
		return
	}
	for i, n := range npcs {
		px, py := WorldToSurface(n.X, n.Y, scale, surfaceH)
		r.placeMarker(clones[i], "npc", i, px, py, markerIconBox)
		if lbl := r.markerLabel(clones[i]); lbl != nil {
			lbl.SetText(TruncateLabel(MarkerDisplayName(n.Name), markerLabelMaxRunes))
		}
	}
}

// buildThreadMarkers clones the amber dot (regular thread, AC-2) or the
// amber star (prioritized thread, AC-3) per active thread location. The
// shapes differ — circle vs five spokes — so priority never relies on hue
// alone (accessibility, control manifest).
func (r *mapRenderer) buildThreadMarkers(threads []ThreadMarkerData, scale, surfaceH float64) {
	dots, stars := 0, 0
	for _, t := range threads {
		if t.IsPrioritized {
			stars++
		} else {
			dots++
		}
	}
	dotClones := r.cloneTemplate(r.threadDotTpl, dots)
	starClones := r.cloneTemplate(r.threadStarTpl, stars)
	di, si := 0, 0
	for _, t := range threads {
		px, py := WorldToSurface(t.LocationX, t.LocationY, scale, surfaceH)
		if t.IsPrioritized {
			if si < len(starClones) {
				r.styleThreadStar(starClones[si], si, px, py)
				si++
			}
		} else if di < len(dotClones) {
			r.styleThreadDot(dotClones[di], di, px, py)
			di++
		}
	}
}

// styleThreadDot places a regular thread's amber dot: an 8px circle
// centered on the referenced location (AC-2).
func (r *mapRenderer) styleThreadDot(elm *document.Element, i int, px, py float64) {
	r.placeMarker(elm, "thread-dot", i, px, py, threadDotSize)
}

// styleThreadStar places a prioritized thread's amber star: five 2px-thick
// spokes radiating from the referenced location (AC-3). Each arm's
// left/top/rotate come from the StarArms geometry; the star's spoke shape
// is distinct from the thread dot's circle (accessibility).
func (r *mapRenderer) styleThreadStar(elm *document.Element, i int, px, py float64) {
	r.placeMarker(elm, "thread-star", i, px, py, threadStarSize)
	arms := make([]*document.Element, 0, 5)
	for _, c := range elm.Children {
		if c.HasClass("marker-star-arm") {
			arms = append(arms, c)
		}
	}
	spokes := StarArms(threadStarSize / 2)
	cx, cy := threadStarSize/2, threadStarSize/2 // star center inside the icon box
	for a := range arms {
		if a >= len(spokes) {
			break
		}
		s := spokes[a]
		r.doc.SetElementStylePropertyWithoutApply(arms[a], "left", fmt.Sprintf("%.1fpx", cx+s.OffsetX-starArmThickness/2))
		r.doc.SetElementStylePropertyWithoutApply(arms[a], "top", fmt.Sprintf("%.1fpx", cy+s.OffsetY-starArmLength/2))
		r.doc.SetElementStylePropertyWithoutApply(arms[a], "rotate", fmt.Sprintf("%.1fdeg", s.RotationDeg))
	}
}

// buildLocationMarkers clones one named pin per discovered location,
// showing the ink X over the pin when the location is cleared (AC-4/AC-5 —
// a cleared location keeps its pin, the X is drawn on top).
func (r *mapRenderer) buildLocationMarkers(locations []LocationMarkerData, scale, surfaceH float64) {
	clones := r.cloneTemplate(r.locationTpl, len(locations))
	if len(clones) == 0 {
		return
	}
	for i, loc := range locations {
		px, py := WorldToSurface(loc.X, loc.Y, scale, surfaceH)
		r.placeMarker(clones[i], "location", i, px, py, markerIconBox)
		if lbl := r.markerLabel(clones[i]); lbl != nil {
			lbl.SetText(TruncateLabel(MarkerDisplayName(loc.Name), markerLabelMaxRunes))
		}
		if loc.Cleared {
			for _, c := range clones[i].Children {
				if c.HasClass("marker-x-arm") {
					r.doc.SetElementStylePropertyWithoutApply(c, "display", "block")
				}
			}
		}
	}
}

// Furniture ---------------------------------------------------------------

// applyRegionName writes the region name label (desk serif, ink).
func (r *mapRenderer) applyRegionName(name string) {
	if !r.online() || r.regionNameLbl == nil {
		return
	}
	r.regionNameLbl.SetText(name)
}

// applyFontScale applies the height-based UI scale (AC-7) to the fixed
// labels: body text 16px at reference height, hint 12px, both never below
// reference size.
func (r *mapRenderer) applyFontScale(scale float64) {
	if !r.online() {
		return
	}
	if r.regionNameElm != nil {
		r.doc.SetElementStyleProperty(r.regionNameElm, "font-size", fmt.Sprintf("%.0fpx", regionNameFontSize*scale))
	}
	if r.hintLabelElm != nil {
		r.doc.SetElementStyleProperty(r.hintLabelElm, "font-size", fmt.Sprintf("%.0fpx", hintFontSize*scale))
	}
}

// destroy tears the markup down. The renderer must not be used afterwards.
func (r *mapRenderer) destroy() {
	if r.online() {
		r.doc.Destroy()
	}
	r.doc = nil
	r.segments = nil
	r.markers = nil
}
