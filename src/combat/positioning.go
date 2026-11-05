package combat

import (
	"marrow-of-empire/entity"
)

// Story 004 (Positioning & Cover, TR-combat-005, ADR-0009) tuning values.
// The ADR fixes these values; a future balance story may move them into
// combat.Config (data-driven pattern: config.go).
const (
	// FlankingBonus is the accuracy granted when an adjacent ally attacks the
	// target from the opposite side (ADR-0009: "Flanking: +2 accuracy").
	FlankingBonus = 2
	// HalfCoverBonus is the evasion granted when an obstacle sits between the
	// attacker and the defender, adjacent to the defender (ADR-0009: "Half
	// cover: +2 evasion").
	HalfCoverBonus = 2
	// ElevationAccuracyPerLevel is the accuracy granted per elevation level
	// the attacker stands above the target (ADR-0009: "Higher ground: +1
	// accuracy, +1 range").
	ElevationAccuracyPerLevel = 1
	// ElevationRangePerLevel is the range granted per elevation level the
	// attacker stands above the target.
	ElevationRangePerLevel = 1
)

// CoverLevel is the cover a tile provides (ADR-0009, TR-combat-005).
type CoverLevel int

const (
	// CoverNone: the tile provides no cover.
	CoverNone CoverLevel = iota
	// CoverHalf: an obstacle (low wall, pillar) grants its defender +2
	// evasion when it sits between the attacker and the defender, adjacent
	// to the defender.
	CoverHalf
	// CoverFull: a solid structure blocks line of sight entirely — no
	// attack may pass through the tile.
	CoverFull
)

// TileData is the terrain of one grid tile (Story 004): its elevation and
// cover. The combat grid is otherwise featureless (ADR-0009: square tiles).
type TileData struct {
	// Elevation is the tile's height level. An attacker above the target
	// gains +1 accuracy and +1 range per level of difference (ADR-0009);
	// climbing one level costs ClimbCost AP (ap.go).
	Elevation int
	// Cover is the tile's cover level (CoverNone/CoverHalf/CoverFull).
	Cover CoverLevel
}

// TileGrid is the terrain data store of the combat grid: per-tile elevation
// and cover (Story 004). The game wiring populates it from scene-authored
// data (obstacle tags, height fields); tests build it directly. Tiles absent
// from the store are level-0 open ground (CoverNone, Elevation 0).
//
// Elevation intentionally does NOT live on the Tile struct itself: Tile is a
// map key throughout grid.go (BFS visited/parent maps) and WorldToTile always
// yields elevation 0, so an elevated Tile would never compare equal to the
// tile its own position maps back to. Storing terrain in TileGrid keeps Tile
// a pure grid address and the terrain query explicit.
//
// Not safe for concurrent use — the game drives combat from a single thread
// (same contract as CombatManager).
type TileGrid struct {
	data map[Tile]TileData
}

// NewTileGrid creates an empty grid: every tile reports Elevation 0 and
// CoverNone until SetElevation/SetCover store data.
func NewTileGrid() *TileGrid {
	return &TileGrid{data: make(map[Tile]TileData)}
}

// SetElevation records the tile's elevation level.
func (g *TileGrid) SetElevation(t Tile, elevation int) {
	d := g.data[t]
	d.Elevation = elevation
	g.data[t] = d
}

// SetCover records the tile's cover level.
func (g *TileGrid) SetCover(t Tile, cover CoverLevel) {
	d := g.data[t]
	d.Cover = cover
	g.data[t] = d
}

// Elevation returns the tile's elevation level (0 for unset tiles).
func (g *TileGrid) Elevation(t Tile) int {
	return g.data[t].Elevation
}

// Cover returns the tile's cover level (CoverNone for unset tiles).
func (g *TileGrid) Cover(t Tile) CoverLevel {
	return g.data[t].Cover
}

// PositioningModifiers is the aggregate of positional bonuses for one attack
// (Story 004, TR-combat-005, ADR-0009). All values are additive: flanking +
// elevation = +3 accuracy when both apply.
type PositioningModifiers struct {
	// Accuracy is added to the attacker's accuracy (flanking +2, elevation
	// +1 per level above).
	Accuracy int
	// Evasion is added to the target's evasion (half cover +2).
	Evasion int
	// Range is added to the attacker's weapon range (elevation +1 per level
	// above). Computed here; the range-aware targeting/AI consumption lands
	// with Story 006 (Combat Resolution).
	Range int
}

// PositioningResolver computes positional bonuses for an attack and judges
// line of sight (Story 004). It is injected into the d20 attacker
// (attack_resolution.go), which folds the modifiers into the hit roll; nil
// injection preserves the Story 003 resolution exactly. The game wiring
// supplies the NewGridPositioning implementation; tests inject stubs or the
// real grid implementation.
type PositioningResolver interface {
	// Modifiers returns the positional accuracy/evasion/range bonuses for an
	// attack from attacker against target. The zero value when no modifiers
	// apply (adjacent combatants on open, level ground — QA AC-5). Positions
	// unavailable through the Positioner yield zero modifiers (defensive).
	Modifiers(attacker, target entity.GameEntityID) PositioningModifiers
	// HasLineOfSight reports whether a clear line exists between the
	// combatants' tiles: false when any tile strictly between them has full
	// cover (ADR-0009: "Full cover: blocks line of sight"). True for
	// adjacent or co-located combatants (no intermediate tiles). Fail-closed:
	// false when either position is unavailable.
	HasLineOfSight(attacker, target entity.GameEntityID) bool
}

// gridPositioning implements PositioningResolver over grid tiles: positions
// come from the injected Positioner (the same interface the enemy AI uses),
// terrain comes from the TileGrid, and the flanking ally / opportunity enemy
// sets come from the injected FactionReader.
type gridPositioning struct {
	position Positioner
	grid     *TileGrid
	factions FactionReader
}

// Compile-time assertion: gridPositioning satisfies PositioningResolver.
var _ PositioningResolver = (*gridPositioning)(nil)

// NewGridPositioning creates the grid positioning resolver (dependency
// injection — no singletons). position supplies world positions (converted
// via WorldToTile); grid supplies per-tile elevation and cover; factions
// supplies the attacker's allies for flanking. A nil grid falls back to an
// empty TileGrid (level open ground); a nil factions reader disables
// flanking.
//
// Example usage:
//
//	positioning := combat.NewGridPositioning(positioner, grid, combat.NewManagerFactions(store, manager))
//	attacker := combat.NewD20Attacker(store, equipment, roller, positioning)
func NewGridPositioning(position Positioner, grid *TileGrid, factions FactionReader) PositioningResolver {
	if grid == nil {
		grid = NewTileGrid()
	}
	return &gridPositioning{position: position, grid: grid, factions: factions}
}

// Modifiers implements PositioningResolver: sums the flanking bonus (any
// adjacent ally on the opposite side of the target), the half-cover evasion
// bonus (an obstacle tile adjacent to the defender on the attack line), and
// the elevation bonus (accuracy and range per level above). Bonuses are
// additive per ADR-0009.
func (p *gridPositioning) Modifiers(attacker, target entity.GameEntityID) PositioningModifiers {
	atkPos, ok := p.position.Position(attacker)
	if !ok {
		return PositioningModifiers{} // unknown attacker — no positioning (defensive)
	}
	tgtPos, ok := p.position.Position(target)
	if !ok {
		return PositioningModifiers{} // unknown target — no positioning (defensive)
	}
	atkTile, tgtTile := WorldToTile(atkPos), WorldToTile(tgtPos)

	var mods PositioningModifiers
	if p.flanked(atkTile, tgtTile, attacker) {
		mods.Accuracy += FlankingBonus
	}
	mods.Evasion += p.coverEvasion(atkTile, tgtTile)
	if delta := p.grid.Elevation(atkTile) - p.grid.Elevation(tgtTile); delta > 0 {
		mods.Accuracy += delta * ElevationAccuracyPerLevel
		mods.Range += delta * ElevationRangePerLevel
	}
	return mods
}

// HasLineOfSight implements PositioningResolver: a line between the
// combatants' tiles containing any full-cover tile is blocked (ADR-0009:
// "Full cover: blocks line of sight"). Endpoint tiles (the combatants' own)
// never block. Fail-closed: false when either position is unavailable.
func (p *gridPositioning) HasLineOfSight(attacker, target entity.GameEntityID) bool {
	atkPos, ok := p.position.Position(attacker)
	if !ok {
		return false
	}
	tgtPos, ok := p.position.Position(target)
	if !ok {
		return false
	}
	for _, t := range LineTiles(WorldToTile(atkPos), WorldToTile(tgtPos)) {
		if p.grid.Cover(t) == CoverFull {
			return false
		}
	}
	return true
}

// flanked reports whether any ally flanks the target for the attacker:
// the ally stands adjacent to the target (Chebyshev distance 1 — GDD
// "Flanking requires the ally to be adjacent to the target") and the
// attacker lies on the opposite side — the 180° arc check on the XZ grid
// (ADR-0009): the dot product of the attacker's and the ally's offsets from
// the target is strictly negative. Strictness means an attacker exactly
// perpendicular to the ally is NOT flanking ("opposite sides" semantics);
// multiple qualifying allies do not stack beyond the single +2 bonus.
func (p *gridPositioning) flanked(atkTile, tgtTile Tile, attacker entity.GameEntityID) bool {
	if p.factions == nil {
		return false
	}
	for _, ally := range p.factions.AlliesOf(attacker) {
		allyPos, ok := p.position.Position(ally)
		if !ok {
			continue // ally without a position cannot hold the flank
		}
		allyTile := WorldToTile(allyPos)
		if Chebyshev(allyTile, tgtTile) != 1 {
			continue
		}
		atkDX, atkDZ := atkTile.X-tgtTile.X, atkTile.Z-tgtTile.Z
		allyDX, allyDZ := allyTile.X-tgtTile.X, allyTile.Z-tgtTile.Z
		if atkDX*allyDX+atkDZ*allyDZ < 0 {
			return true
		}
	}
	return false
}

// coverEvasion returns the half-cover evasion bonus for the defender: the
// obstacle tile immediately before the defender along the attack line
// (GDD "behind low wall, pillar") grants +2 evasion. The defender's own
// tile and the attacker's own tile never grant cover (a defender standing
// inside a cover tile is a future extension — the story specs obstacles
// adjacent to the defender only).
func (p *gridPositioning) coverEvasion(atkTile, tgtTile Tile) int {
	line := LineTiles(atkTile, tgtTile)
	if len(line) == 0 {
		return 0 // adjacent combatants — no tile between them
	}
	if p.grid.Cover(line[len(line)-1]) == CoverHalf {
		return HalfCoverBonus
	}
	return 0
}

// LineTiles returns the tiles a line crosses between from and to (Story 004
// grid-level line of sight; 3D raycasting is out of scope): the deterministic
// integer Bresenham walk over the 8-connected square grid, in order from
// from toward to. Both endpoints are excluded: the caller's own tile never
// blocks its line. Empty when from == to or the tiles are adjacent (no
// intermediate tile).
//
// Full-cover LoS checks iterate the result; the tile adjacent to the
// destination (the last element) is the "obstacle adjacent to the defender"
// half-cover position. The future targeting UI may reuse it to preview
// attack paths.
func LineTiles(from, to Tile) []Tile {
	dx := to.X - from.X
	dz := to.Z - from.Z
	ax, az := dx, dz
	if ax < 0 {
		ax = -ax
	}
	if az < 0 {
		az = -az
	}
	stepX, stepZ := 1, 1
	if dx < 0 {
		stepX = -1
	}
	if dz < 0 {
		stepZ = -1
	}

	var tiles []Tile
	x, z := from.X, from.Z
	if ax >= az {
		// Shallow line: step X once per iteration, correct Z by the error.
		for err := ax / 2; x != to.X; {
			x += stepX
			err -= az
			if err < 0 {
				z += stepZ
				err += ax
			}
			if x == to.X && z == to.Z {
				break // the destination is reached — excluded from the result
			}
			tiles = append(tiles, Tile{X: x, Z: z})
		}
	} else {
		// Steep line: step Z once per iteration, correct X by the error.
		for err := az / 2; z != to.Z; {
			z += stepZ
			err -= ax
			if err < 0 {
				x += stepX
				err += az
			}
			if x == to.X && z == to.Z {
				break // the destination is reached — excluded from the result
			}
			tiles = append(tiles, Tile{X: x, Z: z})
		}
	}
	return tiles
}
