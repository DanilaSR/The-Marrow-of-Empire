// INT-2 talk loop: NPC registry, dialogue entry, and the
// question → topic match → fact query → prompt → LLM → response → thread
// pipeline. This is the first wiring of the AI dialogue, question system,
// knowledge graph, and thread investigation packages into the running game.
package game

import (
	"context"
	"log/slog"
	"math"
	"strconv"
	"strings"
	"time"

	"kaijuengine.com/engine"
	"kaijuengine.com/matrix"

	aidialogue "marrow-of-empire/ai_dialogue"
	"marrow-of-empire/dialogue_ui"
	"marrow-of-empire/entity"
	knowledge "marrow-of-empire/npc_knowledge_graph"
	questions "marrow-of-empire/question_system"
	threads "marrow-of-empire/thread_investigation"
)

// Slice entity IDs (hand-assigned, matched by the .kg authoring under
// assets/npc_knowledge/borderland/). The entity framework's ID allocator
// takes over once EF-wrapped entities land in the game layer.
const (
	entBlacksmith  entity.GameEntityID = 1001
	entMiner       entity.GameEntityID = 1002
	entScholar     entity.GameEntityID = 1003
	entOldMine     entity.GameEntityID = 2001
	entInscription entity.GameEntityID = 2002
)

// domainKeywords are the slice's question-matching keyword sets per
// knowledge domain. SLICE NOTE: keywords.go documents that per-NPC custom
// keywords belong in the .kg format; the current .kg schema has no keyword
// field, so the slice keeps them here until the schema extension lands.
var domainKeywords = map[string][]string{
	"craft":        {"ore", "steel", "smith", "forge", "metal", "mine", "flood", "hammer", "anvil", "guild", "seal", "name", "forgemaster"},
	"people":       {"miner", "smith", "quiet", "lied", "told", "ore", "people", "who"},
	"history":      {"empire", "imperial", "guild", "seal", "name", "forgemaster", "oath", "inscription", "slab", "script", "history"},
	"land":         {"mine", "flood", "shaft", "water", "stone", "warm", "heat", "cavern", "land"},
	"old_imperial": {"script", "inscription", "slab", "old imperial", "empire", "imperial", "judgment", "nameplate"},
}

// entityNames maps folded entity names to their slice GameEntityIDs for the
// question system's entity reference detection and the response processor's
// registry. SLICE NOTE: authored here until an entity-name data asset exists.
var entityNames = map[string]entity.GameEntityID{
	"vek":            entBlacksmith,
	"vek the smith":  entBlacksmith,
	"blacksmith":     entBlacksmith,
	"smith":          entBlacksmith,
	"tomas":          entMiner,
	"tomas the miner": entMiner,
	"miner":          entMiner,
	"mira":           entScholar,
	"mira the scholar": entScholar,
	"scholar":        entScholar,
	"old mine":       entOldMine,
	"mine":           entOldMine,
	"inscription":    entInscription,
	"slab":           entInscription,
}

// npcEntry is one NPC in the slice world: its cube entity, .kg graph, and
// the question-system domain list derived from it.
type npcEntry struct {
	id      entity.GameEntityID
	name    string
	cube    *engine.Entity
	pos     matrix.Vec2
	graph   knowledge.KnowledgeGraph
	domains []questions.Domain
}

// talkSystem owns the conversation pipeline and the thread engine.
type talkSystem struct {
	game  *Game
	panel *dialogue_ui.DialoguePanel
	npcs  []*npcEntry

	backend *aidialogue.Backend
	sm      *aidialogue.StateMachine // one per conversation
	active  *npcEntry

	threads *threads.Engine
	history []questions.QuestionRecord
}

// newTalkSystem loads the borderland NPC content, builds the dialogue panel,
// registers all facts with the thread engine, and starts the LLM backend
// (async spawn + warmup; degradation paths are the backend's own).
func newTalkSystem(g *Game) *talkSystem {
	t := &talkSystem{game: g}
	t.panel = dialogue_ui.NewDialoguePanel(g.host, &g.uiMan)
	t.panel.SetQuestionHandler(t.onQuestion)
	t.threads = threads.NewEngine(threads.WithNamer(t), threads.WithAutoChainMerge(true))

	for _, spec := range []struct {
		id   entity.GameEntityID
		file string
		pos  matrix.Vec2
	}{
		{entBlacksmith, "blacksmith.kg", worldBlacksmithPos},
		{entMiner, "miner.kg", worldMinerPos},
		{entScholar, "scholar.kg", worldScholarPos},
	} {
		graph, err := knowledge.ParseKGFile("assets/npc_knowledge/borderland/" + spec.file)
		if err != nil {
			slog.Error("talk: .kg load failed — NPC omitted from the world",
				"file", spec.file, "error", err)
			continue
		}
		npc := &npcEntry{
			id:      spec.id,
			name:    graph.NPCName,
			pos:     spec.pos,
			graph:   graph,
			domains: buildDomains(graph.Domains),
		}
		// The cube is created by buildWorld; link it here.
		npc.cube = g.world.npcCubeFor(spec.id)
		t.npcs = append(t.npcs, npc)
		for _, f := range graph.Facts {
			t.threads.RegisterFact(f)
		}
		slog.Info("talk: NPC loaded", "id", spec.id, "name", npc.name, "facts", len(graph.Facts))
	}

	t.startBackend()
	return t
}

// buildDomains converts .kg domain names into question-system domains with
// the slice's keyword sets. Domains without keywords still participate in
// entity-bonus matching.
func buildDomains(names []string) []questions.Domain {
	out := make([]questions.Domain, 0, len(names))
	for _, n := range names {
		out = append(out, questions.Domain{Name: n, Keywords: domainKeywords[n]})
	}
	return out
}

// startBackend launches the LLM companion subprocess with warmup running in
// the background (ADR-0008: model load overlaps gameplay). On failure the
// backend enters text mode; the state machine falls back accordingly.
func (t *talkSystem) startBackend() {
	cfg := aidialogue.DefaultConfig()
	cfg.BinaryPath = "bin/companion" // slice build layout (run from project root)
	cfg.ModelDirs = []string{"."}
	t.backend = aidialogue.New(cfg, slog.Default())
	if err := t.backend.Start(context.Background()); err != nil {
		slog.Error("talk: backend start failed", "error", err)
	}
}

// openDialogue begins a conversation with the given NPC.
func (t *talkSystem) openDialogue(npc *npcEntry) {
	t.active = npc
	t.sm = aidialogue.NewStateMachine(t.backend, uint64(npc.id))
	t.panel.Show(uint64(npc.id), npc.name, "")
	slog.Info("talk: dialogue opened", "npc", npc.name)
}

// npcAt returns the NPC whose ground position lies within the click radius
// of the clicked world point, or nil when the click hit open ground.
func (t *talkSystem) npcAt(click matrix.Vec3) *npcEntry {
	if t == nil {
		return nil
	}
	positions := make([]matrix.Vec2, 0, len(t.npcs))
	for _, npc := range t.npcs {
		positions = append(positions, npc.pos)
	}
	idx := PickNPC(matrix.NewVec2(click.X(), click.Z()), positions, worldNPCClickRadius)
	if idx < 0 {
		return nil
	}
	return t.npcs[idx]
}

// PickNPC returns the index of the NPC position within radius of the click
// point (XZ ground plane, nearest wins), or -1 when none qualifies. Pure —
// exported for unit testing.
func PickNPC(click matrix.Vec2, positions []matrix.Vec2, radius float32) int {
	best := radius
	bestIdx := -1
	for i, p := range positions {
		dx := p.X() - click.X()
		dz := p.Y() - click.Y()
		dist := float32(math.Sqrt(float64(dx*dx + dz*dz)))
		// First qualifier wins ties; a strict improvement replaces it. The
		// first-hit clause also absorbs float rounding at the boundary.
		if bestIdx == -1 && dist <= best || dist < best {
			best = dist
			bestIdx = i
		}
	}
	return bestIdx
}

// onQuestion runs the full pipeline for one submitted question.
func (t *talkSystem) onQuestion(q string) {
	npc := t.active
	if npc == nil {
		return
	}
	t.panel.AddMessage("You", q)

	match := questions.MatchTopicsWithContext(q, npc.domains, t, t.history,
		questions.Config{RelevanceThreshold: 0.1}) // slice tuning: single-keyword questions must match (GDD range 0.1–0.6)

	// Keep the last 3 questions for pronoun resolution (question system CR-3).
	t.history = append(t.history, questions.QuestionRecord{
		Question: q,
		Domains:  domainNames(match.MatchedDomains),
		Entities: match.EntityRefs,
	})
	if len(t.history) > 3 {
		t.history = t.history[len(t.history)-3:]
	}

	prompt, err := aidialogue.BuildPrompt(aidialogue.PromptConfig{
		NPC: aidialogue.NPC{
			ID:           strconv.FormatUint(uint64(npc.id), 10),
			Name:         npc.name,
			Personality:  npc.graph.Personality,
			SpeechStyle:  npc.graph.SpeechStyle,
			KnowsNot:     npc.graph.KnowsNot,
		},
		Facts:              toPromptFacts(npc.graph.Facts),
		MatchedDomains:     domainNames(match.MatchedDomains),
		DetectedEntityRefs: refStrings(match.EntityRefs),
		IsFallback:         match.IsFallback,
		Question:           q,
		TrustLevel:         1, // slice: conversations start at trust 1 (the payoff facts are trust-gated; trust progression is a production story)
	})
	if err != nil {
		slog.Error("talk: prompt build failed", "error", err)
		t.panel.AddResponse("(I can't make sense of that right now.)")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 35*time.Second)
	resp, err := t.sm.Submit(ctx, prompt)
	cancel()
	if err != nil {
		slog.Warn("talk: generation failed", "error", err)
		resp = "I've lost my words for a moment. Ask again."
	}

	processed := aidialogue.ProcessResponse(resp, aidialogue.ProcessConfig{
		MaxResponseTokens: 150,
		EntityRegistry:    t,
		FallbackFact:      bestFactContent(npc, match),
		FallbackNPCName:   npc.name,
	})
	t.panel.AddResponse(processed.Text)
	slog.Info("talk: response", "npc", npc.name, "text", processed.Text)

	// The player learned every trust-visible fact in the matched domains —
	// feed the thread engine so leads open threads and chains auto-link.
	for _, f := range factsForMatch(npc, match) {
		t.threads.OnFactDiscovered(f.FactID, npc.id)
	}

	// Keep the journal current if it is open.
	if t.game.journal != nil && t.game.journal.visible {
		t.game.journal.Refresh()
	}
}

// ── Question-system interfaces ──────────────────────────────────────────────

// Lookup implements questions.EntityNameIndex: folded-name → GameEntityID.
func (t *talkSystem) Lookup(name string) (entity.GameEntityID, bool) {
	id, ok := entityNames[name]
	return id, ok
}

// Name implements threads.EntityNamer: GameEntityID → display name.
func (t *talkSystem) Name(id entity.GameEntityID) (string, bool) {
	switch id {
	case entBlacksmith:
		return "Vek the Smith", true
	case entMiner:
		return "Tomas the Miner", true
	case entScholar:
		return "Mira the Scholar", true
	case entOldMine:
		return "the old mine", true
	case entInscription:
		return "the inscription", true
	}
	return "", false
}

// ── Response-processor interfaces ───────────────────────────────────────────

// IsRegistered implements aidialogue.EntityRegistry: name known in the world.
func (t *talkSystem) IsRegistered(name string) bool {
	_, ok := entityNames[strings.ToLower(name)]
	return ok
}

// AllNames implements aidialogue.EntityRegistry.
func (t *talkSystem) AllNames() []string {
	names := make([]string, 0, len(entityNames))
	for n := range entityNames {
		names = append(names, n)
	}
	return names
}

// KnowsNotEntities implements aidialogue.EntityRegistry: the active NPC's
// unknown-entity references (authored in the .kg file).
func (t *talkSystem) KnowsNotEntities() []string {
	if t.active == nil {
		return nil
	}
	return t.active.graph.KnowsNotEntities
}

// ── Helpers ─────────────────────────────────────────────────────────────────

// domainNames extracts the ranked domain names from a match result.
func domainNames(scores []questions.DomainScore) []string {
	names := make([]string, 0, len(scores))
	for _, s := range scores {
		names = append(names, s.Domain)
	}
	return names
}

// toPromptFacts converts KG facts to the prompt builder's fact type.
func toPromptFacts(facts []knowledge.Fact) []aidialogue.Fact {
	out := make([]aidialogue.Fact, 0, len(facts))
	for _, f := range facts {
		out = append(out, aidialogue.Fact{
			FactID:        f.FactID,
			Domain:        f.Domain,
			Type:          aidialogue.FactType(f.Type),
			Content:       f.Content,
			TrustRequired: f.TrustRequired,
			References:    refStrings(f.References),
		})
	}
	return out
}

// refStrings renders GameEntityIDs as canonical strings for the prompt
// builder and question system.
func refStrings(ids []entity.GameEntityID) []string {
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		out = append(out, strconv.FormatUint(uint64(id), 10))
	}
	return out
}

// factsForMatch returns the trust-visible facts in the matched domains
// (authoring order) — the facts the player just learned.
func factsForMatch(npc *npcEntry, match questions.MatchResult) []knowledge.Fact {
	if match.IsFallback {
		return nil // no domain matched — nothing specifically learned
	}
	seen := make(map[string]bool, len(match.MatchedDomains))
	for _, s := range match.MatchedDomains {
		seen[s.Domain] = true
	}
	var out []knowledge.Fact
	for _, f := range npc.graph.Facts {
		if knowledge.IsGated(f, 1) { // slice trust level, see PromptConfig note
			continue
		}
		if seen[f.Domain] {
			out = append(out, f)
		}
	}
	return out
}

// bestFactContent returns the highest-priority trust-visible fact content
// for the fallback path.
func bestFactContent(npc *npcEntry, match questions.MatchResult) string {
	facts := factsForMatch(npc, match)
	if len(facts) == 0 {
		return ""
	}
	return facts[0].Content
}

// npcCubeFor is implemented in world.go (the cubes are created there).
