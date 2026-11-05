// Unit tests for Story 003: Response Post-Processor.
//
// Evidence path: production/epics/ai-dialogue-engine/story-003-response-post-processor.md
// Covers AC-10 through AC-14, AC-30, plus edge cases: knows_not gating,
// player-whitelisted names, deadband, ambiguous fuzzy match, boundaries.
package aidialogue_test

import (
	"strings"
	"testing"

	aidialogue "marrow-of-empire/ai_dialogue"
)

// stubRegistry is a test-only EntityRegistry.
type stubRegistry struct {
	names    map[string]bool
	all      []string
	knowsNot []string
}

func (r *stubRegistry) IsRegistered(name string) bool { return r.names[name] }
func (r *stubRegistry) AllNames() []string             { return r.all }
func (r *stubRegistry) KnowsNotEntities() []string      { return r.knowsNot }

func newRegistry(names ...string) *stubRegistry {
	r := &stubRegistry{names: make(map[string]bool, len(names)), all: names}
	for _, n := range names {
		r.names[n] = true
	}
	return r
}

func (r *stubRegistry) withKnowsNot(names ...string) *stubRegistry {
	r.knowsNot = names
	return r
}

// ---- AC-10: unknown entity → sentence stripped + hallucination logged ----

func TestProcessResponse_AC10_UnknownEntity_StripsSentenceAndLogsHallucination(t *testing.T) {
	reg := newRegistry("Greta", "Orin", "Theras", "OldMine")
	cfg := aidialogue.ProcessConfig{EntityRegistry: reg, MaxResponseTokens: 150}

	raw := "The ore comes from the Shadowforge."
	result := aidialogue.ProcessResponse(raw, cfg)

	if len(result.Hallucinations) != 1 || result.Hallucinations[0] != "Shadowforge" {
		t.Errorf("expected hallucination [Shadowforge], got %v", result.Hallucinations)
	}
	if len(result.SentencesStripped) != 1 {
		t.Errorf("expected 1 sentence stripped, got %d", len(result.SentencesStripped))
	}
	if !result.IsFallback {
		t.Error("all sentences stripped should trigger IsFallback")
	}

	// With an actual fallback fact — check quote-styled format.
	cfg.FallbackFact = "The ore is imported from the north"
	cfg.FallbackNPCName = "Greta"
	result2 := aidialogue.ProcessResponse(raw, cfg)
	if !result2.IsFallback {
		t.Error("expected IsFallback with FallbackFact set")
	}
	expected := `"The ore is imported from the north" — Greta`
	if result2.Text != expected {
		t.Errorf("expected quote-styled fallback %q, got %q", expected, result2.Text)
	}
}

// ---- AC-11: fuzzy match → corrected + near-miss logged ----

func TestProcessResponse_AC11_FuzzyEntityMatch_CorrectsAndLogsNearMiss(t *testing.T) {
	reg := newRegistry("Greta", "Orin")
	cfg := aidialogue.ProcessConfig{EntityRegistry: reg, MaxResponseTokens: 150}

	// Distance 1: Gretta → Greta.
	raw := "Gretta showed me the forge."
	result := aidialogue.ProcessResponse(raw, cfg)

	if !strings.Contains(result.Text, "Greta") || strings.Contains(result.Text, "Gretta") {
		t.Errorf("expected 'Gretta'→'Greta' in text, got %q", result.Text)
	}
	if result.IsFallback {
		t.Error("valid fuzzy-corrected response should not trigger fallback")
	}
	hasNM := false
	for _, nm := range result.NearMisses {
		if nm == "Gretta→Greta" {
			hasNM = true
			break
		}
	}
	if !hasNM {
		t.Errorf("expected near-miss 'Gretta→Greta', got %v", result.NearMisses)
	}
}

// Distance 2 boundary: "Greti" → "Greta" (delete 'i', substitute 'e'→'a'... actually "Greti" vs "Greta": G-r-e-t-i vs G-r-e-t-a, substitute i→a at position 4, dist=1).
// Use "Gretta" which is distance 1. For distance 2: "Grat" → "Greta" — dist=2 (insert 'e', insert 'a'? Let me think: G-r-a-t vs G-r-e-t-a... actually levenshtein("Grat","Greta") = characters differ at positions 2 and 3. Let me try "Gretti" vs "Greta": G-r-e-t-t-i vs G-r-e-t-a = delete one 't', substitute i→a = dist 2.

func TestProcessResponse_AC11_FuzzyMatch_Distance2Boundary_Corrects(t *testing.T) {
	reg := newRegistry("Greta")
	cfg := aidialogue.ProcessConfig{EntityRegistry: reg, MaxResponseTokens: 150}

	// "Gretti" has Levenshtein distance 2 from "Greta" → should correct.
	raw := "Gretti showed me the forge."
	result := aidialogue.ProcessResponse(raw, cfg)
	if strings.Contains(result.Text, "Gretti") || !strings.Contains(result.Text, "Greta") {
		t.Errorf("distance-2 should correct: got %q", result.Text)
	}
}

// Distance 3 boundary: "Gretter" → "Greta" — dist=3 (delete 2 chars r,r and substitute e→a... let me just verify).
// Actually: G-r-e-t-t-e-r (7) vs G-r-e-t-a (5). That's |7-5|=2 length diff, plus substitution at position 3 (t→a) = 3 total. Should NOT correct.

func TestProcessResponse_AC11_FuzzyMatch_Distance3Boundary_Strips(t *testing.T) {
	reg := newRegistry("Greta")
	cfg := aidialogue.ProcessConfig{EntityRegistry: reg, MaxResponseTokens: 150, FallbackFact: "fallback"}

	// "Gretter" has Levenshtein distance 3 from "Greta" → should strip.
	raw := "Gretter showed me the forge."
	result := aidialogue.ProcessResponse(raw, cfg)
	if result.IsFallback {
		// distance 3 → hallucination → strip → all stripped → fallback
	} else {
		t.Error("distance-3 should strip sentence (triggering fallback since all stripped)")
	}
}

// ---- AC-12: empty/garbage → fallback fact ----

func TestProcessResponse_AC12_EmptyResponse_UsesFallbackFact(t *testing.T) {
	cfg := aidialogue.ProcessConfig{FallbackFact: "The best ore comes from the old mine", FallbackNPCName: "Greta"}

	result := aidialogue.ProcessResponse("", cfg)
	if !result.IsFallback {
		t.Error("empty response should trigger IsFallback")
	}
	expected := `"The best ore comes from the old mine" — Greta`
	if result.Text != expected {
		t.Errorf("expected %q, got %q", expected, result.Text)
	}
}

func TestProcessResponse_AC12_StopTokensOnly_ResultsInFallback(t *testing.T) {
	cfg := aidialogue.ProcessConfig{FallbackFact: "The old mine is east of town", FallbackNPCName: "Orin"}

	result := aidialogue.ProcessResponse("<|eot_id|> <|end_of_text|>", cfg)
	if !result.IsFallback || !strings.Contains(result.Text, cfg.FallbackFact) {
		t.Errorf("stop-token-only should fallback, got IsFallback=%v text=%q", result.IsFallback, result.Text)
	}
}

// ---- AC-13: repetition detection ----

func TestProcessResponse_AC13_RepetitionDetection_Above85Pct_NeedsRetry(t *testing.T) {
	cfg := aidialogue.ProcessConfig{
		MaxResponseTokens: 150,
		PreviousResponse:  "The old mine is east of town past the ironwood grove",
	}
	result := aidialogue.ProcessResponse("The old mine is east of town past the ironwood grove", cfg)
	if !result.NeedsRetry {
		t.Error("identical response (Jaccard=1.0) should trigger NeedsRetry")
	}
}

func TestProcessResponse_AC13_RepetitionDetection_Below80Pct_NoAction(t *testing.T) {
	cfg := aidialogue.ProcessConfig{
		MaxResponseTokens: 150,
		PreviousResponse:  "The old mine is east of town past the ironwood grove",
	}
	result := aidialogue.ProcessResponse("I don't know anything about that traveler ask Orin the guard captain", cfg)
	if result.NeedsRetry {
		t.Error("dissimilar responses should not trigger NeedsRetry")
	}
}

func TestProcessResponse_AC13_RepetitionDetection_Deadband_NoAction(t *testing.T) {
	// 9 shared words out of 11 unique words → Jaccard = 9/11 ≈ 0.818 (in 80-85% deadband).
	prev := "a b c d e f g h i j"
	curr := "a b c d e f g h i k"
	// prev tokens: {a,b,c,d,e,f,g,h,i,j}
	// curr tokens: {a,b,c,d,e,f,g,h,i,k}
	// intersection: {a,b,c,d,e,f,g,h,i} = 9
	// union: {a,b,c,d,e,f,g,h,i,j,k} = 11
	// Jaccard = 9/11 ≈ 0.818

	cfg := aidialogue.ProcessConfig{MaxResponseTokens: 150, PreviousResponse: prev}
	result := aidialogue.ProcessResponse(curr, cfg)
	if result.NeedsRetry {
		t.Error("~81.8% Jaccard in deadband should NOT trigger NeedsRetry")
	}
}

func TestProcessResponse_AC13_RepetitionDetection_Exactly85Pct_NoRetry(t *testing.T) {
	// 17 shared out of 20 unique → Jaccard = 17/20 = 0.85 exactly.
	// Threshold is strict >, so exactly 0.85 should NOT trigger retry.
	prev := "a b c d e f g h i j k l m n o p q"
	curr := "a b c d e f g h i j k l m n o p q r s t"
	// prev tokens: {a..q} = 17 unique
	// curr tokens: {a..q} ∪ {r,s,t} = 20 unique (all 17 shared)
	// Jaccard = 17/20 = 0.85

	cfg := aidialogue.ProcessConfig{MaxResponseTokens: 150, PreviousResponse: prev}
	result := aidialogue.ProcessResponse(curr, cfg)
	if result.NeedsRetry {
		t.Error("exactly 85% Jaccard should NOT retry (threshold is strict >)")
	}
}

func TestProcessResponse_AC13_RepetitionDetection_Exactly10Tokens_Checked(t *testing.T) {
	// 10 tokens is the minimum for repetition detection (>=10 triggers check).
	cfg := aidialogue.ProcessConfig{
		MaxResponseTokens: 150,
		PreviousResponse:  "a b c d e f g h i j",
	}
	result := aidialogue.ProcessResponse("a b c d e f g h i j", cfg)
	if !result.NeedsRetry {
		t.Error("exactly 10 tokens with identical text should trigger NeedsRetry")
	}
}

func TestProcessResponse_AC13_RepetitionDetection_ShortResponse_Skipped(t *testing.T) {
	cfg := aidialogue.ProcessConfig{MaxResponseTokens: 150, PreviousResponse: "Yes."}
	result := aidialogue.ProcessResponse("Yes.", cfg)
	if result.NeedsRetry {
		t.Error("<10 token response should skip repetition detection")
	}
}

// ---- AC-14: truncation at last complete sentence boundary ----

func TestProcessResponse_AC14_Truncation_AtSentenceBoundary(t *testing.T) {
	// Word-count truncation (no sentence boundaries).
	var sb strings.Builder
	for i := 0; i < 200; i++ {
		sb.WriteString("Word ")
	}
	result := aidialogue.ProcessResponse(sb.String(), aidialogue.ProcessConfig{MaxResponseTokens: 150})
	if wc := len(strings.Fields(result.Text)); wc != 150 {
		t.Errorf("expected exactly 150 words after truncation, got %d", wc)
	}

	// Sentence-boundary truncation: only the first sentence fits.
	sentences := "One two three four five six seven eight. " +
		"Alpha beta gamma delta epsilon zeta eta theta. " +
		"Red blue green yellow black white brown gold."
	cfg := aidialogue.ProcessConfig{MaxResponseTokens: 12}
	result2 := aidialogue.ProcessResponse(sentences, cfg)
	if !strings.Contains(result2.Text, "eight.") {
		t.Errorf("expected first sentence preserved, got %q", result2.Text)
	}
	if strings.Contains(result2.Text, "theta") || strings.Contains(result2.Text, "gold") {
		t.Errorf("text truncated too late: %q", result2.Text)
	}
}

func TestProcessResponse_AC14_Truncation_UnderLimit_PassesThrough(t *testing.T) {
	cfg := aidialogue.ProcessConfig{MaxResponseTokens: 150}
	raw := "A short response under the limit."
	result := aidialogue.ProcessResponse(raw, cfg)
	if result.Text != raw {
		t.Errorf("under-limit response should pass through unchanged, got %q", result.Text)
	}
}

func TestProcessResponse_AC14_Truncation_DefaultTokens(t *testing.T) {
	// MaxResponseTokens=0 should use the default (150).
	cfg := aidialogue.ProcessConfig{MaxResponseTokens: 0}
	var sb strings.Builder
	for i := 0; i < 200; i++ {
		sb.WriteString("token ")
	}
	result := aidialogue.ProcessResponse(sb.String(), cfg)
	if wc := len(strings.Fields(result.Text)); wc > 150 {
		t.Errorf("default 150 tokens enforced: got %d words", wc)
	}
}

// ---- AC-30: sentence buffer — hallucination stripped BEFORE display ----

func TestProcessResponse_AC30_SentenceBuffer_StripsBeforeDisplay(t *testing.T) {
	reg := newRegistry("Greta", "Orin")
	cfg := aidialogue.ProcessConfig{EntityRegistry: reg, MaxResponseTokens: 150}

	// Single-sentence hallucination — player never sees the name.
	raw := "The ore comes from Shadowforge."
	result := aidialogue.ProcessResponse(raw, cfg)
	if strings.Contains(result.Text, "Shadowforge") {
		t.Errorf("player should never see hallucinated name: %q", result.Text)
	}
	if len(result.SentencesStripped) != 1 || len(result.Hallucinations) == 0 {
		t.Errorf("expected stripped sentence + logged hallucination, got stripped=%d hall=%v",
			len(result.SentencesStripped), result.Hallucinations)
	}

	// Multi-sentence: only the bad sentence is stripped.
	raw2 := "Greta is the blacksmith. The Shadowforge is deep underground. Orin guards the gate."
	result2 := aidialogue.ProcessResponse(raw2, cfg)
	if strings.Contains(result2.Text, "Shadowforge") {
		t.Errorf("hallucinated sentence should be stripped: %q", result2.Text)
	}
	if !strings.Contains(result2.Text, "Greta") || !strings.Contains(result2.Text, "Orin") {
		t.Errorf("valid sentences should be preserved: %q", result2.Text)
	}
	if len(result2.SentencesStripped) != 1 {
		t.Errorf("expected 1 sentence stripped, got %d", len(result2.SentencesStripped))
	}
}

// ---- Edge case: knows_not gating ----

func TestProcessResponse_KnowsNot_PositiveClaimStripped(t *testing.T) {
	reg := newRegistry("Greta", "Theras").withKnowsNot("Theras")
	cfg := aidialogue.ProcessConfig{EntityRegistry: reg, MaxResponseTokens: 150}

	raw := "Theras knows the secret of the old empire ruins."
	result := aidialogue.ProcessResponse(raw, cfg)
	if strings.Contains(result.Text, "Theras") {
		t.Errorf("positive claim about knows_not entity should be stripped: %q", result.Text)
	}
}

func TestProcessResponse_KnowsNot_DeflectionPreserved(t *testing.T) {
	reg := newRegistry("Greta", "Theras", "Orin").withKnowsNot("Theras")
	cfg := aidialogue.ProcessConfig{EntityRegistry: reg, MaxResponseTokens: 150}

	raw := "I don't know about Theras you should ask Orin."
	result := aidialogue.ProcessResponse(raw, cfg)
	if !strings.Contains(result.Text, "Theras") {
		t.Errorf("deflection about knows_not entity should be preserved: %q", result.Text)
	}
}

// ---- Edge case: player-whitelisted names ----

func TestProcessResponse_PlayerNames_ExemptFromStripping(t *testing.T) {
	reg := newRegistry("Greta", "Orin")
	cfg := aidialogue.ProcessConfig{EntityRegistry: reg, MaxResponseTokens: 150, PlayerNames: []string{"Shadowforge"}}

	raw := "You mentioned Shadowforge but I know nothing of it."
	result := aidialogue.ProcessResponse(raw, cfg)
	if !strings.Contains(result.Text, "Shadowforge") {
		t.Errorf("player-whitelisted name should be preserved: %q", result.Text)
	}
}

// ---- Edge case: ambiguous fuzzy match → leave unchanged ----

func TestProcessResponse_AmbiguousFuzzyMatch_LeavesUnchanged(t *testing.T) {
	reg := newRegistry("Cara", "Cora") // both distance 1 from "Cira"
	cfg := aidialogue.ProcessConfig{EntityRegistry: reg, MaxResponseTokens: 150}

	raw := "Cira told me about the forge."
	result := aidialogue.ProcessResponse(raw, cfg)
	if !strings.Contains(result.Text, "Cira") {
		t.Errorf("ambiguous match should leave name unchanged: %q", result.Text)
	}
}

// ---- Edge case: nil registry passes through ----

func TestProcessResponse_NilRegistry_PassesThrough(t *testing.T) {
	cfg := aidialogue.ProcessConfig{MaxResponseTokens: 150}
	raw := "The Shadowforge is deep beneath the mountain."
	result := aidialogue.ProcessResponse(raw, cfg)
	if !strings.Contains(result.Text, "Shadowforge") || len(result.Hallucinations) != 0 {
		t.Errorf("nil registry should pass through without hallucination logging: %q", result.Text)
	}
}

// ---- Edge case: token stripping ----

func TestProcessResponse_StripTokens_RemovesMarkdownAndChatFormat(t *testing.T) {
	cfg := aidialogue.ProcessConfig{MaxResponseTokens: 150}
	raw := "**Greta** is the <|eot_id|> blacksmith. She works *iron*."
	result := aidialogue.ProcessResponse(raw, cfg)
	if strings.Contains(result.Text, "**") || strings.Contains(result.Text, "<|eot_id|>") {
		t.Errorf("markdown and chat tokens should be stripped: %q", result.Text)
	}
}

// ---- Edge case: empty previous → skip repetition ----

func TestProcessResponse_EmptyPrevious_SkipsRepetitionDetection(t *testing.T) {
	cfg := aidialogue.ProcessConfig{MaxResponseTokens: 150, PreviousResponse: ""}
	result := aidialogue.ProcessResponse("The ore is from the old mine beyond the ironwood grove.", cfg)
	if result.NeedsRetry {
		t.Error("empty previous response should skip repetition detection")
	}
}

// ---- Edge case: HardMaxTokens (200) enforced ----

func TestProcessResponse_HardMaxTokens_Enforced(t *testing.T) {
	cfg := aidialogue.ProcessConfig{MaxResponseTokens: 300} // above hard cap
	var sb strings.Builder
	for i := 0; i < 250; i++ {
		sb.WriteString("token ")
	}
	result := aidialogue.ProcessResponse(sb.String(), cfg)
	if wc := len(strings.Fields(result.Text)); wc != 200 {
		t.Errorf("hard max 200 tokens enforced: got %d words", wc)
	}
}

// ---- Edge case: fallback with empty FallbackFact → default placeholder ----

func TestProcessResponse_EmptyFallbackFact_DefaultPlaceholder(t *testing.T) {
	cfg := aidialogue.ProcessConfig{FallbackFact: "", FallbackNPCName: "Greta"}
	result := aidialogue.ProcessResponse("", cfg)
	if result.Text != "..." {
		t.Errorf("empty FallbackFact should produce '...', got %q", result.Text)
	}
}
