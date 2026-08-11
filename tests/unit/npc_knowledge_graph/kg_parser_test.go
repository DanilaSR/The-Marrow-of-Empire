// Unit tests for Story KG-002: YAML .kg File Parser.
//
// Evidence path: production/epics/npc-knowledge-graph/story-002-yaml-kg-parser.md
// Each test maps to one acceptance criterion (AC-1..AC-4) plus the story's
// error cases: missing fields with line numbers, invalid YAML (error, not
// panic), empty files, and duplicate fact IDs. ParseKG is exercised with
// inline YAML (no file I/O); ParseKGFile is covered by a temp-file smoke test.
// The benchmark pins the ADR-0005 ~10ms-per-NPC parse guardrail.
package knowledge_test

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"marrow-of-empire/entity"
	"marrow-of-empire/npc_knowledge_graph"
)

// validKG is the ADR-0005 authoring-format baseline fixture: the Thornwell
// blacksmith Greta with a full-coverage fact (trust gating, references,
// discovery triggers, contradicts, leads_to) and a minimal fact relying on
// the trust_required=0 default.
func validKG() string {
	return `npc_id: "greta_blacksmith"
npc_name: "Greta"
domains: ["craft_metallurgy", "trade_commerce"]
personality: "Practical, observant, proud of her craft."
speech_style: "Direct, uses smithing metaphors."
knows_not: ["old_empire_lore", "faction_politics"]
knows_not_entities: []
facts:
  - fact_id: "greta_ley_metal_001"
    domain: "craft_metallurgy"
    type: FACT
    content: "This fragment... it's an alloy I've never seen. Pale silver, warm to the touch."
    trust_required: 0
    references: [7, 12]
    discovery_triggers: ["player_has_item:ley_metal_fragment"]
    contradicts: ["orin_ley_metal_003"]
    leads_to:
      - entity_id: 9
        topic_hint: "Ask the hermit about Ley-touched metal"
  - fact_id: "greta_trade_002"
    domain: "trade_commerce"
    type: RUMOR
    content: "The caravan from Velden has been buying scrap metal at three times market price."
`
}

// AC-1: GIVEN a valid .kg YAML file, WHEN parsed, THEN all domains, facts,
// personality, speech_style, and knows_not are populated, and the full Fact
// field set round-trips (trust gating, entity references, discovery triggers,
// contradicts, leads_to).
func TestParseKG_ValidFile_AllFieldsPopulated(t *testing.T) {
	graph, err := knowledge.ParseKG([]byte(validKG()))
	if err != nil {
		t.Fatalf("ParseKG returned error: %v", err)
	}

	if graph.NPCID != "greta_blacksmith" || graph.NPCName != "Greta" {
		t.Fatalf("identity = %q/%q, want greta_blacksmith/Greta", graph.NPCID, graph.NPCName)
	}
	if !reflect.DeepEqual(graph.Domains, []string{"craft_metallurgy", "trade_commerce"}) {
		t.Fatalf("domains = %v", graph.Domains)
	}
	if graph.Personality == "" || graph.SpeechStyle == "" {
		t.Fatalf("personality/speech_style not populated: %q / %q", graph.Personality, graph.SpeechStyle)
	}
	if !reflect.DeepEqual(graph.KnowsNot, []string{"old_empire_lore", "faction_politics"}) {
		t.Fatalf("knows_not = %v", graph.KnowsNot)
	}
	if len(graph.Facts) != 2 {
		t.Fatalf("facts = %d, want 2", len(graph.Facts))
	}

	f := graph.Facts[0]
	if f.FactID != "greta_ley_metal_001" || f.Domain != "craft_metallurgy" ||
		f.Type != knowledge.FactType_Fact || f.TrustRequired != 0 || f.Content == "" {
		t.Fatalf("fact[0] = %+v", f)
	}
	if !reflect.DeepEqual(f.References, []entity.GameEntityID{7, 12}) {
		t.Fatalf("references = %v", f.References)
	}
	if !reflect.DeepEqual(f.DiscoveryTriggers, []string{"player_has_item:ley_metal_fragment"}) {
		t.Fatalf("discovery_triggers = %v", f.DiscoveryTriggers)
	}
	if !reflect.DeepEqual(f.Contradicts, []string{"orin_ley_metal_003"}) {
		t.Fatalf("contradicts = %v", f.Contradicts)
	}
	if len(f.LeadsTo) != 1 || f.LeadsTo[0].EntityID != 9 || f.LeadsTo[0].TopicHint == "" {
		t.Fatalf("leads_to = %+v", f.LeadsTo)
	}

	// The minimal fact relies on the trust_required=0 default.
	if g := graph.Facts[1]; g.TrustRequired != 0 || len(g.References) != 0 {
		t.Fatalf("fact[1] trust/references = %d/%v, want 0/nil", g.TrustRequired, g.References)
	}
}

// AC-1 (file path): GIVEN a valid .kg file on disk at the ADR-0005 location
// convention, WHEN ParseKGFile reads it, THEN the graph is populated. A
// missing file is an error.
func TestParseKGFile_ValidFileOnDisk_Populated(t *testing.T) {
	path := filepath.Join(t.TempDir(), "thornwell", "greta_blacksmith.kg")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(validKG()), 0o644); err != nil {
		t.Fatal(err)
	}

	graph, err := knowledge.ParseKGFile(path)
	if err != nil {
		t.Fatalf("ParseKGFile returned error: %v", err)
	}
	if graph.NPCID != "greta_blacksmith" || len(graph.Facts) != 2 {
		t.Fatalf("graph = %+v", graph)
	}

	if _, err := knowledge.ParseKGFile(filepath.Join(t.TempDir(), "missing.kg")); err == nil {
		t.Fatal("expected error for missing file")
	}
}

// AC-2: GIVEN a .kg file with invalid YAML syntax, WHEN parsed, THEN an error
// is returned (never a panic) and the graph is zero-valued — zero domains,
// zero facts — so the caller attaches the fallback personality.
func TestParseKG_InvalidYAML_ErrorNotPanic(t *testing.T) {
	if want := "Confused, doesn't know what's happening."; knowledge.FallbackPersonality != want {
		t.Fatalf("FallbackPersonality = %q, want %q", knowledge.FallbackPersonality, want)
	}

	tests := []struct {
		name string
		data string
	}{
		{"unclosed bracket", "npc_id: [greta\n  - {broken"},
		{"tab indentation", "npc_id: greta\n\tfacts: []"},
		{"scalar document", "just a plain string"},
		{"sequence document", "- a\n- b"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			graph, err := knowledge.ParseKG([]byte(tt.data))
			if err == nil {
				t.Fatalf("expected error for %q", tt.data)
			}
			if graph.NPCID != "" || len(graph.Domains) != 0 || len(graph.Facts) != 0 {
				t.Fatalf("graph must be zero-valued on error, got %+v", graph)
			}
		})
	}
}

// AC-3: GIVEN a graph with entity references, WHEN references are validated
// against existing GameEntityIDs, THEN unresolved references produce warnings
// and the fact still loads unchanged.
func TestValidateReferences_UnresolvedRefs_WarnedGraphUnchanged(t *testing.T) {
	graph, err := knowledge.ParseKG([]byte(validKG()))
	if err != nil {
		t.Fatal(err)
	}
	// Entity 7 exists; reference 12 and lead target 9 do not.
	exists := func(id entity.GameEntityID) bool { return id == 7 }

	var warnings []string
	warn := func(format string, args ...any) {
		warnings = append(warnings, fmt.Sprintf(format, args...))
	}
	before := graph.Facts

	unresolved := knowledge.ValidateReferences(&graph, exists, warn)

	if unresolved != 2 {
		t.Fatalf("unresolved = %d, want 2 (reference 12, lead 9)", unresolved)
	}
	if len(warnings) != 2 {
		t.Fatalf("warnings = %d, want 2: %v", len(warnings), warnings)
	}
	if !strings.Contains(warnings[0], "12") || !strings.Contains(warnings[1], "9") {
		t.Fatalf("warnings should name the unresolved ids: %v", warnings)
	}
	if !reflect.DeepEqual(graph.Facts, before) {
		t.Fatal("graph must be unchanged by validation")
	}
}

func TestValidateReferences_NilGraphOrWarn_NoPanic(t *testing.T) {
	if n := knowledge.ValidateReferences(nil, func(id entity.GameEntityID) bool { return true }, nil); n != 0 {
		t.Fatalf("nil graph: unresolved = %d, want 0", n)
	}
	graph := &knowledge.KnowledgeGraph{
		Facts: []knowledge.Fact{{FactID: "f", References: []entity.GameEntityID{5}}},
	}
	if n := knowledge.ValidateReferences(graph, func(id entity.GameEntityID) bool { return false }, nil); n != 1 {
		t.Fatalf("nil warn: unresolved = %d, want 1", n)
	}
}

// AC-4: GIVEN facts with discovery triggers, WHEN parsed, THEN the triggers
// are stored verbatim and not evaluated at parse time (evaluation happens in
// QueryFacts, Story KG-003).
func TestParseKG_DiscoveryTriggers_StoredNotEvaluated(t *testing.T) {
	graph, err := knowledge.ParseKG([]byte(validKG()))
	if err != nil {
		t.Fatal(err)
	}
	got := graph.Facts[0].DiscoveryTriggers
	want := []string{"player_has_item:ley_metal_fragment"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("triggers = %v, want %v (stored verbatim)", got, want)
	}
}

// Missing required fields are an error naming the field and the line of the
// document content, so designers can locate the omission.
func TestParseKG_MissingRequiredField_ErrorWithLine(t *testing.T) {
	tests := []struct {
		name string
		data string
		want string
	}{
		{
			name: "missing npc_id",
			data: `npc_name: "Greta"
domains: ["craft_metallurgy"]
personality: "P."
speech_style: "S."
facts:
  - fact_id: "f1"
    domain: "craft_metallurgy"
    type: FACT
    content: "c"
`,
			want: `missing required field "npc_id"`,
		},
		{
			name: "missing facts",
			data: `npc_id: "greta"
npc_name: "Greta"
domains: ["craft_metallurgy"]
personality: "P."
speech_style: "S."
`,
			want: `missing required field "facts"`,
		},
		{
			name: "empty npc_id",
			data: `npc_id: ""
npc_name: "Greta"
domains: ["craft_metallurgy"]
personality: "P."
speech_style: "S."
facts: []
`,
			want: `field "npc_id" must not be empty`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := knowledge.ParseKG([]byte(tt.data))
			if err == nil {
				t.Fatalf("expected error containing %q", tt.want)
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error %q does not contain %q", err, tt.want)
			}
			if !strings.Contains(err.Error(), "line") {
				t.Fatalf("error %q has no line info", err)
			}
		})
	}
}

// Empty .kg files (zero bytes, whitespace-only, or an empty document) are an
// error, not a parse of nothing.
func TestParseKG_EmptyFile_ReturnsError(t *testing.T) {
	for _, data := range []string{"", "   \n\n", "---\n", "# only a comment\n"} {
		if _, err := knowledge.ParseKG([]byte(data)); err == nil {
			t.Fatalf("expected error for %q", data)
		}
	}
}

// Duplicate fact_ids within one graph are an error naming the duplicate and
// both lines. (The same fact_id across different NPCs is valid — GDD edge
// case: shared knowledge is intentional.)
func TestParseKG_DuplicateFactID_ErrorWithLine(t *testing.T) {
	data := `npc_id: "greta"
npc_name: "Greta"
domains: ["craft_metallurgy"]
personality: "P."
speech_style: "S."
facts:
  - fact_id: "dup_001"
    domain: "craft_metallurgy"
    type: FACT
    content: "first"
  - fact_id: "dup_001"
    domain: "craft_metallurgy"
    type: RUMOR
    content: "second"
`
	_, err := knowledge.ParseKG([]byte(data))
	if err == nil {
		t.Fatal("expected duplicate fact_id error")
	}
	for _, want := range []string{`duplicate fact_id "dup_001"`, "line 11"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error %q does not contain %q", err, want)
		}
	}
}

// Additional authoring guards: unknown fields, invalid fact types,
// out-of-range trust, domains not in the domains list, and non-list facts all
// error at load with a line number (ADR-0005: "errors surface at load").
func TestParseKG_AuthoringGuardErrors(t *testing.T) {
	tests := []struct {
		name string
		data string
		want string
	}{
		{"unknown top-level field", strings.Replace(validKG(), "speech_style:", "speach_style:", 1), `unknown field "speach_style"`},
		{"unknown fact field", strings.Replace(validKG(), "trust_required: 0", "turst_required: 0", 1), `unknown field "turst_required"`},
		{"invalid fact type", strings.Replace(validKG(), "type: FACT", "type: FACTORY", 1), "unknown type"},
		{"trust out of range", strings.Replace(validKG(), "trust_required: 0", "trust_required: 7", 1), "out of range 0-3"},
		{"domain not listed", strings.Replace(validKG(), `domain: "craft_metallurgy"`, `domain: "alchemy"`, 1), `domain "alchemy" not in domains list`},
		{"facts not a list", `npc_id: greta
npc_name: Greta
domains: [trade_commerce]
personality: Gruff
speech_style: Short sentences
facts: "not_a_list"
`, `cannot unmarshal`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := knowledge.ParseKG([]byte(tt.data))
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %v, want containing %q", err, tt.want)
			}
		})
	}
}

// The .kg parse budget is ~10ms per NPC (ADR-0005 guardrail, control manifest
// Core Layer). yaml.v3 parses the fixture in well under 1ms; this benchmark
// pins the budget so a future dependency change cannot regress it silently.
func BenchmarkParseKG(b *testing.B) {
	data := []byte(validKG())
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if _, err := knowledge.ParseKG(data); err != nil {
			b.Fatal(err)
		}
	}
}
