// INT-4c journal view: a slice-minimal investigation journal overlay built
// with the proven UI recipe (markup, px-absolute positioning) and fed by the
// investigation journal package's pure display functions (BuildChainList,
// BuildThreadList, BuildRevelationPrompt). The full three-panel journal UI
// (investigation_journal package, imperative construction) needs a markup
// conversion — a production story; this view gives the slice its
// connect-the-dots feedback.
package game

import (
	"strconv"

	"kaijuengine.com/engine/ui"
	"kaijuengine.com/engine/ui/markup"
	"kaijuengine.com/engine/ui/markup/document"

	"marrow-of-empire/entity"
	"marrow-of-empire/investigation_journal"
)

const journalHTML = `<!DOCTYPE html>
<html>
<head>
<style>
body { margin: 0; padding: 0; }
#jrn-root {
    position: absolute;
    top: 60px;
    left: 830px;
    width: 430px;
    height: 580px;
    display: flex;
    flex-direction: column;
    background-color: #F2E6D1;
    z-index: 90;
    padding: 16px 20px;
}
#jrn-title { color: #26140A; font-size: 22px; margin-bottom: 10px; }
.jrn-line { color: #26140A; font-size: 15px; }
#jrn-tab {
    position: absolute;
    top: 660px;
    left: 1220px;
    width: 44px;
    height: 44px;
    background-color: #E6D9C4;
    z-index: 95;
}
</style>
</head>
<body>
<div id="jrn-root">
    <label id="jrn-title">Investigation Journal</label>
    <div id="jrn-lines">
        <label id="jrn-line-0" class="jrn-line">&#8203;</label>
        <label id="jrn-line-1" class="jrn-line">&#8203;</label>
        <label id="jrn-line-2" class="jrn-line">&#8203;</label>
        <label id="jrn-line-3" class="jrn-line">&#8203;</label>
        <label id="jrn-line-4" class="jrn-line">&#8203;</label>
        <label id="jrn-line-5" class="jrn-line">&#8203;</label>
        <label id="jrn-line-6" class="jrn-line">&#8203;</label>
        <label id="jrn-line-7" class="jrn-line">&#8203;</label>
        <label id="jrn-line-8" class="jrn-line">&#8203;</label>
        <label id="jrn-line-9" class="jrn-line">&#8203;</label>
        <label id="jrn-line-10" class="jrn-line">&#8203;</label>
        <label id="jrn-line-11" class="jrn-line">&#8203;</label>
        <label id="jrn-line-12" class="jrn-line">&#8203;</label>
        <label id="jrn-line-13" class="jrn-line">&#8203;</label>
    </div>
</div>
<div id="jrn-tab"></div>
</body>
</html>`

// journalLineCount is the static journal line pool size.
const journalLineCount = 14

// journalView is the slice's minimal investigation journal.
type journalView struct {
	g       *Game
	doc     *document.Document
	visible bool
}

// newJournalView builds the overlay (hidden) and the clickable toggle tab.
func newJournalView(g *Game) *journalView {
	v := &journalView{g: g}
	v.doc = markup.DocumentFromHTMLString(&g.uiMan, journalHTML, "", nil, nil, nil)
	if elm, ok := v.doc.GetElementById("jrn-tab"); ok {
		elm.UI.AddEvent(ui.EventTypeClick, func() { v.Toggle() })
	}
	// Hidden until first Toggle; the tab stays visible.
	for i := range v.doc.Elements {
		if v.doc.Elements[i].Attribute("id") == "jrn-tab" {
			continue
		}
		v.doc.Elements[i].UI.Entity().Deactivate()
	}
	return v
}

// Toggle shows or hides the journal and refreshes it on open. Visibility
// uses entity activation (the engine console's proven mechanism) — the CSS
// display toggle does not re-apply layout in this engine version (INT-4).
func (v *journalView) Toggle() {
	v.visible = !v.visible
	if v.visible {
		v.Refresh()
	}
	for i := range v.doc.Elements {
		if v.doc.Elements[i].Attribute("id") == "jrn-tab" {
			continue // the tab always stays visible
		}
		if v.visible {
			v.doc.Elements[i].UI.Entity().Activate()
		} else {
			v.doc.Elements[i].UI.Entity().Deactivate()
		}
	}
}

// Refresh fills the static line pool from the thread engine's knowledge.
// Text assignment only — runtime tree mutations race the UI update pass in
// this engine version (INT-4 live-run panic).
func (v *journalView) Refresh() {
	pk := v.g.talk.threads.Knowledge()

	var lines []string
	chains := investigation_journal.BuildChainList(&pk)
	if len(chains) == 0 {
		lines = append(lines, "No threads yet. Ask someone a question.")
	}
	for _, chain := range chains {
		lines = append(lines, "◆ "+chain.Summary+" ("+
			investigation_journal.ChainStatusLabel(chain.Status)+")")
		for _, th := range investigation_journal.BuildThreadList(&pk, chain.ID) {
			lines = append(lines, "  → "+th.Topic+" "+th.StatusIcon)
		}
		if prompt := investigation_journal.BuildRevelationPrompt(&pk, chain.ID); prompt != nil {
			target := prompt.TargetName
			if target == "" {
				if name, ok := v.g.talk.Name(entity.GameEntityID(prompt.TargetNPC)); ok {
					target = name
				}
			}
			lines = append(lines, "")
			lines = append(lines, "★ "+prompt.Conclusion)
			if target != "" {
				lines = append(lines, "  Confront: "+target)
			}
		}
	}

	for i := 0; i < journalLineCount; i++ {
		elm, ok := v.doc.GetElementById("jrn-line-" + strconv.Itoa(i))
		if !ok {
			continue
		}
		lbl := elm.InnerLabel()
		if lbl == nil {
			continue
		}
		if i < len(lines) {
			lbl.SetText(lines[i])
		} else {
			lbl.SetText("")
		}
	}
}
