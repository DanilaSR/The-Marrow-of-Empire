// Marker support for the local map (Story 002: Map Markers & Pins,
// TR-ui-001/003, TR-kg-005, TR-thread-001/004).
//
// Markers are the second layer of the desk-register map, on top of the
// Story 001 linework: NPC pins at their last known positions (AC-1), amber
// thread dots and stars (AC-2/AC-3), discovered location pins (AC-4), and
// ink X marks over cleared locations (AC-5). Like the player position
// (Story 001), marker data comes through provider interfaces the game
// layer implements — the map never reads entity stores directly and never
// mutates game state. The map trusts its providers (AC-7): an undiscovered
// NPC is simply absent from the provider's list, so no omniscient marker
// can appear.
//
// Markers are region-scoped (AC-6): providers receive the current region
// ID (RegionMapData.RegionID) and return only that region's markers. The
// map snapshots the provider data when it opens and on SetRegion; the game
// layer calls RefreshMarkers to push knowledge changes while the map is
// open (e.g. a thread reaching revelation_ready, AC-3).
//
// The models in this file own no rendering state — the mapRenderer
// (render.go) translates the MarkerSnapshot into document operations, and
// all pure math (world-to-surface conversion, label truncation, star-arm
// geometry) lives here so the headless integration tests can pin it.
package local_map

import (
	"math"
	"slices"
)

// NPCMarkerData describes one discovered NPC's map marker: the entity, its
// display name, and its last known position in region world units (meters,
// y-up). Only discovered NPCs appear (AC-1, AC-7) — the provider decides;
// the map never looks NPCs up itself.
type NPCMarkerData struct {
	EntityID uint64
	Name     string
	X, Y     float64 // world position (region meters, y-up)
	RegionID string
}

// ThreadMarkerData describes one location reference of an active
// investigation thread. A thread referencing several locations yields one
// entry per location (the provider fans out); a thread without a location
// reference yields none (AC-2 QA edge).
type ThreadMarkerData struct {
	ThreadID      string
	LocationX     float64
	LocationY     float64
	RegionID      string
	IsPrioritized bool // true = amber star (revelation_ready), false = amber dot
}

// LocationMarkerData describes one discovered location pin: the entity,
// its discovered name, its position, and whether it has been cleared or
// destroyed (an ink X is drawn over the pin, AC-5).
type LocationMarkerData struct {
	EntityID uint64
	Name     string
	X, Y     float64
	RegionID string
	Cleared  bool // true = ink X over the pin
}

// NPCMarkerProvider supplies discovered NPC positions for map markers
// (AC-1, AC-7). The game layer implements it from the entity spatial index
// (ADR-0001) plus the discovery tracking (PWS/PlayerKnowledge); the map
// depends on the interface and renders exactly what it returns, for the
// given region only (AC-6).
type NPCMarkerProvider interface {
	DiscoveredNPCs(regionID string) []NPCMarkerData
}

// ThreadMarkerProvider supplies active investigation thread locations for
// map markers (AC-2, AC-3). The game layer implements it from
// PlayerKnowledge (ADR-0010), mapping each active thread's location entity
// reference to a world position; threads without a location reference are
// omitted.
type ThreadMarkerProvider interface {
	ActiveThreads(regionID string) []ThreadMarkerData
}

// LocationMarkerProvider supplies discovered and cleared locations for map
// markers (AC-4, AC-5). The game layer implements it from persistent world
// state flags (discovered / cleared); the map renders each entry as a
// named pin, adding the ink X when Cleared is set.
type LocationMarkerProvider interface {
	DiscoveredLocations(regionID string) []LocationMarkerData
}

// MarkerSnapshot is the marker layer's current state: the provider data
// for the installed region, exactly as rendered on the surface
// (region-scoped, AC-6). Empty when no region (or no RegionID) is
// installed.
type MarkerSnapshot struct {
	NPCs      []NPCMarkerData
	Threads   []ThreadMarkerData
	Locations []LocationMarkerData
}

// MarkerKind identifies the icon shape rendered for a thread marker. The
// dot and the star are deliberately different shapes, never hue alone —
// the priority signal must survive colorblind modes (control manifest,
// accessibility).
type MarkerKind int

const (
	// MarkerKindThreadDot is the regular thread marker: an 8px amber
	// circle (AC-2).
	MarkerKindThreadDot MarkerKind = iota
	// MarkerKindThreadStar is the prioritized thread marker: a 12px amber
	// five-spoke star (AC-3, TR-thread-004 revelation_ready).
	MarkerKindThreadStar
)

// ThreadMarkerKind returns the icon kind for a thread marker: the amber
// star for prioritized threads (revelation_ready), the amber dot for
// regular active threads (AC-3). The shapes differ — dot vs star — so the
// distinction is visible to colorblind players.
func ThreadMarkerKind(prioritized bool) MarkerKind {
	if prioritized {
		return MarkerKindThreadStar
	}
	return MarkerKindThreadDot
}

// markerLabelMaxRunes is the maximum rune length of a marker label; longer
// names are truncated with an ellipsis (AC-4 QA edge).
const markerLabelMaxRunes = 24

// TruncateLabel truncates name to at most maxRunes runes, replacing the
// excess with an ellipsis character (U+2026). Names at or under the limit
// are returned unchanged. Rune-based, so multi-byte text (accents, RTL
// scripts) is never split mid-character.
func TruncateLabel(name string, maxRunes int) string {
	if maxRunes <= 0 {
		return ""
	}
	r := []rune(name)
	if len(r) <= maxRunes {
		return name
	}
	return string(r[:maxRunes-1]) + "…"
}

// MarkerDisplayName returns the label text for a named marker: the name
// itself, or "???" when the name is unknown (AC-7 QA edge: the player saw
// the NPC but never learned its name). The map owns this fallback — the
// provider is trusted for which NPCs appear, not for label presentation.
func MarkerDisplayName(name string) string {
	if name == "" {
		return "???"
	}
	return name
}

// WorldToSurface maps region world coordinates (meters, y-up) to surface
// pixels (y-down, north up) — the surface's y axis flips so north is up.
// scale is the map scale in px per world unit; surfaceHeight is the region
// surface height in px. Positions are passed through unclamped (markers
// outside the region simply land off-surface, like the player indicator).
func WorldToSurface(x, y, scale, surfaceHeight float64) (float64, float64) {
	return x * scale, surfaceHeight - y*scale
}

// StarArm is one spoke of the amber thread-star icon: its center offset
// from the star center (surface px, y-down) and its rotation in degrees
// (Kaiju's clockwise-positive screen convention, matching the linework).
type StarArm struct {
	OffsetX     float64
	OffsetY     float64
	RotationDeg float64
}

// StarArms returns the five spokes of the amber thread-star icon (AC-3):
// radius px long from the star center, evenly spaced 72° apart, with one
// spoke pointing north (up on the map). Each arm is a thin panel whose
// long axis runs from the star center outward; the offsets place the arm
// centers at radius/2 from the star center, and the rotations orient the
// long axes along the spokes. The five-spoke shape is distinct from the
// thread dot's circle — never hue alone.
func StarArms(radius float64) []StarArm {
	arms := make([]StarArm, 5)
	for i := range arms {
		// Spoke i at π/2 + i·72° in the math convention (0 = east, CCW);
		// FacingNibOffset converts to screen coordinates (y-down).
		angle := math.Pi/2 + float64(i)*(2*math.Pi/5)
		dx, dy := FacingNibOffset(angle, radius/2)
		arms[i] = StarArm{
			OffsetX:     dx,
			OffsetY:     dy,
			RotationDeg: -float64(i) * 72.0,
		}
	}
	return arms
}

// Marker management --------------------------------------------------------

// SetNPCMarkerProvider installs the discovered-NPC source for map markers
// (AC-1). nil disables NPC markers. The provider is snapshotted on Open
// and SetRegion; call RefreshMarkers to re-query it while the map is open.
func (m *LocalMap) SetNPCMarkerProvider(p NPCMarkerProvider) { m.npcProvider = p }

// SetThreadMarkerProvider installs the active-thread source for map
// markers (AC-2/AC-3). nil disables thread markers.
func (m *LocalMap) SetThreadMarkerProvider(p ThreadMarkerProvider) { m.threadProvider = p }

// SetLocationMarkerProvider installs the discovered/cleared location
// source for map markers (AC-4/AC-5). nil disables location markers.
func (m *LocalMap) SetLocationMarkerProvider(p LocationMarkerProvider) { m.locationProvider = p }

// Markers returns the current marker snapshot: the provider data for the
// installed region, exactly as rendered on the surface (AC-6). The returned
// snapshot is a deep copy — the caller can mutate it without affecting the
// map's rendered layer. Empty when no region is installed or the region has
// no RegionID.
func (m *LocalMap) Markers() MarkerSnapshot {
	return MarkerSnapshot{
		NPCs:      slices.Clone(m.snapshot.NPCs),
		Threads:   slices.Clone(m.snapshot.Threads),
		Locations: slices.Clone(m.snapshot.Locations),
	}
}

// RefreshMarkers re-queries the marker providers for the current region
// and re-renders every marker. Called automatically on Open (snapshot at
// open) and on SetRegion; the game layer calls it when knowledge changes
// while the map is open — e.g. an NPC is discovered or a thread reaches
// revelation_ready (AC-3 QA edge).
func (m *LocalMap) RefreshMarkers() { m.refreshMarkers() }

// refreshMarkers snapshots the marker providers for the current region and
// rebuilds the rendered marker layer. Without a region ID there is nothing
// to scope by, so the marker layer is cleared (AC-6: region-scoped).
//
// The provider slices are copied into the snapshot — a snapshot that the
// game layer could mutate underneath the map would not be a snapshot, and
// the map's rendered layer would drift from its state. Knowledge changes
// reach the map through RefreshMarkers instead.
func (m *LocalMap) refreshMarkers() {
	m.snapshot = MarkerSnapshot{}
	if m.region.RegionID != "" {
		if m.npcProvider != nil {
			m.snapshot.NPCs = slices.Clone(m.npcProvider.DiscoveredNPCs(m.region.RegionID))
		}
		if m.threadProvider != nil {
			m.snapshot.Threads = slices.Clone(m.threadProvider.ActiveThreads(m.region.RegionID))
		}
		if m.locationProvider != nil {
			m.snapshot.Locations = slices.Clone(m.locationProvider.DiscoveredLocations(m.region.RegionID))
		}
	}
	_, rh := m.regionPxSize()
	m.renderer.rebuildMarkers(m.snapshot, m.scale, rh)
}
