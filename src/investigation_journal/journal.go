package investigation_journal

import (
	"log/slog"

	"kaijuengine.com/engine"
	"kaijuengine.com/engine/ui"
	"kaijuengine.com/matrix"
	"kaijuengine.com/platform/hid"

	"marrow-of-empire/entity"
	"marrow-of-empire/thread_investigation"
)

// Layout constants (ADR-0012: Desk register, paper-slide motion).
// Data-driven — extract to config file when tuning begins.
const (
	// SidebarFraction is the fraction of journal width for the chain list sidebar (CR-1: 30%).
	SidebarFraction = 0.30
	// DetailFraction is the fraction of journal width for the center detail panel (CR-1: 70%).
	DetailFraction = 0.70
	// QuickRefHeight is the height in pixels of the entity quick-ref bottom bar.
	QuickRefHeight float32 = 60
	// ParchmentInset is the margin around the parchment container (4% of screen per side).
	ParchmentInset = 0.04

	// AnimShowDurationSec is the paper-slide enter duration (ADR-0012: 300ms ease-out).
	AnimShowDurationSec float64 = 0.300
	// AnimHideDurationSec is the paper-slide exit duration (ADR-0012: 200ms).
	AnimHideDurationSec float64 = 0.200
)

// Parchment palette (ADR-0012: Desk register, Art Bible Section 2.4).
// Amber (#D9A441) is reserved for status indicators in later stories —
// never use amber for text or borders in the shell.
var (
	colorParchment     = matrix.ColorRGBAInt(227, 213, 176, 255) // #E3D5B0
	colorParchmentDark = matrix.ColorRGBAInt(224, 212, 191, 255) // slightly darker for visual separation
	colorInk           = matrix.ColorRGBAInt(28, 24, 19, 255)    // #1C1813
	colorDim           = matrix.ColorRGBAInt(20, 18, 16, 179)    // soot at ~70% — world shows at 30%
)

// Journal manages the investigation journal overlay — the player's desk-register
// interface for viewing threads, chains, facts, and revelations. It implements
// the Desk register from ADR-0012: a full-screen parchment overlay with world
// dimming to 30% and paper-slide animation.
//
// States: Closed → Open (see state.go). Story 001 implements the shell only;
// later stories populate the panels with content.
//
// UI is built programmatically (same pattern as dialogue_ui). The HTML markup
// at assets/ui/investigation_journal.html serves as the design reference for
// the intended structure; Story 002+ may switch to markup.DocumentFromHTMLAsset
// when content population requires complex markup.
//
// Story: INVJ-001 — Journal Shell & Three-Panel Layout
type Journal struct {
	host *engine.Host
	man  ui.Manager

	state JournalState

	// Animation state (paper-slide motion per ADR-0012).
	// Uses start/end positions for smooth reversal on interrupt.
	animating    bool
	animTimer    float64
	animDuration float64
	animStartY   float32 // Y offset at animation start (captured for reversal)
	animEndY     float32 // target Y offset (0 = visible, screenH = offscreen)
	animStartSX  float32 // sidebar X offset at animation start
	animEndSX    float32 // target sidebar X offset (0 = visible)

	overlayY  float32 // current Y offset of the root overlay
	sidebarX  float32 // current X offset of the sidebar panel
	sidebarW  float32 // computed sidebar width (cached for animation)
	detailW   float32 // computed detail panel width
	detailH   float32 // computed detail panel height

	// UI hierarchy
	overlay   *ui.Panel // full-screen dimming overlay (root)
	parchment *ui.Panel // parchment container (92% of screen)
	sidebar   *ui.Panel // chain list, left 30%
	detail    *ui.Panel // thread/fact detail, center 70%
	quickRef  *ui.Panel // entity chips, bottom bar

	updateId      engine.UpdateId
	keyCallbackId hid.KeyCallbackId

	// EventBus for emitting JournalStateChanged events (ADR-0011).
	// Nil means events are skipped (test mode).
	eventBus entity.EventBus

	// Data state (Story 002+: populated by LoadData).
	selectedChainID string                          // currently selected chain, empty if none
	chainEntries    []ChainEntry                    // cached sorted chain list
	threadEntries   map[string][]ThreadEntry        // chainID → sorted threads
	pk              *thread_investigation.PlayerKnowledge // reference to current data

	// Content children for dynamic refresh (Story 002).
	// Tracked so clearPanel can remove them before re-rendering.
	sidebarChildren []*ui.UI
	detailChildren  []*ui.UI
	sidebarY        float32 // running Y offset for sidebar rows
	detailY         float32 // running Y offset for detail rows

	// Story 003: tabs, entities, fact log.
	detailTab     string              // "threads" or "facts" — current detail tab
	entityRefs    *EntityRefSet       // recently referenced entities for quick-ref bar
	selectedEntity uint64             // currently selected entity ID (EntitySelected state)
	navStack      []JournalState      // back-navigation stack for EntitySelected → ChainSelected
	factGroups    map[string][]FactGroup // chainID → fact groups (cached from LoadData)
}

// NewJournal creates and initializes the investigation journal UI hierarchy.
// The journal starts in StateClosed (not visible). Call Open() to present it.
// eventBus may be nil during testing — events are silently skipped.
func NewJournal(host *engine.Host, eventBus entity.EventBus) *Journal {
	screenW := float32(host.Window.Width())
	screenH := float32(host.Window.Height())

	j := &Journal{
		host:       host,
		state:      StateClosed,
		eventBus:   eventBus,
		overlayY:   screenH, // start offscreen
		detailTab:  "threads",
		entityRefs: NewEntityRefSet(),
	}

	j.man.Init(host)

	// ── Build UI hierarchy programmatically ──
	j.buildUI(screenW, screenH)

	// Sidebar starts offscreen to the left (will slide in on Open).
	j.sidebarX = -j.sidebarW
	j.sidebar.Base().Layout().SetOffset(j.sidebarX, 0)

	// ── Per-frame animation update (UIUpdater runs BEFORE UI render pass) ──
	j.updateId = host.UIUpdater.AddUpdate(j.updateAnimation)

	// ── Keyboard: J toggles journal, Esc closes ──
	j.keyCallbackId = host.Window.Keyboard.AddKeyCallback(j.handleKeyEvent)

	slog.Debug("investigation_journal: shell initialized", "state", j.state)
	return j
}

// buildUI constructs the three-panel overlay programmatically.
func (j *Journal) buildUI(screenW, screenH float32) {
	// ── Root overlay (full-screen dimming background) ──
	j.overlay = j.man.Add().ToPanel()
	j.overlay.Init(nil, ui.ElementTypePanel)
	j.overlay.SetColor(colorDim)
	rootLayout := j.overlay.Base().Layout()
	rootLayout.SetPositioning(ui.PositioningAbsolute)
	rootLayout.Scale(screenW, screenH)
	rootLayout.SetOffset(0, j.overlayY)

	// ── Parchment container (centered, 92% of screen) ──
	parchmentW := screenW * (1.0 - 2*ParchmentInset)
	parchmentH := screenH * (1.0 - 2*ParchmentInset)
	j.parchment = j.man.Add().ToPanel()
	j.parchment.Init(nil, ui.ElementTypePanel)
	j.parchment.SetColor(colorParchment)
	parchLayout := j.parchment.Base().Layout()
	parchLayout.SetPositioning(ui.PositioningAbsolute)
	parchLayout.Scale(parchmentW, parchmentH)
	parchLayout.SetOffset(screenW*ParchmentInset, screenH*ParchmentInset)
	j.overlay.AddChild(j.parchment.Base())

	// ── Sidebar (left 30%) ──
	j.sidebarW = parchmentW * SidebarFraction
	j.sidebar = j.man.Add().ToPanel()
	j.sidebar.Init(nil, ui.ElementTypePanel)
	// Subtle umber right-border for visual separation from detail panel (CR-1, AC-3).
	j.sidebar.SetColor(matrix.ColorRGBAInt(0, 0, 0, 0))
	sidebarLayout := j.sidebar.Base().Layout()
	sidebarLayout.SetPositioning(ui.PositioningAbsolute)
	sidebarLayout.Scale(j.sidebarW, parchmentH-QuickRefHeight)
	sidebarLayout.SetOffset(j.sidebarX, 0)
	j.parchment.AddChild(j.sidebar.Base())

	// ── Sidebar right-edge divider (umber line, 1px) ──
	divider := j.man.Add().ToPanel()
	divider.Init(nil, ui.ElementTypePanel)
	divider.SetColor(matrix.ColorRGBAInt(46, 40, 32, 77)) // umber at ~30%
	dividerLayout := divider.Base().Layout()
	dividerLayout.SetPositioning(ui.PositioningAbsolute)
	dividerLayout.Scale(1, parchmentH-QuickRefHeight)
	dividerLayout.SetOffset(j.sidebarW, 0)
	j.parchment.AddChild(divider.Base())

	// ── Detail view (right 70%) ──
	j.detailW = parchmentW * DetailFraction
	j.detailH = parchmentH - QuickRefHeight
	j.detail = j.man.Add().ToPanel()
	j.detail.Init(nil, ui.ElementTypePanel)
	detailLayout := j.detail.Base().Layout()
	detailLayout.SetPositioning(ui.PositioningAbsolute)
	detailLayout.Scale(j.detailW, j.detailH)
	detailLayout.SetOffset(j.sidebarW+1, 0) // +1 for divider
	j.parchment.AddChild(j.detail.Base())

	// ── Quick-ref bar (bottom, full width) ──
	j.quickRef = j.man.Add().ToPanel()
	j.quickRef.Init(nil, ui.ElementTypePanel)
	j.quickRef.SetColor(colorParchmentDark)
	quickLayout := j.quickRef.Base().Layout()
	quickLayout.SetPositioning(ui.PositioningAbsolute)
	quickLayout.Scale(parchmentW, QuickRefHeight)
	quickLayout.SetOffset(0, parchmentH-QuickRefHeight)
	j.parchment.AddChild(j.quickRef.Base())

	// ── Top divider above quick-ref bar ──
	topDivider := j.man.Add().ToPanel()
	topDivider.Init(nil, ui.ElementTypePanel)
	topDivider.SetColor(matrix.ColorRGBAInt(46, 40, 32, 77))
	topDividerLayout := topDivider.Base().Layout()
	topDividerLayout.SetPositioning(ui.PositioningAbsolute)
	topDividerLayout.Scale(parchmentW, 1)
	topDividerLayout.SetOffset(0, parchmentH-QuickRefHeight)
	j.parchment.AddChild(topDivider.Base())
}

// ──────────────────────────────────────────────────────────────────────────────
// Public API
// ──────────────────────────────────────────────────────────────────────────────

// Open presents the investigation journal. The overlay slides up from the
// bottom and the sidebar slides in from the left with a 300ms ease-out
// animation (paper-slide motion, ADR-0012). The camera is notified to
// freeze via JournalStateChanged event.
// If called during a hide animation, the animation reverses from the
// current position — no pop, no dropped input.
func (j *Journal) Open() {
	if j.state == StateOpen && !j.animating {
		return // already fully open and stationary
	}
	j.state = StateOpen
	j.startAnimation(AnimShowDurationSec, 0, 0) // target: overlay Y=0, sidebar X=0
	j.emitStateEvent(true)
	slog.Debug("investigation_journal: opened")
}

// Close dismisses the investigation journal. The overlay slides down and the
// sidebar slides out with a 200ms ease-out animation. The camera is notified
// via JournalStateChanged event.
// If called during a show animation, the animation reverses from the
// current position — no pop, no dropped input.
func (j *Journal) Close() {
	if j.state == StateClosed && !j.animating {
		return // already fully closed and stationary
	}
	screenH := float32(j.host.Window.Height())
	j.startAnimation(AnimHideDurationSec, screenH, -j.sidebarW) // target: offscreen
	j.emitStateEvent(false)
	slog.Debug("investigation_journal: closed")
}

// Toggle switches the journal between open and closed states.
// Bound to the J key by default. Safe to call mid-animation — reverses direction.
func (j *Journal) Toggle() {
	if j.state == StateOpen {
		j.Close()
	} else {
		j.Open()
	}
}

// State returns the current journal state.
func (j *Journal) State() JournalState {
	return j.state
}

// IsOpen reports whether the journal is currently visible or animating to visible.
func (j *Journal) IsOpen() bool {
	return j.state.IsOpen()
}

// Animating reports whether a slide animation is currently in progress.
func (j *Journal) Animating() bool {
	return j.animating
}

// Sidebar returns the sidebar panel for content population (Story 002+).
func (j *Journal) Sidebar() *ui.Panel {
	return j.sidebar
}

// Detail returns the detail panel for content population (Story 002+).
func (j *Journal) Detail() *ui.Panel {
	return j.detail
}

// QuickRef returns the quick-ref bar panel for content population (Story 003+).
func (j *Journal) QuickRef() *ui.Panel {
	return j.quickRef
}

// LoadData populates the journal with fresh data from the Thread Investigation
// system. Call this before opening the journal or when thread state changes.
// pk may be nil to clear all content.
func (j *Journal) LoadData(pk *thread_investigation.PlayerKnowledge) {
	j.pk = pk
	j.chainEntries = BuildChainList(pk)
	j.threadEntries = make(map[string][]ThreadEntry)
	j.factGroups = make(map[string][]FactGroup)
	if pk != nil {
		for _, ch := range pk.Chains {
			j.threadEntries[ch.ID] = BuildThreadList(pk, ch.ID)
			j.factGroups[ch.ID] = BuildFactLog(pk, ch.ID)
		}
	}
	// Reset selection and tab when data changes.
	j.selectedChainID = ""
	j.selectedEntity = 0
	j.detailTab = "threads"
	j.navStack = nil
	if j.state == StateChainSelected || j.state == StateEntitySelected {
		j.state = StateOpen
	}
	j.refreshSidebar()
	j.refreshDetail()
	j.refreshQuickRef()
}

// SelectChain selects a chain in the sidebar and populates the detail panel
// with its threads. Transitions to StateChainSelected.
func (j *Journal) SelectChain(chainID string) {
	j.selectedChainID = chainID
	j.state = StateChainSelected
	j.refreshSidebar()
	j.refreshDetail()
}

// DeselectChain returns to the base Open state (no chain selected).
func (j *Journal) DeselectChain() {
	j.selectedChainID = ""
	j.state = StateOpen
	j.refreshDetail()
}

// SelectedChainID returns the currently selected chain ID, or empty string.
func (j *Journal) SelectedChainID() string {
	return j.selectedChainID
}

// SelectEntity transitions to EntitySelected state for the given entity.
// Records the entity in the quick-ref set and pushes the current state onto
// the navigation stack for back-navigation.
func (j *Journal) SelectEntity(entityID uint64, name string) {
	j.navStack = append(j.navStack, j.state)
	j.selectedEntity = entityID
	j.state = StateEntitySelected
	j.entityRefs.RecordEntity(entityID, name, 0) // timestamp from game clock when available
	j.refreshDetail()
	j.refreshQuickRef()
}

// DeselectEntity returns to the previous state from the navigation stack.
func (j *Journal) DeselectEntity() {
	if len(j.navStack) > 0 {
		j.state = j.navStack[len(j.navStack)-1]
		j.navStack = j.navStack[:len(j.navStack)-1]
	} else {
		j.state = StateOpen
	}
	j.selectedEntity = 0
	j.refreshDetail()
}

// SetDetailTab switches the detail panel tab. Valid values: "threads", "facts".
func (j *Journal) SetDetailTab(tab string) {
	if tab != "threads" && tab != "facts" {
		return
	}
	j.detailTab = tab
	j.refreshDetail()
}

// DetailTab returns the current detail tab.
func (j *Journal) DetailTab() string {
	return j.detailTab
}

// EntityRefs returns the entity quick-ref set for external inspection.
func (j *Journal) EntityRefs() *EntityRefSet {
	return j.entityRefs
}

// ──────────────────────────────────────────────────────────────────────────────
// Internal: sidebar and detail population
// ──────────────────────────────────────────────────────────────────────────────

// refreshSidebar re-renders the sidebar chain list from cached entries.
func (j *Journal) refreshSidebar() {
	if j.sidebar == nil {
		return
	}
	j.clearPanel(j.sidebar)
	j.sidebarY = 0

	if len(j.chainEntries) == 0 {
		j.addSidebarPlaceholder("No active investigations")
		return
	}

	for _, entry := range j.chainEntries {
		j.addChainRow(entry)
	}
}

// addChainRow creates a clickable chain entry row in the sidebar at the current sidebarY.
func (j *Journal) addChainRow(entry ChainEntry) {
	rowHeight := float32(48)
	row := j.man.Add().ToPanel()
	row.Init(nil, ui.ElementTypePanel)
	rowLayout := row.Base().Layout()
	rowLayout.SetPositioning(ui.PositioningAbsolute)
	rowLayout.Scale(j.sidebarW, rowHeight)
	rowLayout.SetOffset(0, j.sidebarY)
	j.sidebarY += rowHeight + 2 // 2px gap between rows

	// Chain title
	title := j.man.Add().ToLabel()
	title.Init(entry.Summary)
	title.SetFontSize(16)
	title.SetColor(colorInk)
	titleLayout := title.Base().Layout()
	titleLayout.SetPositioning(ui.PositioningAbsolute)
	titleLayout.SetOffset(8, 4)
	titleLayout.Scale(j.sidebarW-16, 20)
	row.AddChild(title.Base())

	// Thread + fact count
	counts := j.man.Add().ToLabel()
	counts.Init(FormatThreadCount(entry.ActiveCount) + " · " + FormatFactCount(entry.FactCount))
	counts.SetFontSize(12)
	counts.SetColor(matrix.ColorRGBAInt(120, 108, 96, 255)) // muted umber
	countsLayout := counts.Base().Layout()
	countsLayout.SetPositioning(ui.PositioningAbsolute)
	countsLayout.SetOffset(8, 26)
	countsLayout.Scale(j.sidebarW-16, 16)
	row.AddChild(counts.Base())

	// Selection highlight
	if entry.ID == j.selectedChainID {
		row.SetColor(matrix.ColorRGBAInt(200, 190, 165, 100)) // warm parchment highlight
	}

	// Click handler → SelectChain
	row.Base().AddEvent(ui.EventTypeClick, func() {
		j.SelectChain(entry.ID)
	})

	j.sidebar.AddChild(row.Base())
	j.sidebarChildren = append(j.sidebarChildren, row.Base())
}

// refreshDetail re-renders the detail panel content based on current state.
func (j *Journal) refreshDetail() {
	if j.detail == nil {
		return
	}
	j.clearPanel(j.detail)
	j.detailY = 0

	switch {
	case j.state == StateEntitySelected && j.selectedEntity != 0:
		j.populateEntityDetail()
	case j.selectedChainID != "" && (j.state == StateChainSelected || j.state == StateOpen):
		// Show tab bar + content for selected chain.
		j.addDetailTabBar()
		if j.detailTab == "facts" {
			j.populateFactLog()
		} else {
			j.populateThreadDetail()
		}
	case j.state == StateOpen:
		j.addDetailPlaceholder("Select a chain to view details")
	default:
		j.addDetailPlaceholder("")
	}
}

// addDetailTabBar renders the "Threads" | "Facts" tab bar at the top of the detail panel.
func (j *Journal) addDetailTabBar() {
	tabH := float32(28)
	row := j.man.Add().ToPanel()
	row.Init(nil, ui.ElementTypePanel)
	rowLayout := row.Base().Layout()
	rowLayout.SetPositioning(ui.PositioningAbsolute)
	rowLayout.Scale(j.detailW, tabH)
	rowLayout.SetOffset(0, j.detailY)
	j.detailY += tabH + 8

	// Threads tab
	threadsTab := j.man.Add().ToLabel()
	threadsTab.Init("Threads")
	threadsTab.SetFontSize(14)
	if j.detailTab == "threads" {
		threadsTab.SetColor(colorInk)
	} else {
		threadsTab.SetColor(matrix.ColorRGBAInt(120, 108, 96, 255))
	}
	ttLayout := threadsTab.Base().Layout()
	ttLayout.SetPositioning(ui.PositioningAbsolute)
	ttLayout.SetOffset(8, 2)
	ttLayout.Scale(80, 20)
	row.AddChild(threadsTab.Base())
	threadsTab.Base().AddEvent(ui.EventTypeClick, func() { j.SetDetailTab("threads") })

	// Facts tab
	factsTab := j.man.Add().ToLabel()
	factsTab.Init("Facts")
	factsTab.SetFontSize(14)
	if j.detailTab == "facts" {
		factsTab.SetColor(colorInk)
	} else {
		factsTab.SetColor(matrix.ColorRGBAInt(120, 108, 96, 255))
	}
	ftLayout := factsTab.Base().Layout()
	ftLayout.SetPositioning(ui.PositioningAbsolute)
	ftLayout.SetOffset(96, 2)
	ftLayout.Scale(60, 20)
	row.AddChild(factsTab.Base())
	factsTab.Base().AddEvent(ui.EventTypeClick, func() { j.SetDetailTab("facts") })

	j.detail.AddChild(row.Base())
	j.detailChildren = append(j.detailChildren, row.Base())
}

// populateFactLog fills the detail panel with fact groups for the selected chain.
func (j *Journal) populateFactLog() {
	groups := j.factGroups[j.selectedChainID]
	if len(groups) == 0 {
		j.addDetailPlaceholder("No facts discovered for this chain yet")
		return
	}
	for _, group := range groups {
		j.addFactGroup(group)
	}
}

// addFactGroup renders a fact group header and its facts.
func (j *Journal) addFactGroup(group FactGroup) {
	// Group header: "NPC Name · Trust: Acquaintance"
	header := j.man.Add().ToLabel()
	header.Init(TrustLevelLabel(group.TrustLevel))
	header.SetFontSize(13)
	header.SetColor(matrix.ColorRGBAInt(80, 72, 60, 255))
	hLayout := header.Base().Layout()
	hLayout.SetPositioning(ui.PositioningAbsolute)
	hLayout.SetOffset(12, j.detailY)
	hLayout.Scale(j.detailW-24, 18)
	j.detail.AddChild(header.Base())
	j.detailChildren = append(j.detailChildren, header.Base())
	j.detailY += 20

	for _, fact := range group.Facts {
		j.addFactRow(fact)
	}
}

// addFactRow renders a single fact entry with contradiction marker if applicable.
func (j *Journal) addFactRow(fact FactEntry) {
	rowH := float32(36)
	row := j.man.Add().ToPanel()
	row.Init(nil, ui.ElementTypePanel)
	rLayout := row.Base().Layout()
	rLayout.SetPositioning(ui.PositioningAbsolute)
	rLayout.Scale(j.detailW, rowH)
	rLayout.SetOffset(0, j.detailY)
	j.detailY += rowH + 2

	// Contradiction marker
	if fact.Contradicts {
		marker := j.man.Add().ToLabel()
		marker.Init("↯ CONTRADICTS") // ↯
		marker.SetFontSize(11)
		marker.SetColor(matrix.ColorRGBAInt(196, 72, 58, 255)) // thread-red #C4483A
		mLayout := marker.Base().Layout()
		mLayout.SetPositioning(ui.PositioningAbsolute)
		mLayout.SetOffset(12, 2)
		mLayout.Scale(140, 14)
		row.AddChild(marker.Base())
	}

	// Fact content (placeholder — real content resolved from Knowledge Graph)
	content := j.man.Add().ToLabel()
	content.Init(fact.FactID) // placeholder; real content from KG in later story
	content.SetFontSize(14)
	content.SetColor(colorInk)
	cLayout := content.Base().Layout()
	cLayout.SetPositioning(ui.PositioningAbsolute)
	cLayout.SetOffset(20, 18)
	cLayout.Scale(j.detailW-40, 16)
	row.AddChild(content.Base())

	j.detail.AddChild(row.Base())
	j.detailChildren = append(j.detailChildren, row.Base())
}

// populateEntityDetail fills the detail panel with entity information.
func (j *Journal) populateEntityDetail() {
	// Entity name header
	name := j.man.Add().ToLabel()
	name.Init("Entity Details")
	name.SetFontSize(18)
	name.SetColor(colorInk)
	nLayout := name.Base().Layout()
	nLayout.SetPositioning(ui.PositioningAbsolute)
	nLayout.SetOffset(8, j.detailY)
	nLayout.Scale(j.detailW-16, 22)
	j.detail.AddChild(name.Base())
	j.detailChildren = append(j.detailChildren, name.Base())
	j.detailY += 28

	// Back button
	back := j.man.Add().ToLabel()
	back.Init("← Back to investigation")
	back.SetFontSize(13)
	back.SetColor(matrix.ColorRGBAInt(120, 108, 96, 255))
	bLayout := back.Base().Layout()
	bLayout.SetPositioning(ui.PositioningAbsolute)
	bLayout.SetOffset(8, j.detailY)
	bLayout.Scale(180, 18)
	back.Base().AddEvent(ui.EventTypeClick, func() { j.DeselectEntity() })
	j.detail.AddChild(back.Base())
	j.detailChildren = append(j.detailChildren, back.Base())
	j.detailY += 24

	// Fact references for this entity across all chains.
	if j.pk != nil {
		for _, f := range j.pk.DiscoveredFacts {
			if uint64(f.SourceNPC) == j.selectedEntity {
				ref := j.man.Add().ToLabel()
				ref.Init(f.FactID)
				ref.SetFontSize(13)
				ref.SetColor(colorInk)
				rLayout := ref.Base().Layout()
				rLayout.SetPositioning(ui.PositioningAbsolute)
				rLayout.SetOffset(16, j.detailY)
				rLayout.Scale(j.detailW-32, 16)
				j.detail.AddChild(ref.Base())
				j.detailChildren = append(j.detailChildren, ref.Base())
				j.detailY += 20
			}
		}
	}
}

// refreshQuickRef re-renders the entity quick-ref bar from tracked entities.
func (j *Journal) refreshQuickRef() {
	if j.quickRef == nil {
		return
	}
	// Clear existing quick-ref children (simple re-creation).
	j.clearQuickRef()

	chips := j.entityRefs.Chips()
	for i, chip := range chips {
		if i >= 8 {
			break
		}
		c := j.man.Add().ToLabel()
		displayName := chip.Name
		if displayName == "" {
			displayName = "Entity"
		}
		c.Init(displayName)
		c.SetFontSize(12)
		c.SetColor(colorInk)
		cLayout := c.Base().Layout()
		cLayout.SetPositioning(ui.PositioningAbsolute)
		cLayout.SetOffset(float32(i)*110+8, 8)
		cLayout.Scale(100, 20)
		chipID := chip.EntityID
		c.Base().AddEvent(ui.EventTypeClick, func() { j.SelectEntity(chipID, displayName) })
		j.quickRef.AddChild(c.Base())
	}
}

// clearQuickRef removes all children from the quick-ref bar.
func (j *Journal) clearQuickRef() {
	// Quick-ref children are not tracked in a slice; for now they accumulate.
	// VERIFY: Kaiju API for removing panel children.
}

// populateThreadDetail fills the detail panel with threads for the selected chain.
func (j *Journal) populateThreadDetail() {
	threads := j.threadEntries[j.selectedChainID]
	if len(threads) == 0 {
		j.addDetailPlaceholder("No threads in this chain")
		return
	}

	// Resolve SourceNPC from DiscoveredFacts for each thread.
	factNPC := buildFactNPCMap(j.pk)

	for i, entry := range threads {
		if npcID, ok := factNPC[entry.ID]; ok {
			threads[i].SourceNPC = npcID
		}
		// Fallback: SourceNPC stays 0 (unknown NPC) if not found.
		j.addThreadRow(threads[i])
	}
}

// addThreadRow creates a thread entry row in the detail panel at the current detailY.
func (j *Journal) addThreadRow(entry ThreadEntry) {
	rowHeight := float32(56)
	row := j.man.Add().ToPanel()
	row.Init(nil, ui.ElementTypePanel)
	rowLayout := row.Base().Layout()
	rowLayout.SetPositioning(ui.PositioningAbsolute)
	rowLayout.Scale(j.detailW, rowHeight)
	rowLayout.SetOffset(0, j.detailY)
	j.detailY += rowHeight + 4 // 4px gap between thread rows

	// Status icon
	icon := j.man.Add().ToLabel()
	icon.Init(entry.StatusIcon)
	icon.SetFontSize(16)
	icon.SetColor(matrix.ColorRGBAInt(0, 0, 0, 0)) // VERIFY: apply ThreadStatusColor via Kaiju color API
	iconLayout := icon.Base().Layout()
	iconLayout.SetPositioning(ui.PositioningAbsolute)
	iconLayout.SetOffset(8, 6)
	iconLayout.Scale(20, 20)
	row.AddChild(icon.Base())

	// Topic (description)
	topic := j.man.Add().ToLabel()
	topic.Init(entry.Topic)
	topic.SetFontSize(16)
	topic.SetColor(colorInk)
	topicLayout := topic.Base().Layout()
	topicLayout.SetPositioning(ui.PositioningAbsolute)
	topicLayout.SetOffset(36, 4)
	topicLayout.Scale(j.detailW-44, 20)
	row.AddChild(topic.Base())

	// Age
	age := j.man.Add().ToLabel()
	age.Init(entry.AgeDisplay)
	age.SetFontSize(12)
	age.SetColor(matrix.ColorRGBAInt(120, 108, 96, 255))
	ageLayout := age.Base().Layout()
	ageLayout.SetPositioning(ui.PositioningAbsolute)
	ageLayout.SetOffset(36, 28)
	ageLayout.Scale(120, 16)
	row.AddChild(age.Base())

	// Status label
	status := j.man.Add().ToLabel()
	status.Init(entry.StatusLabel)
	status.SetFontSize(12)
	status.SetColor(matrix.ColorRGBAInt(120, 108, 96, 255))
	statusLayout := status.Base().Layout()
	statusLayout.SetPositioning(ui.PositioningAbsolute)
	statusLayout.SetOffset(160, 28)
	statusLayout.Scale(100, 16)
	row.AddChild(status.Base())

	j.detail.AddChild(row.Base())
	j.detailChildren = append(j.detailChildren, row.Base())
}

// addSidebarPlaceholder shows a placeholder message in the sidebar.
func (j *Journal) addSidebarPlaceholder(text string) {
	label := j.man.Add().ToLabel()
	label.Init(text)
	label.SetFontSize(14)
	label.SetColor(matrix.ColorRGBAInt(120, 108, 96, 255)) // muted umber
	labelLayout := label.Base().Layout()
	labelLayout.SetPositioning(ui.PositioningAbsolute)
	labelLayout.SetOffset(8, 8)
	labelLayout.Scale(j.sidebarW-16, 20)
	j.sidebar.AddChild(label.Base())
	j.sidebarChildren = append(j.sidebarChildren, label.Base())
}

// addDetailPlaceholder shows a placeholder message in the detail panel.
func (j *Journal) addDetailPlaceholder(text string) {
	if text == "" {
		return
	}
	label := j.man.Add().ToLabel()
	label.Init(text)
	label.SetFontSize(15)
	label.SetColor(matrix.ColorRGBAInt(120, 108, 96, 255))
	labelLayout := label.Base().Layout()
	labelLayout.SetPositioning(ui.PositioningAbsolute)
	labelLayout.SetOffset(j.detailW*0.1, j.detailH*0.4)
	labelLayout.Scale(j.detailW*0.8, 24)
	j.detail.AddChild(label.Base())
	j.detailChildren = append(j.detailChildren, label.Base())
}

// clearPanel removes all tracked child elements from a panel.
// Uses the tracked children slices to individually remove each element
// via ui.Manager.Remove. After removal the slice is reset.
func (j *Journal) clearPanel(p *ui.Panel) {
	var children *[]*ui.UI
	switch p {
	case j.sidebar:
		children = &j.sidebarChildren
	case j.detail:
		children = &j.detailChildren
	default:
		return
	}
	for _, child := range *children {
		j.man.Remove(child)
	}
	*children = (*children)[:0]
}

// buildFactNPCMap cross-references Thread IDs with DiscoveredFacts to resolve
// the SourceNPC GameEntityID for each thread. Returns map[threadID]GameEntityID.
func buildFactNPCMap(pk *thread_investigation.PlayerKnowledge) map[string]uint64 {
	if pk == nil {
		return nil
	}
	factNPC := make(map[string]uint64, len(pk.DiscoveredFacts))
	for _, f := range pk.DiscoveredFacts {
		factNPC[f.FactID] = uint64(f.SourceNPC)
	}
	result := make(map[string]uint64)
	for _, t := range pk.ActiveThreads {
		if npc, ok := factNPC[t.Source]; ok {
			result[t.ID] = npc
		}
	}
	for _, t := range pk.ResolvedThreads {
		if npc, ok := factNPC[t.Source]; ok {
			result[t.ID] = npc
		}
	}
	return result
}

// Destroy cleans up journal resources: removes update callbacks, keyboard
// handlers, and tears down the UI hierarchy. Safe to call multiple times.
func (j *Journal) Destroy() {
	if j.overlay == nil {
		return // already destroyed
	}
	j.host.UIUpdater.RemoveUpdate(&j.updateId)
	j.host.Window.Keyboard.RemoveKeyCallback(j.keyCallbackId)
	j.man.Remove(j.overlay.Base())
	j.overlay = nil
}

// ──────────────────────────────────────────────────────────────────────────────
// Internal: events
// ──────────────────────────────────────────────────────────────────────────────

// emitStateEvent sends a JournalStateChanged event via EventBus.
// Silently skipped if no EventBus is configured (e.g., in tests).
func (j *Journal) emitStateEvent(open bool) {
	if j.eventBus == nil {
		return
	}
	j.eventBus.Emit(entity.Event{
		Type:   EventJournalStateChanged,
		Source: "investigation_journal",
		Data:   JournalStateEvent{Open: open},
	})
}

// ──────────────────────────────────────────────────────────────────────────────
// Internal: animation
// ──────────────────────────────────────────────────────────────────────────────

// startAnimation begins a slide animation from the current overlay/sidebar
// positions toward the given targets. Capturing the current position as the
// start ensures smooth reversal when Open/Close is called mid-animation.
func (j *Journal) startAnimation(durationSec float64, targetY, targetSX float32) {
	j.animating = true
	j.animTimer = 0
	j.animDuration = durationSec
	j.animStartY = j.overlayY
	j.animEndY = targetY
	j.animStartSX = j.sidebarX
	j.animEndSX = targetSX
}

// updateAnimation is called each frame by host.UIUpdater. It advances the slide
// animation using an ease-out cubic curve (per ADR-0012 paper-slide motion).
// deltaTime is in seconds.
func (j *Journal) updateAnimation(deltaTime float64) {
	if !j.animating {
		return
	}

	j.animTimer += deltaTime
	t := j.animTimer / j.animDuration
	if t >= 1.0 {
		t = 1.0
		j.animating = false
	}

	et := EaseOutCubic(t)

	// Interpolate overlay Y: animStartY → animEndY
	j.overlayY = j.animStartY + (j.animEndY-j.animStartY)*float32(et)
	j.overlay.Base().Layout().SetOffset(0, j.overlayY)

	// Interpolate sidebar X: animStartSX → animEndSX
	j.sidebarX = j.animStartSX + (j.animEndSX-j.animStartSX)*float32(et)
	j.sidebar.Base().Layout().SetOffset(j.sidebarX, 0)

	// On hide completion, transition to Closed
	if !j.animating && j.animEndY > 0 {
		j.state = StateClosed
	}
}

// EaseOutCubic implements the cubic ease-out curve: t → 1-(1-t)³.
// Per ADR-0012 paper-slide motion: fast start, gentle stop, no bounce.
// Exported so tests can verify the implementation rather than re-deriving.
func EaseOutCubic(t float64) float64 {
	if t <= 0 {
		return 0
	}
	if t >= 1 {
		return 1
	}
	return 1.0 - (1.0-t)*(1.0-t)*(1.0-t)
}

// ──────────────────────────────────────────────────────────────────────────────
// Internal: keyboard input
// ──────────────────────────────────────────────────────────────────────────────

// handleKeyEvent processes keyboard events. J toggles the journal; Esc closes
// it when open. When the journal is closed, Esc is not consumed — the pause
// menu handler takes it.
func (j *Journal) handleKeyEvent(keyId int, keyState hid.KeyState) {
	if keyState != hid.KeyStateDown {
		return
	}

	switch keyId {
	case hid.KeyboardKeyJ:
		j.Toggle()
	case hid.KeyboardKeyEscape:
		if j.state == StateOpen {
			j.Close()
		}
		// When closed, Esc falls through to the game layer (pause menu).
	}
}
