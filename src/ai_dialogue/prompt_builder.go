// prompt_builder.go implements the AI Dialogue Engine's prompt constructor
// (Story 002; TR-ai-002; GDD CR-2 Prompt Template, CR-3 Prompt Assembly;
// ADR-0008): a deterministic pure function that builds a Llama 3.2 Instruct
// chat-format prompt from NPC Knowledge Graph data, trust-filtered facts,
// the player's question, conversation history, and relationship context.
//
// BuildPrompt performs no I/O and holds no global state — every input
// arrives via PromptConfig, so the same config always produces the same
// prompt string. The two-tier structure (GDD CR-2, revised 2026-08-09)
// splits the prompt into a static, cacheable core (Character, Exemplars,
// Knowledge, Who You Know, Relationship, Rules) and a per-query dynamic
// suffix (Conversation So Far, What You've Already Discussed, Player
// Question).
//
// Token counting uses a word-count estimate (1 token ≈ 0.75 words for
// English) as an MVP simplification; the pinned tokenizer.json extracted
// from the authoritative GGUF (GDD Open Questions) is engine-programmer
// work and can replace EstimateTokens without touching the assembly logic.
package aidialogue

import (
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"
)

// Prompt budgets and limits (GDD Formulas § Token Budget, CR-3 steps 5/10).
const (
	// DefaultFactsTokenBudget is the Knowledge section budget (GDD: S_facts
	// ~300–600 tokens; budget-constrained, not count-constrained).
	DefaultFactsTokenBudget = 600
	// DefaultMaxPromptTokens is the effective context window guard (GDD:
	// effective window 4096). The assembled prompt must never exceed it.
	DefaultMaxPromptTokens = 4096
	// MaxQuestionRunes caps the player question at 500 characters (GDD CR-3
	// step 10: length-capped at 500 chars, ~125 tokens).
	MaxQuestionRunes = 500
	// MaxReferralEntries is the roster truncation floor used by the overflow
	// cascade (Story 002 implementation notes).
	MaxReferralEntries = 5
	// questionTruncationMarker is appended when a question is truncated.
	// The marker counts toward MaxQuestionRunes, so the truncated question
	// never exceeds the cap (GDD: "length-capped at 500 characters").
	questionTruncationMarker = "[...]"
)

// InstructionDeflectionPrompt is the authored deflection substituted for
// instruction-phrased player input (GDD CR-3 step 10, Edge Cases). The
// injection does not reach the model as an instruction; the NPC answers in
// character with confusion and the generation proceeds normally. This is a
// quality-of-illusion guard, not a security boundary — a determined player
// can still jailbreak a 3B model (accepted risk, single-player game).
const InstructionDeflectionPrompt = `[The player's words make no sense to you. Stay in character and respond with confusion, like "I'm not sure what you mean by that. What are you actually asking?"]`

// rulesText is the load-bearing Rules block (GDD CR-2: "Rules (3 rules near
// generation point) | Hardcoded in template"). Pinned verbatim to the GDD
// template so the Knowledge boundary and fact-grounding constraints sit
// closest to the generation point.
const rulesText = `- You know ONLY what is listed in your Knowledge section. If asked about something not in your Knowledge, say so and suggest who might know from Who You Know.
- For RUMOR facts, hedge your statements. For SECRET facts, be reluctant and lower your voice. For OPINION facts, state it as your personal view.
- Never break character. Never reference game mechanics, stats, or that you are an AI.`

// emptyKnowledgeDeflection is rendered into the Knowledge section when no
// facts match the question (GDD CR-3 "Empty fact set").
const emptyKnowledgeDeflection = "You have no specific knowledge about this. Deflect naturally and suggest who might know from Who You Know."

// FactType is the presentation category of a fact (ADR-0005 CR-2). The
// type selects the tag rendered in the Knowledge section.
type FactType string

const (
	// FactTypeFact is a verified, confident statement.
	FactTypeFact FactType = "FACT"
	// FactTypeRumor is second-hand or unverified; the NPC hedges.
	FactTypeRumor FactType = "RUMOR"
	// FactTypeOpinion is subjective; the NPC states it as a personal view.
	FactTypeOpinion FactType = "OPINION"
	// FactTypeSecret is sensitive, trust-gated knowledge; the NPC is
	// reluctant and lowers their voice.
	FactTypeSecret FactType = "SECRET"
)

// Fact is one piece of NPC knowledge as seen by the prompt builder. Calling
// systems (Knowledge Graph, Trust, Question) map their data into this shape;
// the builder never queries them (pure function). Facts are matched against
// the question's matched domains and detected entity references.
type Fact struct {
	// FactID is the stable identifier within the NPC's graph (e.g.
	// "greta_ley_metal_001"). Used for tie-breaking and deduplication.
	FactID string
	// Domain is the knowledge domain this fact belongs to; it must be one of
	// the NPC's domains.
	Domain string
	// Type selects the Knowledge-section tag ([FACT], [RUMOR — ...], etc).
	Type FactType
	// Content is the natural-language statement of the fact, fed to the
	// prompt without transformation.
	Content string
	// TrustRequired is the minimum trust level (0-3) needed to learn this
	// fact; facts above the current trust level are excluded.
	TrustRequired int
	// References are canonical identifiers (GameEntityID strings or entity
	// names — any stable string, consistently with DetectedEntityRefs) of
	// entities this fact is about.
	References []string
	// DiscoveryTriggers are conditions that unlock this fact; a fact with
	// triggers is excluded unless its FactID is in PromptConfig.UnlockedFactIDs.
	DiscoveryTriggers []string
	// Priority is the authored importance of the fact; higher values are
	// selected first within the same relevance tier (CR-3 step 5).
	Priority int
}

// NPC is the character identity and knowledge boundary data for prompt
// construction.
type NPC struct {
	// ID is the unique NPC identifier (cache-key component).
	ID string
	// Name is the display name (e.g. "Greta").
	Name string
	// Role is the NPC's occupation (e.g. "blacksmith").
	Role string
	// Personality is a natural-language description of the NPC's character.
	Personality string
	// SpeechStyle is a natural-language description of how the NPC talks.
	SpeechStyle string
	// KnowsNot lists knowledge domains the NPC explicitly does not know
	// (knowledge boundaries, ADR-0005 CR-5); rendered as "You do NOT know
	// about: ..." in the referral roster.
	KnowsNot []string
}

// Exemplar is one authored few-shot Q&A pair showing the NPC's voice
// (GDD CR-2: 2–3 pairs per archetype; the strongest voice anchor for a
// ~3B model).
type Exemplar struct {
	PlayerQuestion string
	NPCResponse    string
}

// Referral is one entry of the "Who You Know" roster: a person the NPC can
// point the player to, with a one-line hint about what to ask there.
type Referral struct {
	// Name is the resolved display name (the calling system resolves
	// GameEntityIDs; the builder performs no lookups).
	Name string
	// Role is a compact role description.
	Role string
	// Hint is a one-line topic hint (leads_to.topic_hint) or domain summary
	// (knows_who entries).
	Hint string
}

// ReferralRoster is the "Who You Know" roster data (CR-3 step 6): leads_to
// hints collected from all of the NPC's facts plus the authored knows_who
// list. The caller flattens both into these fields.
type ReferralRoster struct {
	// LeadsTo are referral hints pointing to other entities.
	LeadsTo []Referral
	// KnowsWho are authored roster entries.
	KnowsWho []Referral
}

// RelationshipContext carries the Trust & Relationship metadata (CR-3
// step 9) that colors the Relationship section.
type RelationshipContext struct {
	// VouchedBy are the names of NPCs who vouched for the player to this
	// NPC (e.g. "Orin"); rendered as "Orin spoke well of them. You trust
	// Orin's judgment."
	VouchedBy []string
	// BetrayalHistory is a list of pre-authored narrative lines describing
	// past betrayals (rendered verbatim, e.g. "They once shared your secret
	// with the wrong person. Be guarded.").
	BetrayalHistory []string
}

// DisclosedFact is one fact already revealed in this conversation (CR-6);
// rendered as a brief label in "What You've Already Discussed".
type DisclosedFact struct {
	// FactID is the NPC-graph fact identifier.
	FactID string
	// Label is a brief natural-language label (e.g. "the strange ore from
	// the old mine").
	Label string
}

// PromptConfig carries every input the prompt builder needs. All data comes
// from calling systems (Knowledge Graph, Trust & Relationship, Question
// System, Story 004 state machine) — the builder is a pure function with no
// I/O and no global state. Zero-valued budgets fall back to the GDD
// defaults; everything else is used verbatim.
type PromptConfig struct {
	// NPC is the character identity and knowledge boundaries.
	NPC NPC
	// Facts is the NPC's full fact list (all trust levels); the builder
	// applies trust gating, trigger gating, deduplication, and the
	// budget-constrained priority selection (CR-3 steps 4-5).
	Facts []Fact
	// MatchedDomains are the question's matched domains from the Question
	// System; index 0 is the primary domain, the rest are secondary.
	MatchedDomains []string
	// DetectedEntityRefs are the entity references detected in the player's
	// question (Question System), in the same canonical form as
	// Fact.References.
	DetectedEntityRefs []string
	// IsFallback reports the Question System's no-match fallback
	// (MatchResult.IsFallback: no domain matched above the threshold).
	// In fallback mode every trust-visible fact is routed at low relevance
	// (tier 4) instead of being excluded, so the NPC can answer with the
	// knowledge they have even when the question matched nothing.
	IsFallback bool
	// Exemplars are the authored few-shot Q&A pairs for this archetype.
	Exemplars []Exemplar
	// ReferralRoster is the "Who You Know" roster data.
	ReferralRoster ReferralRoster
	// TrustLevel is the current trust level (0-3) with this NPC.
	TrustLevel int
	// TrustDescription is the trust tier name (e.g. "stranger").
	TrustDescription string
	// Relationship carries vouched_by / betrayal_history context.
	Relationship RelationshipContext
	// RollingSummary is the conversation summary so far (CR-6); omitted
	// from the prompt when empty (first exchange).
	RollingSummary string
	// DisclosedFacts are the facts already revealed this conversation;
	// omitted from the prompt when empty.
	DisclosedFacts []DisclosedFact
	// Question is the raw player question; it is sanitized (chat-format
	// token stripping, instruction detection, 500-char cap) by the builder.
	Question string
	// UnlockedFactIDs marks facts whose discovery triggers are satisfied
	// (e.g. "player_has_item:X"); facts with DiscoveryTriggers are excluded
	// unless listed here.
	UnlockedFactIDs map[string]bool
	// FactsTokenBudget caps the Knowledge section (default 600).
	FactsTokenBudget int
	// MaxPromptTokens caps the total prompt (default 4096).
	MaxPromptTokens int
}

// EstimateTokens estimates the token count of s using the MVP word-count
// rule 1 token ≈ 0.75 words, i.e. ceil(words × 4/3). Deterministic. This is
// a simplification until the pinned tokenizer.json fixture (GDD Open
// Questions) replaces it.
func EstimateTokens(s string) int {
	words := len(strings.Fields(s))
	return (words*4 + 2) / 3
}

// SanitizeQuestion applies CR-3 step 10 input sanitization: chat-format
// tokens (<|begin_of_text|>, <|eot_id|>, <|start_header_id|>,
// <|end_header_id|>, and role-carrying header blocks) are stripped, and
// instruction-phrased input (prompt-injection patterns like "ignore your
// instructions") is detected. The second return value reports whether the
// question matched an instruction pattern and must be routed to the
// authored deflection (InstructionDeflectionPrompt).
func SanitizeQuestion(q string) (string, bool) {
	clean := stripChatTokens(q)
	return clean, instructionPattern.MatchString(clean)
}

// ExtractPlayerIntroducedNames whitelists proper-noun candidates the player
// typed that are not in the entity registry (CR-3 step 10). The response
// post-processor (Story 003) exempts these from hallucination stripping —
// the NPC may echo and deflect names the player introduced. knownNames is
// the caller's entity registry names (matched case-insensitively); matches
// are excluded, as are common sentence-initial words (heuristic; multi-word
// names arrive as separate candidates for the post-processor to re-join).
// Returns unique candidates in first-seen order.
func ExtractPlayerIntroducedNames(question string, knownNames []string) []string {
	known := make(map[string]bool, len(knownNames))
	for _, k := range knownNames {
		known[strings.ToLower(k)] = true
	}
	seen := make(map[string]bool)
	var out []string
	for _, m := range properNounPattern.FindAllString(question, -1) {
		key := strings.ToLower(m)
		if known[key] || playerNameStopwords[key] || seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, m)
	}
	return out
}

// BuildStaticCore renders the cacheable static core of the prompt — the
// system header, Character, Exemplars, Knowledge, Who You Know, Relationship
// and Rules blocks (GDD CR-2). It is invariant for a fixed (NPC identity,
// trust level, exemplar variant, matched domains, detected entity refs),
// which makes it the prompt-prefix cache unit for llama-server KV reuse;
// the caller owns caching (Story 004/006).
func BuildStaticCore(cfg PromptConfig) (string, error) {
	if err := validateConfig(cfg); err != nil {
		return "", err
	}
	return renderStaticCore(cfg, selectFacts(cfg), rosterEntryCount(cfg.ReferralRoster)), nil
}

// BuildPrompt constructs a full Llama 3.2 Instruct chat-format prompt
// (GDD CR-2 template) from cfg. It is a pure function — no I/O, no global
// state; the same config always produces the same prompt. Construction
// completes well within the 5ms budget (GDD AC-24).
//
// Assembly (CR-3): the static core is rendered with the trust-filtered,
// budget-constrained fact selection, then the dynamic suffix (Conversation
// So Far, What You've Already Discussed, Player Question) is appended. The
// player question is sanitized (chat-format token stripping, instruction
// deflection, 500-char cap). If the estimated total exceeds
// MaxPromptTokens, the overflow cascade runs in priority order: drop the
// lowest-priority fact, trim the summary to one sentence, truncate the
// roster to MaxReferralEntries. If the floor is still over budget the
// prompt is returned as-is — the cascade is exhaustive.
func BuildPrompt(cfg PromptConfig) (string, error) {
	if err := validateConfig(cfg); err != nil {
		return "", err
	}

	question, deflected := SanitizeQuestion(cfg.Question)
	question = truncateRunes(question, MaxQuestionRunes)
	if deflected {
		question = InstructionDeflectionPrompt
	}

	facts := append([]Fact(nil), selectFacts(cfg)...)
	rosterLimit := rosterEntryCount(cfg.ReferralRoster)
	summary := cfg.RollingSummary
	maxTokens := effectiveMaxTokens(cfg)

	for {
		prompt := renderStaticCore(cfg, facts, rosterLimit) + renderSuffix(summary, cfg.DisclosedFacts, question)
		if EstimateTokens(prompt) <= maxTokens {
			return prompt, nil
		}
		switch {
		case len(facts) > 0:
			facts = facts[:len(facts)-1] // drop the lowest-priority fact
		case summary != "":
			summary = firstSentence(summary)
		case rosterLimit > MaxReferralEntries:
			rosterLimit = MaxReferralEntries
		default:
			return prompt, nil // exhaustive cascade — nothing left to drop
		}
	}
}

// --- internal helpers ---

var (
	// headerBlockPattern removes role-carrying header blocks such as
	// <|start_header_id|>system<|end_header_id|> (CR-3 step 10: "and their
	// attribute variants").
	headerBlockPattern = regexp.MustCompile(`<\|start_header_id\|>[^<]*?<\|end_header_id\|>`)
	// chatTokenPattern removes standalone chat-format tokens.
	chatTokenPattern = regexp.MustCompile(`<\|[a-z_]+(?:\|[a-z_]+)*\|>`)
	// instructionPattern detects instruction-phrased player input
	// (CR-3 step 10): "ignore your (previous )?instructions",
	// "you are (actually|now)", "forget (everything|your)", "act as",
	// "pretend you are", "your (new |real )?(rules|system prompt)".
	instructionPattern = regexp.MustCompile(`(?i)ignore your (previous )?instructions|you are (actually|now)|forget (everything|your)|act as|pretend you are|your (new |real )?(rules|system prompt)`)
	// properNounPattern extracts capitalized words as candidate
	// player-introduced names for the post-processor whitelist.
	properNounPattern = regexp.MustCompile(`\b[A-Z][a-z]+\b`)
)

// playerNameStopwords are common sentence-initial words that are not names;
// the whitelist heuristic excludes them so "Ask Greta about Orin" yields
// only "Orin". The post-processor still filters candidates against the
// entity registry (CR-3 step 10).
var playerNameStopwords = map[string]bool{
	"about": true, "after": true, "and": true, "are": true, "ask": true,
	"before": true, "but": true, "can": true, "could": true, "did": true,
	"do": true, "does": true, "for": true, "from": true, "had": true,
	"has": true, "have": true, "hello": true, "hey": true, "how": true,
	"if": true, "is": true, "it": true, "me": true, "my": true, "no": true,
	"not": true, "please": true, "say": true, "she": true, "should": true,
	"so": true, "tell": true, "than": true, "that": true, "the": true,
	"their": true, "them": true, "then": true, "there": true, "these": true,
	"they": true, "this": true, "those": true, "was": true, "we": true,
	"were": true, "what": true, "when": true, "where": true, "which": true,
	"who": true, "why": true, "will": true, "with": true, "would": true,
	"yes": true, "you": true, "your": true,
}

// stripChatTokens removes chat-format tokens and role header blocks, then
// collapses whitespace so adjacent removed tokens leave a clean gap.
func stripChatTokens(s string) string {
	s = headerBlockPattern.ReplaceAllString(s, "")
	s = chatTokenPattern.ReplaceAllString(s, "")
	return strings.Join(strings.Fields(s), " ")
}

// truncateRunes truncates s to at most max runes (character boundary, never
// a byte boundary — the result is always valid UTF-8), appending
// questionTruncationMarker when truncated. The marker counts toward max.
func truncateRunes(s string, max int) string {
	if utf8.RuneCountInString(s) <= max {
		return s
	}
	runes := []rune(s)
	keep := max - utf8.RuneCountInString(questionTruncationMarker)
	return string(runes[:keep]) + questionTruncationMarker
}

// firstSentence reduces a summary to its first sentence (overflow cascade
// step 2).
func firstSentence(s string) string {
	if i := strings.Index(s, "."); i >= 0 {
		return strings.TrimSpace(s[:i+1])
	}
	return s
}

func effectiveMaxTokens(cfg PromptConfig) int {
	if cfg.MaxPromptTokens > 0 {
		return cfg.MaxPromptTokens
	}
	return DefaultMaxPromptTokens
}

func effectiveFactsBudget(cfg PromptConfig) int {
	if cfg.FactsTokenBudget > 0 {
		return cfg.FactsTokenBudget
	}
	return DefaultFactsTokenBudget
}

func rosterEntryCount(r ReferralRoster) int {
	return len(r.LeadsTo) + len(r.KnowsWho)
}

// validateConfig rejects programmer errors (invalid trust level, empty NPC
// name, negative budgets). It is not a security boundary.
func validateConfig(cfg PromptConfig) error {
	if strings.TrimSpace(cfg.NPC.Name) == "" {
		return errors.New("aidialogue: BuildPrompt: PromptConfig.NPC.Name must not be empty")
	}
	if cfg.TrustLevel < 0 || cfg.TrustLevel > 3 {
		return fmt.Errorf("aidialogue: BuildPrompt: TrustLevel %d out of range 0-3", cfg.TrustLevel)
	}
	if cfg.FactsTokenBudget < 0 || cfg.MaxPromptTokens < 0 {
		return errors.New("aidialogue: BuildPrompt: token budgets must not be negative")
	}
	return nil
}

// selectFacts implements CR-3 steps 4-5: deduplicate by FactID, exclude
// trust-gated facts and trigger-locked facts, classify each fact into a
// relevance tier (both domain+entity → entity-only → primary domain →
// secondary domain), rank within the tier by authored priority then FactID,
// and keep whole facts until the facts token budget is exhausted. The
// result is a pure function of (npc_id, trust_level, matched_domain,
// detected_entity_refs). The budget counts fact content tokens only (GDD:
// S_facts is the facts' own token sum).
func selectFacts(cfg PromptConfig) []Fact {
	seen := make(map[string]bool)
	var ranked []Fact
	for _, f := range cfg.Facts {
		if seen[f.FactID] {
			continue
		}
		seen[f.FactID] = true
		if f.TrustRequired > cfg.TrustLevel {
			continue
		}
		if len(f.DiscoveryTriggers) > 0 && !cfg.UnlockedFactIDs[f.FactID] {
			continue
		}
		if factTier(cfg, f) == 0 {
			continue
		}
		ranked = append(ranked, f)
	}
	sort.SliceStable(ranked, func(i, j int) bool {
		ti, tj := factTier(cfg, ranked[i]), factTier(cfg, ranked[j])
		if ti != tj {
			return ti < tj
		}
		if ranked[i].Priority != ranked[j].Priority {
			return ranked[i].Priority > ranked[j].Priority
		}
		return ranked[i].FactID < ranked[j].FactID
	})

	budget := effectiveFactsBudget(cfg)
	var selected []Fact
	tokens := 0
	for _, f := range ranked {
		t := EstimateTokens(f.Content)
		if tokens+t > budget {
			break // all-or-nothing per fact — no partial truncation
		}
		tokens += t
		selected = append(selected, f)
	}
	return selected
}

// factTier classifies a fact's relevance to the question (CR-3 step 5):
// 1 = domain AND entity reference match, 2 = entity reference only,
// 3 = primary domain match, 4 = secondary domain match, 0 = no match
// (excluded). When PromptConfig.IsFallback is set (Question System no-match
// fallback routes every NPC domain at low relevance), facts without any
// match rank at tier 4 instead of being excluded; entity-matched facts keep
// their tier (entity references are prioritized regardless of domain).
func factTier(cfg PromptConfig, f Fact) int {
	domainMatch := 0 // 0 none, 1 primary, 2 secondary
	if len(cfg.MatchedDomains) > 0 {
		if f.Domain == cfg.MatchedDomains[0] {
			domainMatch = 1
		} else {
			for _, d := range cfg.MatchedDomains[1:] {
				if f.Domain == d {
					domainMatch = 2
					break
				}
			}
		}
	}
	entityMatch := false
	for _, r := range f.References {
		for _, d := range cfg.DetectedEntityRefs {
			if r == d {
				entityMatch = true
				break
			}
		}
		if entityMatch {
			break
		}
	}
	switch {
	case domainMatch != 0 && entityMatch:
		return 1
	case entityMatch:
		return 2
	case domainMatch == 1:
		return 3
	case domainMatch == 2:
		return 4
	case cfg.IsFallback:
		return 4
	default:
		return 0
	}
}

// renderStaticCore renders the system header through the Rules block
// (GDD CR-2). facts is the already-selected fact list; rosterLimit of 0
// means no limit.
func renderStaticCore(cfg PromptConfig, facts []Fact, rosterLimit int) string {
	var b strings.Builder
	b.WriteString("<|begin_of_text|><|start_header_id|>system<|end_header_id|>\n\n")
	b.WriteString("You are ")
	b.WriteString(cfg.NPC.Name)
	if cfg.NPC.Role != "" {
		b.WriteString(", ")
		b.WriteString(cfg.NPC.Role)
	}
	b.WriteString(" in the borderland town of Thornwell.\n")
	b.WriteString("You are a fictional character in a game. You are not an AI assistant. Never break character.\n\n")

	b.WriteString("## Character\n")
	b.WriteString("Personality: ")
	b.WriteString(cfg.NPC.Personality)
	b.WriteString("\nSpeech style: ")
	b.WriteString(cfg.NPC.SpeechStyle)
	b.WriteString("\n\n")

	if ex := renderExemplars(cfg.NPC.Name, cfg.Exemplars); ex != "" {
		b.WriteString(ex)
		b.WriteString("\n\n")
	}

	b.WriteString("## Knowledge\n")
	b.WriteString(renderKnowledge(facts))
	b.WriteString("\n\n## Who You Know\n")
	b.WriteString(renderRoster(cfg.ReferralRoster, rosterLimit))
	if len(cfg.NPC.KnowsNot) > 0 {
		b.WriteString("\nYou do NOT know about: ")
		b.WriteString(strings.Join(cfg.NPC.KnowsNot, ", "))
	}

	b.WriteString("\n\n## Your Relationship With This Person\n")
	b.WriteString("Trust level: ")
	b.WriteString(strconv.Itoa(cfg.TrustLevel))
	b.WriteString(" out of 3")
	if cfg.TrustDescription != "" {
		b.WriteString(" (")
		b.WriteString(cfg.TrustDescription)
		b.WriteString(")")
	}
	b.WriteString("\n")
	b.WriteString(renderRelationship(cfg))

	b.WriteString("\n\n## Rules\n")
	b.WriteString(rulesText)
	return b.String()
}

// renderSuffix renders the dynamic per-query suffix: Conversation So Far,
// What You've Already Discussed, and the player question in the user turn
// (GDD CR-2). Empty sections are omitted (CR-3 steps 7-8).
func renderSuffix(summary string, disclosed []DisclosedFact, question string) string {
	var b strings.Builder
	if summary != "" {
		b.WriteString("\n\n## Conversation So Far\n")
		b.WriteString(summary)
	}
	if len(disclosed) > 0 {
		b.WriteString("\n\n## What You've Already Discussed\n")
		b.WriteString(renderDisclosed(disclosed))
	}
	b.WriteString("\n\n<|eot_id|><|start_header_id|>user<|end_header_id|>\n\n")
	b.WriteString(question)
	b.WriteString("<|eot_id|><|start_header_id|>assistant<|end_header_id|>")
	return b.String()
}

// renderExemplars renders the few-shot Q&A pairs (empty when none).
func renderExemplars(npcName string, ex []Exemplar) string {
	if len(ex) == 0 {
		return ""
	}
	var b strings.Builder
	for i, e := range ex {
		if i > 0 {
			b.WriteString("\n\n")
		}
		b.WriteString("Player: ")
		b.WriteString(e.PlayerQuestion)
		b.WriteString("\n")
		b.WriteString(npcName)
		b.WriteString(": ")
		b.WriteString(e.NPCResponse)
	}
	return b.String()
}

// renderKnowledge renders the Knowledge section fact lines. When no facts
// are selected (out-of-domain question, knows_not topic), the section
// carries the authored empty-fact deflection (GDD CR-3 "Empty fact set").
func renderKnowledge(facts []Fact) string {
	if len(facts) == 0 {
		return emptyKnowledgeDeflection
	}
	var b strings.Builder
	for i, f := range facts {
		if i > 0 {
			b.WriteString("\n")
		}
		b.WriteString("- ")
		b.WriteString(factTag(f.Type))
		b.WriteString(" ")
		b.WriteString(f.Content)
	}
	return b.String()
}

// factTag returns the Knowledge-section tag for a fact type (AC-7: RUMOR
// and SECRET facts carry the authored hedge/reluctance tags).
func factTag(t FactType) string {
	switch t {
	case FactTypeRumor:
		return "[RUMOR — you heard this second-hand and cannot vouch for it]"
	case FactTypeSecret:
		return "[SECRET — revealed only to someone you deeply trust]"
	case FactTypeOpinion:
		return "[OPINION — your personal view]"
	default:
		return "[FACT]"
	}
}

// renderRoster renders the "Who You Know" entries as compact
// name/role/hint lines, limited to limit entries (0 = no limit). Empty
// entries are skipped.
func renderRoster(r ReferralRoster, limit int) string {
	var entries []Referral
	for _, e := range append(append([]Referral(nil), r.LeadsTo...), r.KnowsWho...) {
		if e.Name == "" && e.Role == "" && e.Hint == "" {
			continue
		}
		entries = append(entries, e)
	}
	if limit > 0 && limit < len(entries) {
		entries = entries[:limit]
	}
	var b strings.Builder
	for i, e := range entries {
		if i > 0 {
			b.WriteString("\n")
		}
		b.WriteString(renderReferral(e))
	}
	return b.String()
}

func renderReferral(r Referral) string {
	var parts []string
	if r.Name != "" {
		parts = append(parts, r.Name)
	}
	if r.Role != "" {
		parts = append(parts, r.Role)
	}
	if r.Hint != "" {
		parts = append(parts, r.Hint)
	}
	if len(parts) == 0 {
		return "- (unnamed)"
	}
	return "- " + strings.Join(parts, " — ")
}

// renderRelationship renders the Relationship context narrative (CR-3
// step 9): vouched-for lines, betrayal lines, or a neutral default.
func renderRelationship(cfg PromptConfig) string {
	var b strings.Builder
	for _, v := range cfg.Relationship.VouchedBy {
		b.WriteString(v)
		b.WriteString(" spoke well of them. You trust ")
		b.WriteString(v)
		b.WriteString("'s judgment.\n")
	}
	for _, line := range cfg.Relationship.BetrayalHistory {
		b.WriteString(line)
		b.WriteString("\n")
	}
	if b.Len() == 0 {
		return "No vouching or betrayal history."
	}
	return strings.TrimSuffix(b.String(), "\n")
}

// renderDisclosed renders the disclosed-fact labels (CR-6).
func renderDisclosed(ds []DisclosedFact) string {
	var b strings.Builder
	for i, d := range ds {
		if i > 0 {
			b.WriteString("\n")
		}
		b.WriteString("- ")
		b.WriteString(d.Label)
		b.WriteString(" (")
		b.WriteString(d.FactID)
		b.WriteString(")")
	}
	return b.String()
}
