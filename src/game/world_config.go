package game

// INT-1 debug-world layout and tuning values (slice standard: named
// constants, no inline magic numbers). Positions are ground-plane (x, z)
// pairs in world units; the town sits at the origin area and the old mine
// lies up the path to the north-east. Colors follow the art bible registers:
// soot/ink earth, muted stone, amber marker, bone-ivory player.

import "kaijuengine.com/matrix"

// Movement tuning.
const (
	worldGroundSize     = 60.0  // ground plane side length (world units)
	worldMoveSpeed      = 5.0   // player walk speed (world units per second)
	worldArriveEpsilon  = 0.25  // stop distance from the move target (world units)
	worldPlayerScale    = 1.0   // debug player cube edge length (world units)
	worldMarkerScale    = 1.0   // NPC marker cube edge length (world units)
)

// Box dimensions: width (x), height (y), depth (z) in world units.
const (
	worldSmithyW, worldSmithyH, worldSmithyD = 3.0, 2.0, 3.0
	worldHouseW, worldHouseH, worldHouseD    = 2.5, 1.8, 2.5
	worldWellW, worldWellH, worldWellD       = 0.8, 0.7, 0.8
	worldMineW, worldMineH, worldMineD       = 3.5, 2.5, 3.5
)

// Ground-plane positions (x, z).
var (
	worldSmithyPos      = matrix.NewVec2(-4, 2)
	worldHousePos       = matrix.NewVec2(3, -2)
	worldWellPos        = matrix.NewVec2(2, 4)
	worldMinePos        = matrix.NewVec2(12, 18)
	worldBlacksmithPos  = matrix.NewVec2(-4.5, 3.5)
	worldMinerPos       = matrix.NewVec2(12.5, 16.5)
	worldScholarPos     = matrix.NewVec2(3.5, -1)
	worldPlayerStartPos = matrix.NewVec2(0, 0)
)

// Interaction tuning.
const (
	// The click ray's ground hit lands up to ~1.7u beyond an NPC's base
	// (camera pitch through the cube's visual body), so the pick radius must
	// absorb that offset for cube clicks to register (INT-2 live-run finding).
	worldNPCClickRadius = 3.0
	// Conversation range: dialogue opens only when the player stands this
	// close to the NPC; distant clicks walk there instead (INT-3 fix).
	worldTalkRange = 3.5
)

// Debug-shape colors (RGB float32, art bible grounded).
var (
	worldColorGround     = matrix.NewColor(0.16, 0.14, 0.11, 1) // soot-earth (#2A231C)
	worldColorStone      = matrix.NewColor(0.42, 0.40, 0.35, 1) // muted stone (#6B645A)
	worldColorMineStone  = matrix.NewColor(0.29, 0.27, 0.24, 1) // darker stone (#4A443C)
	worldColorNPC        = matrix.NewColor(0.78, 0.64, 0.29, 1) // amber NPC markers (#C8A24B)
	worldColorEnemy      = matrix.NewColor(0.48, 0.18, 0.16, 1) // oxblood enemies (#7A2E28 — art bible combat register)
	worldColorInscription = matrix.NewColor(0.90, 0.85, 0.77, 1) // parchment inscription (#E6D9C4 — ivory rationed to empire items)
	worldColorPlayer     = matrix.NewColor(0.91, 0.90, 0.86, 1) // bone ivory (#E9E6DC)
)
