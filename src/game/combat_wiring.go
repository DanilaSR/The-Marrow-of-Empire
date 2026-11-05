// INT-3 combat wiring: composes the completed combat engine (stories
// 001-005: state machine, initiative, grid/AP, d20 attack resolution, death
// & dying) into the running game and attaches the mine encounter. The combat
// engine was built headless; this is its first live composition.
//
// Slice simplifications (documented for INT-4 / production stories):
//   - no positioning resolver (Story 003 behavior — no flanking/cover)
//   - enemies walk straight lines toward the player (no grid pathing)
//   - no combat HUD yet (INT-2 lesson: never-engine-run UI misrenders)
//   - slice-authored enemy stat blocks; player uses point-buy 13s
package game

import (
	"log/slog"
	"math"

	"kaijuengine.com/engine"
	"kaijuengine.com/matrix"

	"marrow-of-empire/combat"
	"marrow-of-empire/entity"
)

// Slice combat tuning (INT-3).
const (
	combatAttackAPCost    = 2     // ADR-0009: Attack costs 2 AP
	combatRollerSeed      = 42    // deterministic roll sequence for the slice
	encounterTriggerDist  = 4.0   // XZ distance from the mine that starts combat
	wolfAttackReach       = 1.5   // wolf melee reach (slightly more than 1 tile)
	inscriptionDropOffset = 2.0   // inscription cube offset from the mine
)

// Wolf positions at the mine (ground-plane XZ), flanking the entrance.
var wolfPositions = []matrix.Vec2{
	matrix.NewVec2(13.5, 15.5),
	matrix.NewVec2(11.5, 15.0),
}

// wolfEntry is one enemy: its EF entity id and the world cube that
// represents it.
type wolfEntry struct {
	id       entity.GameEntityID
	cube     *engine.Entity
	defeated bool
}

// combatSystem owns the entity framework store, event bus, combat manager,
// and the encounter state for the INT-3 slice.
type combatSystem struct {
	game  *Game
	bus   entity.EventBus
	store entity.EntityStore
	mgr   *combat.CombatManager
	pool  *combat.APPool
	attacker combat.Attacker

	playerEF entity.GameEntityID
	wolves   []*wolfEntry

	armed              bool // encounter re-arms after a defeat
	inscriptionSpawned bool
}

// newCombatSystem composes the combat engine for the game. Called once from
// startNewGame after buildWorld.
func newCombatSystem(g *Game) *combatSystem {
	c := &combatSystem{game: g, armed: true}
	c.bus = entity.NewEventBus()
	c.store = entity.NewEntityStore(g.host, c.bus, entity.EntityStoreConfig{})

	// Player combat entity: slice-default stats (point-buy 13s) + health.
	c.playerEF, _ = c.store.CreateEntity(entity.EntityTypeCharacter, "Investigator", matrix.Vec3Zero())
	_ = entity.SetComponent(c.store, c.playerEF, SlicePlayerStats())
	_ = entity.SetComponent(c.store, c.playerEF, entity.Health{Max: 16, Current: 16})

	c.pool = combat.NewAPPool()
	positioner := &slicePositioner{c: c}
	c.attacker = combat.NewD20Attacker(c.store, &sliceEquipment{c: c},
		combat.NewRandomD20Roller(combatRollerSeed), nil)
	ai := combat.NewBasicEnemyAI(positioner, c.attacker, &sliceMover{c: c}, wolfAttackReach)
	c.mgr = combat.NewCombatManager(c.bus, c.store, sliceClock{}, ai, c.pool, combat.Config{AttackRange: wolfAttackReach})

	c.bus.Subscribe(combat.EventCombatEnded, 1, c.onCombatEnded)
	c.bus.Subscribe(combat.EventEnemyDefeated, 1, c.onEnemyDefeated)
	return c
}

// update runs each world frame: triggers the encounter when the player
// approaches the mine, and resolves player defeat between turns.
func (c *combatSystem) update() {
	if !c.mgr.IsCombatActive() {
		if c.armed && DistanceXZ(c.game.world.player.Transform.WorldPosition(), worldMinePos) < encounterTriggerDist {
			c.startEncounter()
		}
		return
	}
	// Defeat check: the player's health reached zero between turns.
	hp, ok := entity.GetComponent[entity.Health](c.store, c.playerEF)
	if ok && hp.Current <= 0 {
		_ = c.mgr.RemoveCombatant(c.playerEF, combat.RemovalDefeated)
	}
}

// startEncounter spawns the wolf cubes + EF entities and starts combat.
func (c *combatSystem) startEncounter() {
	c.armed = false
	entries := []combat.CombatantEntry{{EntityID: c.playerEF, Side: combat.SidePlayer}}
	for _, p := range wolfPositions {
		cube := c.game.addBox(p.X(), p.Y(), 1.0, 1.0, 1.0, worldColorEnemy)
		id, err := c.store.CreateEntity(entity.EntityTypeWildlife, "Wolf", matrix.NewVec3(p.X(), 0.5, p.Y()))
		if err != nil {
			slog.Error("combat: wolf entity creation failed", "error", err)
			continue
		}
		_ = entity.SetComponent(c.store, id, SliceWolfStats())
		_ = entity.SetComponent(c.store, id, entity.Health{Max: 8, Current: 8})
		c.wolves = append(c.wolves, &wolfEntry{id: id, cube: cube})
		entries = append(entries, combat.CombatantEntry{EntityID: id, Side: combat.SideEnemy})
	}
	if err := c.mgr.StartCombat(entries); err != nil {
		slog.Error("combat: StartCombat failed", "error", err)
		return
	}
	slog.Info("combat: encounter started", "wolves", len(c.wolves))
}

// playerAttack resolves the player's click-to-attack on a wolf: spends AP,
// rolls the d20 resolution, and removes defeated wolves. The player's turn
// ends after the attack.
func (c *combatSystem) playerAttack(w *wolfEntry) {
	if err := c.pool.Spend(combatAttackAPCost); err != nil {
		slog.Warn("combat: attack rejected", "error", err)
		return
	}
	result := c.attacker.ResolveAttack(c.playerEF, w.id)
	slog.Info("combat: player attack", "wolf", w.id, "hit", result.Hit,
		"damage", result.Damage, "crit", result.Crit, "roll", result.Roll)

	// ResolveAttack returns the outcome; the caller applies the damage
	// (the enemy AI does the same internally for its own attacks).
	if result.Hit {
		hp, ok := entity.GetComponent[entity.Health](c.store, w.id)
		if ok {
			hp.Current = max(0, hp.Current-result.Damage)
			_ = entity.SetComponent(c.store, w.id, hp)
			if hp.Current <= 0 {
				c.removeWolf(w)
			}
		}
	}
	_ = c.mgr.EndTurn()
}

// removeWolf removes a defeated wolf from the encounter and the world.
// The cube is destroyed (not merely deactivated — a deactivated entity's
// drawing still renders in this engine version, INT-3 live-run finding).
func (c *combatSystem) removeWolf(w *wolfEntry) {
	if w.defeated {
		return // already removed — repeated clicks on a corpse must not re-fire
	}
	w.defeated = true
	_ = c.mgr.RemoveCombatant(w.id, combat.RemovalDefeated)
	if w.cube != nil {
		c.game.host.DestroyEntity(w.cube)
	}
	slog.Info("combat: wolf defeated", "wolf", w.id)
}

// wolfAt returns the living wolf whose cube lies within radius of the
// clicked ground point (nearest wins), or nil. Defeated wolves are not
// clickable (INT-3 live-run fix: corpses kept intercepting clicks).
func (c *combatSystem) wolfAt(clickX, clickZ, radius float32) *wolfEntry {
	type candidate struct {
		w   *wolfEntry
		pos matrix.Vec2
	}
	var living []candidate
	for _, w := range c.wolves {
		if w.defeated || w.cube == nil {
			continue
		}
		p := w.cube.Transform.WorldPosition()
		living = append(living, candidate{w, matrix.NewVec2(p.X(), p.Z())})
	}
	positions := make([]matrix.Vec2, 0, len(living))
	for _, cand := range living {
		positions = append(positions, cand.pos)
	}
	idx := PickNPC(matrix.NewVec2(clickX, clickZ), positions, radius)
	if idx < 0 {
		return nil
	}
	return living[idx].w
}

// handleClick processes a ground click during the player's combat turn.
func (c *combatSystem) handleClick(hit matrix.Vec3) bool {
	if current, ok := c.mgr.CurrentCombatant(); !ok || current != c.playerEF {
		return true // not the player's turn — consume the click
	}
	if w := c.wolfAt(hit.X(), hit.Z(), worldNPCClickRadius); w != nil {
		c.playerAttack(w)
		return true
	}
	return true
}

// ── Event handlers ───────────────────────────────────────────────────────────

// onEnemyDefeated logs defeats (cube removal happens in removeWolf).
func (c *combatSystem) onEnemyDefeated(ev entity.Event) {
	if data, ok := ev.Data.(combat.EnemyDefeatedData); ok {
		slog.Info("combat: enemy defeated event", "id", data.EntityID)
	}
}

// onCombatEnded handles victory (inscription drops) and defeat (respawn).
func (c *combatSystem) onCombatEnded(ev entity.Event) {
	data, ok := ev.Data.(combat.CombatEndedData)
	if !ok {
		return
	}
	switch data.Outcome {
	case combat.OutcomeVictory:
		c.spawnInscription()
		slog.Info("combat: VICTORY — the inscription is yours")
	case combat.OutcomeDefeat:
		c.respawnPlayer()
		slog.Info("combat: DEFEAT — the investigator wakes back in town")
	}
}

// spawnInscription drops the inscription cube at the mine — the material
// evidence the INT-4 chain consumes.
func (c *combatSystem) spawnInscription() {
	if c.inscriptionSpawned {
		return
	}
	c.inscriptionSpawned = true
	p := worldMinePos
	c.game.addBox(p.X()+inscriptionDropOffset, p.Y(), 0.6, 0.3, 0.6, worldColorInscription)
}

// respawnPlayer resets the player after defeat: back to town, full health,
// surviving wolves cleared, encounter re-armed for another attempt.
func (c *combatSystem) respawnPlayer() {
	c.game.world.player.Transform.SetPosition(matrix.NewVec3(
		worldPlayerStartPos.X(), 0.5, worldPlayerStartPos.Y()))
	c.game.world.moving = false
	_ = entity.SetComponent(c.store, c.playerEF, entity.Health{Max: 16, Current: 16})
	for _, w := range c.wolves {
		if w.cube != nil {
			c.game.host.DestroyEntity(w.cube)
		}
	}
	c.wolves = nil
	c.armed = true
}

// ── Slice implementations of combat interfaces ──────────────────────────────

// sliceEquipment implements combat.EquipmentReader: fixed slice loadouts.
type sliceEquipment struct{ c *combatSystem }

func (e *sliceEquipment) Weapon(id entity.GameEntityID) (combat.WeaponStats, bool) {
	if id == e.c.playerEF {
		// Slice blade: accuracy 12, base damage 4 (melee → STR mod applies).
		return combat.WeaponStats{Accuracy: 12, BaseDamage: 4, Ranged: false}, true
	}
	// Wolf claws: melee, modest.
	return combat.WeaponStats{Accuracy: 9, BaseDamage: 3, Ranged: false}, true
}

func (e *sliceEquipment) Armor(id entity.GameEntityID) int {
	if id == e.c.playerEF {
		return 1 // traveler's leathers
	}
	return 0
}

// slicePositioner implements combat.Positioner from the world cubes.
type slicePositioner struct{ c *combatSystem }

func (p *slicePositioner) Position(id entity.GameEntityID) (matrix.Vec3, bool) {
	if id == p.c.playerEF {
		return p.c.game.world.player.Transform.WorldPosition(), true
	}
	for _, w := range p.c.wolves {
		if w.id == id && w.cube != nil {
			return w.cube.Transform.WorldPosition(), true
		}
	}
	return matrix.Vec3Zero(), false
}

// sliceMover implements combat.Mover: one-meter straight-line steps toward
// the target (no grid pathing in the slice).
type sliceMover struct{ c *combatSystem }

func (m *sliceMover) MoveToward(combatant, target entity.GameEntityID) error {
	var cube *engine.Entity
	for _, w := range m.c.wolves {
		if w.id == combatant {
			cube = w.cube
			break
		}
	}
	if cube == nil {
		return nil
	}
	targetPos, ok := (&slicePositioner{c: m.c}).Position(target)
	if !ok {
		return nil
	}
	pos := cube.Transform.WorldPosition()
	toTarget := targetPos.Subtract(pos)
	toTarget.SetY(0)
	dist := toTarget.Length()
	if dist < 0.01 {
		return nil
	}
	step := toTarget.Scale(1.0 / dist)
	cube.Transform.SetPosition(pos.Add(step))
	return nil
}

// sliceClock implements combat.Clock: the slice has no live game clock
// (ADR-0007 env system is not wired into the game yet), so pause/resume are
// no-ops. Combat still freezes world input via updateWorld gating.
type sliceClock struct{}

func (sliceClock) Pause()  {}
func (sliceClock) Resume() {}

// ── Stat blocks ─────────────────────────────────────────────────────────────

// SlicePlayerStats builds the slice-default player: point-buy 13s plus
// derived values (EF component form — combat reads entity.Stats, not the
// stats-package type; bridging those two is a production story).
func SlicePlayerStats() entity.Stats {
	return entity.Stats{
		Strength: 13, Dexterity: 13, Constitution: 13,
		Intelligence: 13, Wisdom: 13, Charisma: 13,
		MaxHealth: 16, ActionPoints: 5, // 4 + DEX_mod(13) = 5
	}
}

// SliceWolfStats is the slice-authored wolf block: STR 12, DEX 14, CON 10,
// INT 8, WIS 10, CHA 8 (quick, weak, skittish).
func SliceWolfStats() entity.Stats {
	return entity.Stats{
		Strength: 12, Dexterity: 14, Constitution: 10,
		Intelligence: 8, Wisdom: 10, Charisma: 8,
		MaxHealth: 8, ActionPoints: 6, // 4 + DEX_mod(14) = 6
	}
}

// DistanceXZ returns the ground-plane distance between a position and an XZ
// point. Pure — exported for unit testing.
func DistanceXZ(pos matrix.Vec3, p matrix.Vec2) float32 {
	dx := pos.X() - p.X()
	dz := pos.Z() - p.Y()
	return float32(math.Sqrt(float64(dx*dx + dz*dz)))
}
