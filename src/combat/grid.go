package combat

import (
	"errors"
	"math"
	"sort"

	"kaijuengine.com/matrix"

	"marrow-of-empire/entity"
)

// ErrNoPath is returned when no route exists between two tiles: the
// destination equals the starting tile, it is unreachable within the AP
// budget, or every route is blocked. GridMover surfaces it so the enemy AI
// ends its turn without acting (Story 001 AC-6 edge: "enemy has no path to
// any target").
var ErrNoPath = errors.New("combat: no path")

// Tile is one square-grid cell in the XZ plane (ADR-0009: square grid,
// 1 unit = 1 meter per tile). X maps to world X, Z maps to world Z; the Y
// axis carries elevation, which Story 004 (Positioning) introduces.
type Tile struct {
	// X is the tile column in world-X units.
	X int
	// Z is the tile row in world-Z units.
	Z int
}

// Chebyshev returns the Chebyshev distance between two tiles (ADR-0009):
// max(|dx|, |dz|). On the 8-connected square grid this equals the minimum
// number of single-tile steps between the tiles — a diagonal step costs the
// same as an orthogonal step (1 AP, TR-combat-002, AC-1).
func Chebyshev(a, b Tile) int {
	dx := a.X - b.X
	if dx < 0 {
		dx = -dx
	}
	dz := a.Z - b.Z
	if dz < 0 {
		dz = -dz
	}
	if dx > dz {
		return dx
	}
	return dz
}

// WorldToTile converts a world position to the tile containing it. Floor
// semantics: tile (x, z) covers [x, x+1) × [z, z+1) in world space, so a
// combatant standing at a tile center (X+0.5, Z+0.5) maps back to its own
// tile. The Y component is ignored — the grid lives in the XZ plane.
func WorldToTile(pos matrix.Vec3) Tile {
	return Tile{X: int(math.Floor(float64(pos.X()))), Z: int(math.Floor(float64(pos.Z())))}
}

// TileToWorld returns the tile's center in world coordinates:
// (X+0.5, 0, Z+0.5). Y is 0 — elevation (Story 004) has no grid
// representation yet.
func TileToWorld(t Tile) matrix.Vec3 {
	return matrix.NewVec3(float64(t.X)+0.5, 0, float64(t.Z)+0.5)
}

// neighborOffsets are the eight adjacent tiles in canonical order (west,
// then counter-clockwise). ReachableTiles iterates this fixed order so its
// listing is deterministic; bfsPath iterates the order rotated by progress
// toward the destination (orderedNeighbors) so paths stay straight.
var neighborOffsets = [...]Tile{
	{X: -1, Z: 0}, {X: -1, Z: -1}, {X: 0, Z: -1}, {X: 1, Z: -1},
	{X: 1, Z: 0}, {X: 1, Z: 1}, {X: 0, Z: 1}, {X: -1, Z: 1},
}

// orderedNeighbors returns the eight neighbor offsets sorted by progress
// toward the destination: Chebyshev distance from t+off to to ascending,
// then Manhattan distance ascending. Ties keep the canonical order (stable
// sort). A fixed canonical order alone would make every shortest path hug
// the north edge of the grid — the first-found tie-break would staircase
// toward a target directly east. The ordering changes only which shortest
// path BFS finds, never its length.
func orderedNeighbors(t, to Tile) [8]Tile {
	order := neighborOffsets
	sort.SliceStable(order[:], func(i, j int) bool {
		ci := Chebyshev(Tile{X: t.X + order[i].X, Z: t.Z + order[i].Z}, to)
		cj := Chebyshev(Tile{X: t.X + order[j].X, Z: t.Z + order[j].Z}, to)
		if ci != cj {
			return ci < cj
		}
		mi := manhattan(Tile{X: t.X + order[i].X, Z: t.Z + order[i].Z}, to)
		mj := manhattan(Tile{X: t.X + order[j].X, Z: t.Z + order[j].Z}, to)
		return mi < mj
	})
	return order
}

// manhattan returns the Manhattan distance between two tiles: |dx| + |dz|.
func manhattan(a, b Tile) int {
	dx := a.X - b.X
	if dx < 0 {
		dx = -dx
	}
	dz := a.Z - b.Z
	if dz < 0 {
		dz = -dz
	}
	return dx + dz
}

// ReachableTiles returns every tile the combatant can reach from from within
// the AP budget (AC-3, ADR-0009): a BFS over the 8-connected grid, pruned at
// depth ap, excluding blocked tiles. On an unobstructed grid this is exactly
// the square of tiles with Chebyshev(from, t) <= ap — including from itself
// (distance 0); obstacles carve reachable subsets out of that square (AC-4).
//
// The result is deterministic (BFS order with fixed neighbor iteration) and
// always contains from. combatant is passed to the TileBlocker for
// self-exclusion; pass 0 for tile-centric queries.
func ReachableTiles(from Tile, ap int, blocked TileBlocker, combatant entity.GameEntityID) []Tile {
	if ap < 0 {
		ap = 0
	}
	reachable := []Tile{from}
	visited := map[Tile]bool{from: true}
	frontier := []Tile{from}
	for depth := 1; depth <= ap && len(frontier) > 0; depth++ {
		var next []Tile
		for _, t := range frontier {
			for _, off := range neighborOffsets {
				n := Tile{X: t.X + off.X, Z: t.Z + off.Z}
				if visited[n] || blocked.Blocked(n, combatant) {
					continue
				}
				visited[n] = true
				reachable = append(reachable, n)
				next = append(next, n)
			}
		}
		frontier = next
	}
	return reachable
}

// Path returns the shortest path from from to to within the AP budget
// (AC-4, ADR-0009): a BFS over the 8-connected grid, pruned at depth ap,
// excluding blocked tiles. The returned path lists every step tile after
// from, ending at to; it is deterministic (BFS order with fixed neighbor
// iteration).
//
// The destination tile is enterable even when blocked: a combatant's tile is
// blocked by its occupant, and paths toward a combatant must be able to
// terminate there. Intermediate tiles must be unblocked.
//
// Returns ErrNoPath when to == from, when no route exists, or when the
// shortest route exceeds the budget.
func Path(from, to Tile, ap int, blocked TileBlocker, combatant entity.GameEntityID) ([]Tile, error) {
	if ap < 0 {
		ap = 0
	}
	path, ok := bfsPath(from, to, blocked, combatant, ap)
	if !ok {
		return nil, ErrNoPath
	}
	return path, nil
}

// bfsPath finds the shortest path from from to to over the 8-connected grid,
// excluding blocked tiles (the destination may be blocked). maxSteps >= 0
// caps the search depth (the AP budget); maxSteps < 0 searches without
// limit. Returns the path (from excluded, to included) and whether one was
// found.
func bfsPath(from, to Tile, blocked TileBlocker, combatant entity.GameEntityID, maxSteps int) ([]Tile, bool) {
	if from == to {
		return nil, false
	}
	parent := map[Tile]Tile{}
	visited := map[Tile]bool{from: true}
	frontier := []Tile{from}
	for depth := 1; maxSteps < 0 || depth <= maxSteps; depth++ {
		if len(frontier) == 0 {
			break
		}
		var next []Tile
		for _, t := range frontier {
			for _, off := range orderedNeighbors(t, to) {
				n := Tile{X: t.X + off.X, Z: t.Z + off.Z}
				if visited[n] {
					continue
				}
				if n != to && blocked.Blocked(n, combatant) {
					continue
				}
				visited[n] = true
				parent[n] = t
				if n == to {
					return reconstructPath(parent, from, to), true
				}
				next = append(next, n)
			}
		}
		frontier = next
	}
	return nil, false
}

// reconstructPath walks the BFS parent chain from to back to from and
// returns the tiles in order (from excluded, to included). The chain
// terminates at from, which is never part of the returned path.
func reconstructPath(parent map[Tile]Tile, from, to Tile) []Tile {
	path := []Tile{to}
	for t := to; ; {
		t = parent[t]
		if t == from {
			break
		}
		path = append(path, t)
	}
	for i, j := 0, len(path)-1; i < j; i, j = i+1, j-1 {
		path[i], path[j] = path[j], path[i]
	}
	return path
}
