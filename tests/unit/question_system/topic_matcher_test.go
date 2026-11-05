// Unit tests for Story Q-001: Topic Matcher & Domain Scoring.
//
// Evidence path:
// production/epics/question-system/story-001-topic-matcher-domain-scoring.md
// Each test maps to one acceptance criterion: AC-1
// (TestMatchTopics_AC1_DomainScoring_RoutesCraftMetallurgy), AC-2
// (TestMatchTopics_AC2_NoDomainMatched_Fallback), and the performance
// criterion (TestMatchTopics_Performance_P95Under5ms plus
// BenchmarkMatchTopics_P95Under5ms). The remaining tests cover the
// implementation notes: entity reference bonus, multi-word entity names,
// CR-3 pronoun resolution, data-driven config, determinism, and edge
// inputs (empty question, unicode).
package questions_test

import (
	"fmt"
	"math"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"

	"marrow-of-empire/entity"
	questions "marrow-of-empire/question_system"
)

// Test entity IDs (GameEntityID, ADR-0001).
const (
	entityOldMine      entity.GameEntityID = 1001 // location: "old mine"
	entityCaptainVeren entity.GameEntityID = 1002 // NPC: "Captain Veren"
)

// stubIndex is a test EntityNameIndex: folded entity name → GameEntityID.
// The matcher probes with lower-cased names (see EntityNameIndex contract).
type stubIndex map[string]entity.GameEntityID

func (s stubIndex) Lookup(name string) (entity.GameEntityID, bool) {
	id, ok := s[name]
	return id, ok
}

// gretaDomains returns the story's NPC fixture: Greta the blacksmith with
// domains ["craft_metallurgy", "trade_commerce"] and the default keyword
// sets (GDD CR-2 table).
func gretaDomains() []questions.Domain {
	return questions.DefaultDomains("craft_metallurgy", "trade_commerce")
}

// scoreFor returns the DomainScore for name, failing the test when missing.
func scoreFor(t *testing.T, scores []questions.DomainScore, name string) questions.DomainScore {
	t.Helper()
	for _, s := range scores {
		if s.Domain == name {
			return s
		}
	}
	t.Fatalf("domain %q not in scores: %+v", name, scores)
	return questions.DomainScore{}
}

// contains reports whether list contains want.
func contains(list []string, want string) bool {
	for _, s := range list {
		if s == want {
			return true
		}
	}
	return false
}

// near compares floats with tolerance for the matcher's exact arithmetic.
func near(a, b float64) bool {
	return math.Abs(a-b) <= 1e-9
}

// AC-1: GIVEN an NPC with domains ["craft_metallurgy", "trade_commerce"],
// WHEN the player types "What do you know about the ore from the old mine?",
// THEN craft_metallurgy scores above threshold ("ore" keyword overlap plus
// the "old mine" entity reference bonus) and the result routes matching
// facts: craft_metallurgy matched first, no fallback, entity refs detected.
//
// Note: "mine" is not a craft_metallurgy keyword in the GDD table, so the
// AC-1 match depends on the entity reference bonus (1.0 × 0.5) — the
// reading approved for this story. The entity bonus applies to every
// domain, so trade_commerce also matches at 0.5; the caller prioritizes
// facts referencing the detected entities (CR-2 step 1).
func TestMatchTopics_AC1_DomainScoring_RoutesCraftMetallurgy(t *testing.T) {
	// Arrange
	idx := stubIndex{"old mine": entityOldMine}

	// Act
	got := questions.MatchTopics(
		"What do you know about the ore from the old mine?",
		gretaDomains(), idx,
	)

	// Assert
	if got.IsFallback {
		t.Fatalf("AC-1: expected a match, got fallback: %+v", got)
	}
	craft := scoreFor(t, got.MatchedDomains, "craft_metallurgy")
	wantCraft := 1.0/13.0 + 0.5 // "ore" + entity bonus
	if !near(craft.Score, wantCraft) {
		t.Errorf("craft_metallurgy score = %v, want %v", craft.Score, wantCraft)
	}
	if len(craft.Keywords) != 1 || craft.Keywords[0] != "ore" {
		t.Errorf("craft_metallurgy matched keywords = %v, want [ore]", craft.Keywords)
	}
	if craft.FromContext {
		t.Error("craft_metallurgy matched directly, must not be FromContext")
	}
	// Entity bonus applies to every domain: trade_commerce also matches at
	// 0.5, ranked below craft_metallurgy (0.577 > 0.5).
	if len(got.MatchedDomains) != 2 || got.MatchedDomains[0].Domain != "craft_metallurgy" {
		t.Errorf("matched domains = %+v, want [craft_metallurgy, trade_commerce] ranked by score", got.MatchedDomains)
	}
	if !reflect.DeepEqual(got.EntityRefs, []entity.GameEntityID{entityOldMine}) {
		t.Errorf("entity refs = %v, want [old mine]", got.EntityRefs)
	}
}

// AC-2: GIVEN the same NPC, WHEN the player types "What happened to the old
// empire?", THEN no domain scores above threshold (the NPC has no
// old_empire_lore domain) and the result falls back: all domains at low
// relevance, IsFallback set — the caller routes all facts with a deflection
// instruction.
func TestMatchTopics_AC2_NoDomainMatched_Fallback(t *testing.T) {
	// Arrange
	idx := stubIndex{}

	// Act
	got := questions.MatchTopics("What happened to the old empire?", gretaDomains(), idx)

	// Assert
	if !got.IsFallback {
		t.Fatalf("AC-2: expected fallback, got: %+v", got.MatchedDomains)
	}
	if len(got.MatchedDomains) != 2 {
		t.Fatalf("fallback must include all NPC domains, got %d: %+v", len(got.MatchedDomains), got.MatchedDomains)
	}
	for _, d := range got.MatchedDomains {
		if d.Score != 0 {
			t.Errorf("fallback domain %q score = %v, want 0", d.Domain, d.Score)
		}
	}
	if len(got.EntityRefs) != 0 {
		t.Errorf("entity refs = %v, want none", got.EntityRefs)
	}
}

// Entity reference bonus: an entity mention alone (zero keyword matches)
// activates every domain at 1.0 × 0.5 = 0.5 ≥ 0.3 — facts referencing the
// entity surface regardless of domain (CR-2 step 1).
func TestMatchTopics_EntityBonus_ActivatesDomainsWithoutKeywords(t *testing.T) {
	idx := stubIndex{"old mine": entityOldMine}

	got := questions.MatchTopics("Tell me about the old mine.", gretaDomains(), idx)

	if got.IsFallback {
		t.Fatalf("entity mention must match, got fallback: %+v", got)
	}
	craft := scoreFor(t, got.MatchedDomains, "craft_metallurgy")
	if !near(craft.Score, 0.5) {
		t.Errorf("craft_metallurgy score = %v, want 0.5 (entity bonus only)", craft.Score)
	}
	if len(craft.Keywords) != 0 {
		t.Errorf("craft_metallurgy matched keywords = %v, want none", craft.Keywords)
	}
	if !reflect.DeepEqual(got.EntityRefs, []entity.GameEntityID{entityOldMine}) {
		t.Errorf("entity refs = %v, want [old mine]", got.EntityRefs)
	}
}

// Multi-word entity names: n-gram scanning detects "old mine" and "Captain
// Veren" as single references, the longest phrase wins at each position,
// and a shorter overlapping name ("mine" alone) is never reported.
func TestMatchTopics_MultiWordEntities_LongestMatchWins(t *testing.T) {
	idx := stubIndex{
		"old mine":      entityOldMine,
		"mine":          7, // shorter overlapping name must lose to "old mine"
		"captain veren": entityCaptainVeren,
	}

	got := questions.MatchTopics("What about the old mine and Captain Veren?", gretaDomains(), idx)

	want := []entity.GameEntityID{entityOldMine, entityCaptainVeren}
	if !reflect.DeepEqual(got.EntityRefs, want) {
		t.Errorf("entity refs = %v, want %v (order of first mention, longest match)", got.EntityRefs, want)
	}
}

// Repeated entity mentions are deduplicated in the refs list.
func TestMatchTopics_EntityRefs_Deduplicated(t *testing.T) {
	idx := stubIndex{"old mine": entityOldMine}

	got := questions.MatchTopics("The old mine and the old mine again?", gretaDomains(), idx)

	want := []entity.GameEntityID{entityOldMine}
	if !reflect.DeepEqual(got.EntityRefs, want) {
		t.Errorf("entity refs = %v, want %v", got.EntityRefs, want)
	}
}

// A single keyword match (1/13 ≈ 0.077) stays below the 0.3 threshold
// without an entity reference → fallback.
func TestMatchTopics_SingleKeywordMatch_BelowThreshold(t *testing.T) {
	got := questions.MatchTopics("What about ore?", gretaDomains(), stubIndex{})

	if !got.IsFallback {
		t.Fatalf("expected fallback, got: %+v", got.MatchedDomains)
	}
	craft := scoreFor(t, got.MatchedDomains, "craft_metallurgy")
	if !near(craft.Score, 1.0/13.0) {
		t.Errorf("craft_metallurgy score = %v, want 1/13", craft.Score)
	}
}

// All keywords match → score exactly 1.0, matched, every keyword reported
// in authoring order.
func TestMatchTopics_AllKeywordsMatch_FullScore(t *testing.T) {
	q := "ore metal forge smith steel iron alloy weapon armor blade bronze copper silver"

	got := questions.MatchTopics(q, questions.DefaultDomains("craft_metallurgy"), stubIndex{})

	if got.IsFallback {
		t.Fatal("full keyword overlap must match")
	}
	craft := scoreFor(t, got.MatchedDomains, "craft_metallurgy")
	if craft.Score != 1.0 {
		t.Errorf("craft_metallurgy score = %v, want 1.0", craft.Score)
	}
	if len(craft.Keywords) != 13 {
		t.Errorf("matched keywords = %v, want all 13", craft.Keywords)
	}
}

// Multi-word keyword "how long" matches as a contiguous token run:
// "Why have you lived here? How long?" scores 3/10 = 0.3, exactly at the
// threshold (≥ 0.3 matches).
func TestMatchTopics_MultiWordKeyword_HowLongScoresAtThreshold(t *testing.T) {
	got := questions.MatchTopics(
		"Why have you lived here? How long?",
		questions.DefaultDomains("personal_history"), stubIndex{},
	)

	if got.IsFallback {
		t.Fatal("3/10 = 0.3 must match at threshold 0.3")
	}
	ph := scoreFor(t, got.MatchedDomains, "personal_history")
	if !near(ph.Score, 0.3) {
		t.Errorf("personal_history score = %v, want 0.3", ph.Score)
	}
	if !contains(ph.Keywords, "how long") {
		t.Errorf("matched keywords = %v, want multi-word \"how long\" included", ph.Keywords)
	}
}

// Empty question: no tokens, no matches, clean fallback with all domains.
func TestMatchTopics_EmptyQuestion_FallbackAllDomains(t *testing.T) {
	got := questions.MatchTopics("", gretaDomains(), stubIndex{})

	if !got.IsFallback {
		t.Fatal("empty question must fall back")
	}
	if len(got.MatchedDomains) != 2 || len(got.EntityRefs) != 0 {
		t.Errorf("result = %+v, want 2 domains and no entity refs", got)
	}
}

// Unicode input: Cyrillic text tokenizes cleanly, matches no default
// keywords, and does not crash.
func TestMatchTopics_UnicodeQuestion_NoKeywordsNoCrash(t *testing.T) {
	got := questions.MatchTopics("Что ты знаешь о старой империи?", gretaDomains(), stubIndex{})

	if !got.IsFallback {
		t.Fatal("Cyrillic question with no keyword overlap must fall back")
	}
	if len(got.EntityRefs) != 0 {
		t.Errorf("entity refs = %v, want none", got.EntityRefs)
	}
}

// Unicode entity names match case-insensitively after folding.
func TestMatchTopics_UnicodeEntity_Detected(t *testing.T) {
	idx := stubIndex{"старая шахта": entityOldMine}

	got := questions.MatchTopics("Старая шахта. Что там?", questions.DefaultDomains("craft_metallurgy"), idx)

	if !reflect.DeepEqual(got.EntityRefs, []entity.GameEntityID{entityOldMine}) {
		t.Errorf("entity refs = %v, want [old mine]", got.EntityRefs)
	}
	if got.IsFallback {
		t.Fatal("entity bonus must match the domain")
	}
}

// CR-3: "What about it?" after "Tell me about the forge" resolves the
// pronoun against the prior topic context: craft_metallurgy is carried
// forward, marked FromContext, and the result is not a fallback.
func TestMatchTopics_PronounResolution_ResolvesFromHistory(t *testing.T) {
	history := []questions.QuestionRecord{
		{Question: "Tell me about the forge", Domains: []string{"craft_metallurgy"}},
	}

	got := questions.MatchTopicsWithContext("What about it?", gretaDomains(), stubIndex{}, history, questions.Config{})

	if got.IsFallback {
		t.Fatalf("pronoun question must resolve from history, got fallback: %+v", got)
	}
	craft := scoreFor(t, got.MatchedDomains, "craft_metallurgy")
	if !craft.FromContext {
		t.Errorf("craft_metallurgy must be marked FromContext: %+v", craft)
	}
	// Re-scored over "what about it tell me about the forge": "forge" 1/13.
	if !near(craft.Score, 1.0/13.0) {
		t.Errorf("craft_metallurgy score = %v, want 1/13 (augmented text)", craft.Score)
	}
	if !contains(craft.Keywords, "forge") {
		t.Errorf("craft_metallurgy matched keywords = %v, want forge", craft.Keywords)
	}
}

// A carried domain must exist in the NPC's domain list: history about
// old_empire_lore cannot be carried to an NPC without that domain.
func TestMatchTopics_PronounResolution_DomainNotOnNPC_NotCarried(t *testing.T) {
	history := []questions.QuestionRecord{
		{Question: "What happened to the old empire?", Domains: []string{"old_empire_lore"}},
	}

	got := questions.MatchTopicsWithContext("What about it?", gretaDomains(), stubIndex{}, history, questions.Config{})

	if !got.IsFallback {
		t.Fatalf("must fall back: NPC has no old_empire_lore domain: %+v", got.MatchedDomains)
	}
	for _, d := range got.MatchedDomains {
		if d.Domain == "old_empire_lore" {
			t.Error("old_empire_lore carried to an NPC that does not have it")
		}
	}
}

// A pronoun question that already matches directly ignores history: the
// direct match is used and nothing is marked FromContext.
func TestMatchTopics_PronounWithDirectMatch_HistoryIgnored(t *testing.T) {
	history := []questions.QuestionRecord{
		{Question: "Tell me about the forge", Domains: []string{"craft_metallurgy"}},
	}

	got := questions.MatchTopicsWithContext(
		"What about that steel blade and iron ore?", gretaDomains(), stubIndex{}, history, questions.Config{})

	if got.IsFallback {
		t.Fatal("4/13 = 0.308 ≥ 0.3 must match directly")
	}
	craft := scoreFor(t, got.MatchedDomains, "craft_metallurgy")
	if craft.FromContext {
		t.Error("direct match must not be marked FromContext")
	}
	if !near(craft.Score, 4.0/13.0) {
		t.Errorf("craft_metallurgy score = %v, want 4/13", craft.Score)
	}
}

// Entity references resolve through history: "What about him?" re-detects
// "Captain Veren" in the augmented text and carries the entity ref.
func TestMatchTopics_PronounResolution_EntityResolvedFromHistory(t *testing.T) {
	idx := stubIndex{"captain veren": entityCaptainVeren}
	history := []questions.QuestionRecord{
		{
			Question: "Tell me about Captain Veren",
			Domains:  []string{"craft_metallurgy"},
			Entities: []entity.GameEntityID{entityCaptainVeren},
		},
	}

	got := questions.MatchTopicsWithContext("What about him?", gretaDomains(), idx, history, questions.Config{})

	if got.IsFallback {
		t.Fatalf("pronoun must resolve via history: %+v", got)
	}
	if !reflect.DeepEqual(got.EntityRefs, []entity.GameEntityID{entityCaptainVeren}) {
		t.Errorf("entity refs = %v, want [captain veren]", got.EntityRefs)
	}
}

// Entity references stored on a record but absent from its text (a record
// resolved from an earlier context) are carried forward directly.
func TestMatchTopics_PronounResolution_RecordEntityCarriedForward(t *testing.T) {
	idx := stubIndex{"captain veren": entityCaptainVeren}
	history := []questions.QuestionRecord{
		{
			Question: "Tell me about the forge",
			Domains:  []string{"craft_metallurgy"},
			Entities: []entity.GameEntityID{entityCaptainVeren}, // not in the text
		},
	}

	got := questions.MatchTopicsWithContext("What about it?", gretaDomains(), idx, history, questions.Config{})

	if !reflect.DeepEqual(got.EntityRefs, []entity.GameEntityID{entityCaptainVeren}) {
		t.Errorf("entity refs = %v, want [captain veren] carried from record", got.EntityRefs)
	}
}

// Ranking: matched domains are ordered by score descending; ties keep
// authoring order (stable sort). Fallback domains are ranked the same way.
func TestMatchTopics_Ranking_ScoreDescendingTiesStable(t *testing.T) {
	got := questions.MatchTopics("ore steel and trade goods", gretaDomains(), stubIndex{})

	// trade_commerce 2/10 = 0.2 > craft_metallurgy 2/13 ≈ 0.154 — both below
	// the 0.3 threshold, so this is a fallback, still ranked by score.
	if !got.IsFallback {
		t.Fatalf("both scores below threshold, want fallback: %+v", got.MatchedDomains)
	}
	if len(got.MatchedDomains) != 2 {
		t.Fatalf("fallback must list all domains, got %d", len(got.MatchedDomains))
	}
	if got.MatchedDomains[0].Domain != "trade_commerce" {
		t.Errorf("first ranked domain = %q, want trade_commerce (0.2 > 0.154)", got.MatchedDomains[0].Domain)
	}
	if got.MatchedDomains[0].Score < got.MatchedDomains[1].Score {
		t.Error("domains must be ranked by score descending")
	}
}

// Config tuning: lowering the relevance threshold to 0.1 matches
// craft_metallurgy at 2/13 ≈ 0.154, which the default 0.3 rejects — the
// threshold is data-driven, not hardcoded.
func TestMatchTopics_Config_ThresholdIsDataDriven(t *testing.T) {
	q := "What about ore and steel?"

	got := questions.MatchTopicsWithContext(q, gretaDomains(), stubIndex{}, nil,
		questions.Config{RelevanceThreshold: 0.1})

	if got.IsFallback {
		t.Fatal("threshold 0.1 must match 2/13 ≈ 0.154")
	}
	craft := scoreFor(t, got.MatchedDomains, "craft_metallurgy")
	if !near(craft.Score, 2.0/13.0) {
		t.Errorf("craft_metallurgy score = %v, want 2/13", craft.Score)
	}

	// Control: the default 0.3 threshold falls back for the same question.
	control := questions.MatchTopics(q, gretaDomains(), stubIndex{})
	if !control.IsFallback {
		t.Fatal("default threshold 0.3 must reject 2/13 ≈ 0.154")
	}
}

// A nil EntityNameIndex is treated as an empty index: no entity refs, no
// panic, scoring proceeds normally.
func TestMatchTopics_NilEntityIndex_NoEntitiesNoPanic(t *testing.T) {
	got := questions.MatchTopics("What about ore?", gretaDomains(), nil)

	if len(got.EntityRefs) != 0 {
		t.Errorf("entity refs = %v, want none", got.EntityRefs)
	}
	if !got.IsFallback {
		t.Fatal("1/13 without entity bonus must fall back")
	}
}

// Determinism: identical inputs produce identical results — scoring is a
// pure function, the same question always routes the same domains.
func TestMatchTopics_Deterministic_SameInputSameOutput(t *testing.T) {
	idx := stubIndex{"old mine": entityOldMine}
	q := "What do you know about the ore from the old mine?"

	a := questions.MatchTopics(q, gretaDomains(), idx)
	b := questions.MatchTopics(q, gretaDomains(), idx)

	if !reflect.DeepEqual(a, b) {
		t.Fatalf("same input produced different results:\na = %+v\nb = %+v", a, b)
	}
}

// p95Latency runs matchFn n times and returns the p95 (95th percentile) of
// the per-call wall-clock durations. Wall-clock includes scheduler noise,
// so the 5ms guardrail keeps an order-of-magnitude safety margin over
// typical matching times (tens of microseconds).
func p95Latency(n int, matchFn func()) time.Duration {
	durs := make([]time.Duration, n)
	for i := range durs {
		start := time.Now()
		matchFn()
		durs[i] = time.Since(start)
	}
	sort.Slice(durs, func(i, j int) bool { return durs[i] < durs[j] })
	idx := int(float64(n)*0.95) - 1
	if idx < 0 {
		idx = 0
	}
	return durs[idx]
}

// performanceDomains returns the full standard keyword set: all 9 domains
// from the GDD CR-2 table (~10 keywords each).
func performanceDomains() []questions.Domain {
	return questions.DefaultDomains(
		"craft_metallurgy", "herbalism_medicine", "town_gossip",
		"military_security", "old_empire_lore", "wilderness_survival",
		"trade_commerce", "faction_politics", "personal_history",
	)
}

// performanceIndex returns an entity index of ~200 names, mostly
// multi-word, exercising n-gram probing at the story's registry scale.
func performanceIndex() questions.EntityNameIndex {
	idx := stubIndex{
		"old mine":         entityOldMine,
		"captain veren":    entityCaptainVeren,
		"blackrock quarry": 1003,
		"ley tower":        1004,
		"greta blacksmith": 1005,
		"white river":      1006,
	}
	for i := 0; i < 200; i++ {
		idx[fmt.Sprintf("citizen%d surname%d", i, i)] = entity.GameEntityID(2000 + i)
	}
	return idx
}

// performanceQuestion returns a 500-character question mixing domain
// keywords and entity names, per the story's performance criterion.
func performanceQuestion() string {
	base := "What do you know about the ore from the old mine near the river " +
		"where the merchant sells rare goods, and how does the guard patrol " +
		"the wall when the weather turns cold and beasts track toward camp? "
	return strings.Repeat(base, 500/len(base)+1)[:500]
}

// Performance acceptance criterion: with the full domain keyword set (9
// domains × ~10 keywords) and a 500-character question, p95 matching time
// over 1000 iterations is ≤ 5ms (GDD performance AC, story Performance).
func TestMatchTopics_Performance_P95Under5ms(t *testing.T) {
	idx := performanceIndex()
	q := performanceQuestion()
	domains := performanceDomains()

	p95 := p95Latency(1000, func() {
		questions.MatchTopics(q, domains, idx)
	})

	if p95 > 5*time.Millisecond {
		t.Fatalf("p95 matching time = %v, want ≤ 5ms (story performance AC)", p95)
	}
	t.Logf("p95 over 1000 iterations: %v", p95)
}

// BenchmarkMatchTopics_P95Under5ms runs the same acceptance criterion under
// the Go benchmark harness: each benchmark iteration performs 1000 matches
// and reports the p95 latency in milliseconds.
func BenchmarkMatchTopics_P95Under5ms(b *testing.B) {
	idx := performanceIndex()
	q := performanceQuestion()
	domains := performanceDomains()
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		p95 := p95Latency(1000, func() {
			questions.MatchTopics(q, domains, idx)
		})
		b.ReportMetric(float64(p95)/float64(time.Millisecond), "p95-ms")
		if p95 > 5*time.Millisecond {
			b.Fatalf("p95 = %v exceeds the 5ms budget", p95)
		}
	}
}
