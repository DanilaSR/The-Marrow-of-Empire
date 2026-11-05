// INT-1 world bootstrap: a debug-shape game world proving the loop glue —
// isometric camera swap, entity/drawing rendering, click-to-move input, and
// camera follow. All shapes are colored unit meshes; KitBash3D content
// replaces them once the asset pipeline lands. Gameplay systems (dialogue,
// combat, journal) attach in later integration stories.
package game

import (

	"kaijuengine.com/engine"
	"kaijuengine.com/engine/assets"
	"kaijuengine.com/matrix"
	"kaijuengine.com/platform/hid"
	"kaijuengine.com/registry/shader_data_registry"
	"kaijuengine.com/rendering"

	"marrow-of-empire/camera"
	"marrow-of-empire/entity"
)

// worldState is the INT-1 game world runtime state: the player entity and
// the current click-to-move destination.
type worldState struct {
	player     *engine.Entity
	moveTarget matrix.Vec3
	moving     bool

	// INT-2: NPC cubes by GameEntityID, linked to the talk system after load.
	npcCubes map[entity.GameEntityID]*engine.Entity
}

// startNewGame destroys the title screen and enters the game world: builds
// the debug-shape town, swaps in the fixed isometric camera (ADR-0003), and
// starts the world update loop.
func (g *Game) startNewGame() {
	if g.titleDoc != nil {
		g.titleDoc.Destroy()
		g.titleDoc = nil
	}

	g.buildWorld()
	g.talk = newTalkSystem(g)
	g.combat = newCombatSystem(g)
	g.journal = newJournalView(g)
	g.worldUpdateID = g.host.Updater.AddUpdate(g.updateWorld)
}

// buildWorld constructs the INT-1 world: primary camera, ground plane, town
// structures, the mine, the blacksmith marker, and the player entity.
func (g *Game) buildWorld() {
	winW := float32(g.host.Window.Width())
	winH := float32(g.host.Window.Height())
	g.iso = camera.NewIsometric(matrix.Vec3Zero(), camera.DefaultConfig(), camera.ZoomDefault, winW, winH)
	g.host.Cameras.Primary.ChangeCamera(g.iso.Camera())

	g.addShape(rendering.NewMeshPlane(g.host.MeshCache()), matrix.Vec3Zero(), matrix.NewVec3XYZ(worldGroundSize), worldColorGround)
	g.addBox(worldSmithyPos.X(), worldSmithyPos.Y(), worldSmithyW, worldSmithyH, worldSmithyD, worldColorStone)
	g.addBox(worldHousePos.X(), worldHousePos.Y(), worldHouseW, worldHouseH, worldHouseD, worldColorStone)
	g.addBox(worldWellPos.X(), worldWellPos.Y(), worldWellW, worldWellH, worldWellD, worldColorStone)
	g.addBox(worldMinePos.X(), worldMinePos.Y(), worldMineW, worldMineH, worldMineD, worldColorMineStone)

	player := g.addBox(worldPlayerStartPos.X(), worldPlayerStartPos.Y(), worldPlayerScale, worldPlayerScale, worldPlayerScale, worldColorPlayer)

	// NPC marker cubes (INT-2): amber figures at their .kg-authored spots.
	npcCubes := map[entity.GameEntityID]*engine.Entity{
		entBlacksmith: g.addBox(worldBlacksmithPos.X(), worldBlacksmithPos.Y(), worldMarkerScale, worldMarkerScale, worldMarkerScale, worldColorNPC),
		entMiner:      g.addBox(worldMinerPos.X(), worldMinerPos.Y(), worldMarkerScale, worldMarkerScale, worldMarkerScale, worldColorNPC),
		entScholar:    g.addBox(worldScholarPos.X(), worldScholarPos.Y(), worldMarkerScale, worldMarkerScale, worldMarkerScale, worldColorNPC),
	}

	g.world = &worldState{
		player:     player,
		moveTarget: player.Transform.WorldPosition(),
		npcCubes:   npcCubes,
	}
}

// npcCubeFor returns the world cube for a GameEntityID (talk.go link-up).
func (w *worldState) npcCubeFor(id entity.GameEntityID) *engine.Entity {
	if w == nil {
		return nil
	}
	return w.npcCubes[id]
}

// updateWorld runs the world loop each frame: a left-click either opens
// dialogue with an NPC near the click point or sets a walk destination via
// the camera's plane-hit raycast; the player walks at a constant speed and
// the camera follows. While the dialogue panel is open, world input is
// suspended (dialogue owns the interaction).
func (g *Game) updateWorld(deltaTime float64) {
	w := g.world
	mouse := g.host.Window.Mouse

	// Journal toggle (J) — handled before the dialogue input gate.
	if g.host.Window.Keyboard.KeyHeld(hid.KeyboardKeyJ) {
		if g.journal != nil {
			g.journal.Toggle()
		}
	}

	// INT-3 combat: the encounter triggers on proximity even mid-dialogue,
	// and combat input outranks the dialogue panel (an ambush interrupts
	// conversation).
	if g.combat != nil {
		g.combat.update()
		if g.combat.mgr.IsCombatActive() {
			if g.talk != nil && g.talk.panel.IsVisible() {
				g.talk.panel.Hide()
			}
			if mouse.Pressed(hid.MouseButtonLeft) {
				if hit, ok := g.iso.Camera().TryPlaneHit(mouse.ScreenPosition(), matrix.Vec3Zero(), matrix.Vec3Up()); ok {
					g.combat.handleClick(hit)
				}
			}
			g.iso.SetFocus(w.player.Transform.WorldPosition())
			return
		}
	}

	if g.talk != nil && g.talk.panel.IsVisible() {
		return
	}

	if mouse.Pressed(hid.MouseButtonLeft) {
		if hit, ok := g.iso.Camera().TryPlaneHit(mouse.ScreenPosition(), matrix.Vec3Zero(), matrix.Vec3Up()); ok {
			// NPC dialogue opens only within conversation range; a click on
			// a distant NPC walks there instead (CRPG convention — the
			// click-anywhere slice simplification let distant clicks open
			// dialogue and freeze the walk, INT-3 live-run fix).
			if npc := g.talk.npcAt(hit); npc != nil &&
				DistanceXZ(w.player.Transform.WorldPosition(), npc.pos) <= worldTalkRange {
				w.moving = false
				g.talk.openDialogue(npc)
			} else {
				w.moveTarget = hit
				w.moving = true
			}
		}
	}

	if w.moving {
		next, arrived := MoveToward(w.player.Transform.WorldPosition(), w.moveTarget, worldMoveSpeed, float32(deltaTime), worldArriveEpsilon)
		w.player.Transform.SetPosition(next)
		w.moving = !arrived
	}

	g.iso.SetFocus(w.player.Transform.WorldPosition())
}

// addBox creates a ground-standing box entity: the cube mesh is centered at
// its origin, so the position is raised by half the height to sit on the
// ground plane. Scale is (w, h, d) in world units.
func (g *Game) addBox(x, z, w, h, d float32, color matrix.Color) *engine.Entity {
	return g.addShape(
		rendering.NewMeshCube(g.host.MeshCache()),
		matrix.NewVec3(x, h/2, z),
		matrix.NewVec3(w, h, d),
		color,
	)
}

// addShape creates an entity with the given unit mesh positioned at pos,
// scaled by scale, and tinted color. The drawing's Transform links to the
// entity so it tracks movement, and it is enqueued for rendering with the
// primary camera's view culler.
func (g *Game) addShape(mesh *rendering.Mesh, pos, scale matrix.Vec3, color matrix.Color) *engine.Entity {
	e := engine.NewEntity(g.host.WorkGroup())
	e.Transform.SetPosition(pos)
	e.Transform.SetScale(scale)

	sd := shader_data_registry.Create("basic")
	sd.(*shader_data_registry.ShaderDataStandard).Color = color

	mat, err := g.host.MaterialCache().Material(assets.MaterialDefinitionBasic)
	if err != nil {
		panic("INT-1: basic material not found in content database: " + err.Error())
	}
	tex, err := g.host.TextureCache().Texture(assets.TextureSquare, rendering.TextureFilterLinear)
	if err != nil {
		panic("INT-1: square texture not found in content database: " + err.Error())
	}

	g.host.Drawings.AddDrawing(rendering.Drawing{
		Material:   mat.CreateInstance([]*rendering.Texture{tex}),
		Mesh:       mesh,
		ShaderData: sd,
		Transform:  &e.Transform,
		ViewCuller: &g.host.Cameras.Primary,
	})
	e.StoreShaderData(sd)
	return e
}

// MoveToward moves from current toward target at speed for dt seconds along
// the ground plane (Y is preserved) and reports arrival. Movement stops
// within epsilon of the target; the final step snaps exactly to the target
// instead of overshooting. A target already within epsilon reports arrived
// immediately. Exported so the movement math is unit-testable engine-free
// (investigation_journal pattern).
func MoveToward(current, target matrix.Vec3, speed, dt, epsilon float32) (matrix.Vec3, bool) {
	toTarget := target.Subtract(current)
	toTarget.SetY(0)
	dist := toTarget.Length()
	if dist <= epsilon {
		return current, true
	}
	step := speed * dt
	if step >= dist-epsilon {
		return matrix.NewVec3(target.X(), current.Y(), target.Z()), true
	}
	dir := toTarget.Scale(1.0 / dist)
	return current.Add(dir.Scale(step)), false
}
