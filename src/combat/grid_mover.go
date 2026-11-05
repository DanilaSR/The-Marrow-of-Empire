package combat

import (
	"kaijuengine.com/matrix"

	"marrow-of-empire/entity"
)

// PositionSetter moves an entity's authoritative position. The Kaiju
// Transform is the authoritative position source (ADR-0001): combat never
// mutates transforms directly — movement goes through the Mover, and the
// Mover applies position changes through this interface. The game wiring
// implements it by setting the Kaiju transform and calling
// EntityStore.UpdatePosition to reindex the spatial index; tests inject a
// stub.
type PositionSetter interface {
	// SetPosition updates the entity's world position.
	SetPosition(id entity.GameEntityID, pos matrix.Vec3) error
}

// TileBlocker reports whether a tile is impassable (ADR-0009: "Blocked:
// occupied entities, wall/obstacle tags"). Tiles occupied by another entity
// block; the combatant's own tile never blocks for itself.
// NewEntityTileBlocker covers entity occupancy. Wall/obstacle tags are not
// filterable through the Entity Framework yet (Query.Tags is accepted but
// not filtered), so scene-authored walls must be composed into a combined
// blocker by the game wiring when that data exists.
type TileBlocker interface {
	// Blocked returns true when the tile cannot be entered by the combatant.
	Blocked(tile Tile, combatant entity.GameEntityID) bool
}

// entityTileBlocker blocks tiles occupied by any entity other than the
// combatant itself (ADR-0009: occupied entities are blocked). Entities are
// located through the injected Positioner (the same interface the enemy AI
// uses), so the blocker works with any position source.
type entityTileBlocker struct {
	store    entity.EntityStore
	position Positioner
}

// NewEntityTileBlocker creates the store-backed blocker: a tile is blocked
// when any other entity's position falls in it. Every entity counts —
// combatants, props, wildlife — because Query has no type filter yet;
// passability rules for items and props arrive with Entity Framework tag
// filtering.
func NewEntityTileBlocker(store entity.EntityStore, position Positioner) TileBlocker {
	return entityTileBlocker{store: store, position: position}
}

// Blocked implements TileBlocker: true when any entity other than the
// combatant occupies the tile.
func (b entityTileBlocker) Blocked(tile Tile, combatant entity.GameEntityID) bool {
	for _, id := range b.store.QueryEntities(entity.Query{}) {
		if id == combatant {
			continue
		}
		pos, ok := b.position.Position(id)
		if !ok {
			continue // unknown or destroyed entity — no position, no tile
		}
		if WorldToTile(pos) == tile {
			return true
		}
	}
	return false
}

// GridMover implements Mover (Story 001's interface) with tile-based,
// AP-costed movement (Story 002, TR-combat-002, ADR-0009). MoveToward
// advances the combatant one tile along the shortest path toward the target
// and spends AP: MoveCost (1) per tile, or the Story 004 climb cost when the
// step ascends elevation (ClimbCost per level climbed — a one-level climb
// costs 2 AP total).
//
// The step is a single tile: the Story 001 enemy AI moves one step per turn,
// and the full route within the AP budget is exposed by Path and
// ReachableTiles for player move commands (future UI story). The search is
// deliberately NOT limited to the AP budget — with the AI moving one step
// per turn, a budget-limited mover would strand distant enemies forever; the
// budget gates route planning, not the single step.
//
// Story 004 (Positioning & Cover): after a successful step the mover reports
// the move to the optional OpportunityWatcher, which resolves the free
// attacks a combatant leaving an enemy's melee range triggers (ADR-0009).
// Every movement path — the enemy AI today, player move commands later —
// flows through MoveToward, so the hook fires exactly when a combatant's
// position changes.
type GridMover struct {
	position Positioner
	setter   PositionSetter
	blocked  TileBlocker
	ap       *APPool
	// grid supplies the Story 004 per-tile elevation for the climb cost.
	// Nil disables elevation costs (all steps cost MoveCost).
	grid *TileGrid
	// opportunity receives the Story 004 step report after every successful
	// move. Nil disables opportunity attacks.
	opportunity OpportunityWatcher
}

// NewGridMover creates the tile mover. The AP pool must be the same pool
// wired into NewCombatManager — the manager resets it at every turn start,
// and movement spends from it; a nil pool panics on the first move. grid
// supplies per-tile elevation for the climb cost (nil = no elevation);
// opportunity receives the step report for opportunity attacks (nil =
// none).
func NewGridMover(position Positioner, setter PositionSetter, blocked TileBlocker, ap *APPool, grid *TileGrid, opportunity OpportunityWatcher) *GridMover {
	return &GridMover{position: position, setter: setter, blocked: blocked, ap: ap, grid: grid, opportunity: opportunity}
}

// MoveToward implements Mover: moves the combatant one tile toward the
// target along the shortest path (Story 002, AC-1/AC-2). The step costs
// MoveCost AP (1) plus ClimbCost-1 per elevation level climbed (Story 004:
// a one-level climb costs 2 AP total), and the combatant's position updates
// to the new tile's center via PositionSetter. After a successful step the
// optional OpportunityWatcher is notified (Story 004: leaving an enemy's
// melee range triggers its free attack).
//
// Errors — the enemy then ends its turn with no action (Story 001 AC-6
// edge):
//   - ErrNoPath when the combatant and target share a tile, either position
//     is unavailable, or every route is blocked. A blocked destination tile
//     (the target's own tile) is still reachable.
//   - ErrInsufficientAP when the pool holds less than the step's cost.
//
// A failed step never spends AP: the budget is checked first and the
// position is applied before the spend; the opportunity report follows only
// a completed step.
func (m *GridMover) MoveToward(combatant, target entity.GameEntityID) error {
	fromPos, ok := m.position.Position(combatant)
	if !ok {
		return ErrNoPath
	}
	toPos, ok := m.position.Position(target)
	if !ok {
		return ErrNoPath
	}
	from, to := WorldToTile(fromPos), WorldToTile(toPos)
	if from == to {
		return ErrNoPath // already in the target's tile — nothing to move toward
	}
	cost := m.stepCost(from, to)
	if m.ap.Current() < cost {
		return ErrInsufficientAP
	}
	path, ok := bfsPath(from, to, m.blocked, combatant, -1) // unbounded: one step per turn
	if !ok {
		return ErrNoPath
	}
	// The step's destination is the FIRST path tile — the opportunity check
	// and the climb cost concern this step, not the target's tile.
	step := path[0]
	if err := m.setter.SetPosition(combatant, TileToWorld(step)); err != nil {
		return err
	}
	// Guaranteed to succeed: the budget was checked above and nothing else
	// spends from the pool on this thread in between.
	if err := m.ap.Spend(cost); err != nil {
		return err
	}
	if m.opportunity != nil {
		m.opportunity.CheckStep(combatant, from, step)
	}
	return nil
}

// stepCost returns the AP cost of a step from tile from to tile to (Story 004,
// ADR-0009): MoveCost (1) per tile, plus ClimbCost-1 (1) per elevation level
// climbed — a one-level climb costs MoveCost + (ClimbCost - 1) = 2 AP total.
// Descending and level steps cost MoveCost. Without a TileGrid (nil grid)
// every step costs MoveCost.
func (m *GridMover) stepCost(from, to Tile) int {
	cost := MoveCost
	if m.grid != nil {
		if climb := m.grid.Elevation(to) - m.grid.Elevation(from); climb > 0 {
			cost += (ClimbCost - MoveCost) * climb
		}
	}
	return cost
}
