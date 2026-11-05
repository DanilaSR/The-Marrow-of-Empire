package game

import (
	"log/slog"

	"kaijuengine.com/engine/ui"
	"kaijuengine.com/engine/ui/markup"
)

// titleScreenHTML is the main menu markup in the desk register (art bible §6):
// soot backdrop, bone-ivory title, ink buttons with amber hover accents.
const titleScreenHTML = `<!DOCTYPE html>
<html>
<head>
<style>
	body {
		margin: 0;
		padding: 0;
		background-color: #141210;
		display: flex;
		flex-direction: column;
		align-items: center;
	}
	#title-panel {
		margin-top: 14%;
		display: flex;
		flex-direction: column;
		align-items: center;
	}
	#game-title {
		color: #E9E6DC;
		font-size: 36px;
	}
	#game-subtitle {
		color: #988E84;
		font-size: 18px;
		margin-top: 8px;
		margin-bottom: 40px;
	}
	.menu-btn {
		width: 340px;
		height: 44px;
		margin: 6px;
		background-color: #1C1813;
		color: #E9E6DC;
		font-size: 16px;
	}
	.menu-btn:hover {
		background-color: #2A231C;
	}
</style>
</head>
<body>
	<div id="title-panel">
		<div id="game-title">The Marrow of Empire</div>
		<div id="game-subtitle">An Isometric Investigation</div>
		<button id="btn-new" class="menu-btn">New Game</button>
		<button id="btn-load" class="menu-btn">Load Game</button>
		<button id="btn-settings" class="menu-btn">Settings</button>
		<button id="btn-exit" class="menu-btn">Exit</button>
	</div>
</body>
</html>`

// buildTitleScreen renders the main menu via the Kaiju markup system.
// The markup path is proven (console.html pattern) and handles root sizing,
// flex layout, and CSS mapping to engine layout calls.
func (g *Game) buildTitleScreen() {
	g.titleDoc = markup.DocumentFromHTMLString(&g.uiMan, titleScreenHTML, "", nil, nil, nil)

	if elm, ok := g.titleDoc.GetElementById("btn-exit"); ok {
		elm.UI.AddEvent(ui.EventTypeClick, func() {
			g.host.Close()
		})
	}
	if elm, ok := g.titleDoc.GetElementById("btn-new"); ok {
		elm.UI.AddEvent(ui.EventTypeClick, func() {
			slog.Info("New Game pressed — entering the game world")
			g.startNewGame()
		})
	}
	if elm, ok := g.titleDoc.GetElementById("btn-load"); ok {
		elm.UI.AddEvent(ui.EventTypeClick, func() {
			slog.Info("Load Game pressed — save/load screens not yet wired")
		})
	}
	if elm, ok := g.titleDoc.GetElementById("btn-settings"); ok {
		elm.UI.AddEvent(ui.EventTypeClick, func() {
			slog.Info("Settings pressed — settings screen not yet wired")
		})
	}
}
