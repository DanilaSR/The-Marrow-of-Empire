// Package game provides the GameInterface implementation that bootstraps
// The Marrow of Empire within the Kaiju Engine.
package game

import (
	"reflect"

	"kaijuengine.com/bootstrap"
	"kaijuengine.com/engine"
	"kaijuengine.com/engine/assets"
	"kaijuengine.com/engine/ui"
	"kaijuengine.com/engine/ui/markup/document"

	"marrow-of-empire/camera"
)

// Game implements bootstrap.GameInterface for Marrow of Empire.
// It owns the top-level UI manager and orchestrates screen transitions.
type Game struct {
	host     *engine.Host
	uiMan    ui.Manager
	titleDoc *document.Document // slice-era title; nil (destroyed) in production boot — retained so world.go's destroy no-ops
	menu     *menuWiring        // production main-menu flow (Story 008)

	// INT-1 game world state (nil until New Game is pressed).
	world         *worldState
	iso           *camera.Isometric
	worldUpdateID engine.UpdateId

	// INT-2 talk system (dialogue pipeline, NPCs, thread engine).
	talk *talkSystem

	// INT-3 combat system (mine encounter).
	combat *combatSystem

	// INT-4c slice journal view (J key / tab click).
	journal *journalView
}

// PluginRegistry returns types to expose to the Lua scripting layer.
// Marrow of Empire uses Go for all game logic; no Lua types are exposed.
func (g *Game) PluginRegistry() []reflect.Type {
	return nil
}

// ContentDatabase returns the game content database rooted at "game_content/"
// wrapped with the standard Kaiju extension-based path mapping.
func (g *Game) ContentDatabase() (assets.Database, error) {
	base, err := assets.NewFileDatabase("game_content")
	if err != nil {
		return nil, err
	}
	return &contentDB{base: base}, nil
}

// Launch is the game entry point called by the engine after initialisation.
// It builds the production main-menu flow (Story 008: the game boots into the
// production Title screen, AC-1) and registers per-frame updates.
func (g *Game) Launch(host *engine.Host) {
	g.host = host
	g.uiMan.Init(host)

	// Production flow: Title/Settings/Keybindings/PauseMenu + settings.cfg
	// adapter (AC-1/AC-2). The slice's buildTitleScreen is retired.
	g.menu = newMenuWiring(g)

	// Persist debounced settings on clean shutdown (AC-2).
	host.OnClose.Add(func() { g.menu.Flush() })

	// Keep the isometric camera extents in sync with the window (ADR-0003).
	host.Window.OnResize.Add(func() {
		if g.iso != nil {
			g.iso.Resize(float32(host.Window.Width()), float32(host.Window.Height()))
		}
	})

	_ = host.Updater.AddUpdate(g.update)
}

// update runs each frame. deltaTime is in seconds.
func (g *Game) update(deltaTime float64) {
	if g.menu != nil {
		g.menu.update()
	}
}

// GetGame returns the game instance for the engine's getGame() entry point.
func GetGame() bootstrap.GameInterface {
	return &Game{}
}
