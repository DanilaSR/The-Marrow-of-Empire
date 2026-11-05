// Package questions implements the Question System's topic matcher (Story
// Q-001, TR-q-001, TR-q-002): free-text player questions are scored against
// an NPC's domain keyword sets, with entity reference detection, to
// determine which knowledge domains to query and which facts to route
// (GDD CR-2, ADR-0005).
//
// The matcher is a pure function — no I/O, no global state, no engine
// dependencies. Every input (question, NPC domains, entity name index,
// conversation history, tuning config) is passed in, and the same input
// always produces the same MatchResult. The caller — the question routing
// layer — queries the Knowledge Graph (ADR-0005) with the result and emits
// the EventQuestionMatched event (ADR-0011).
package questions

import (
	"strings"

	"marrow-of-empire/entity"
)

// Domain couples one knowledge domain (e.g. "craft_metallurgy") with the
// keyword set used to detect player interest in it (GDD CR-2). Keyword sets
// are data-driven: DefaultKeywords holds the standard nine, and per-NPC
// overrides are authored in the NPC's .kg file (ADR-0005).
type Domain struct {
	// Name is the domain identifier (e.g. "craft_metallurgy"), matching
	// KnowledgeGraph.Domains and Fact.Domain (ADR-0005).
	Name string
	// Keywords are the words and phrases that indicate a question is about
	// this domain. Matching is case-insensitive, whole-word; entries may be
	// multi-word phrases (e.g. "how long").
	Keywords []string
}

// DomainScore is the relevance of one domain to the question (CR-2).
type DomainScore struct {
	// Domain is the scored domain's name.
	Domain string
	// Score is the relevance: (matched_keywords / total_keywords) plus the
	// entity reference bonus when the question mentions a known entity.
	Score float64
	// Keywords lists the domain keywords that matched the question, in
	// authoring order. Used to explain a match and to prioritize facts.
	Keywords []string
	// FromContext reports that the domain matched via conversation history
	// (CR-3 pronoun resolution) rather than directly from the question text.
	FromContext bool
}

// MatchResult describes which domains matched the question and at what
// relevance (CR-2). It is the routing payload for fact queries: matched
// domains → KnowledgeStore.QueryFacts; on fallback, all domains are queried
// at low relevance with a deflection instruction.
type MatchResult struct {
	// MatchedDomains are the domains at or above the relevance threshold,
	// ranked by score descending (authoring order breaks ties). When
	// IsFallback, every NPC domain is present at low relevance.
	MatchedDomains []DomainScore
	// EntityRefs are the GameEntityIDs detected in the question text,
	// deduplicated, in order of first mention. Facts referencing these
	// entities are prioritized regardless of domain (CR-2 step 1).
	EntityRefs []entity.GameEntityID
	// IsFallback is true when no domain matched above the threshold and the
	// question could not be resolved from conversation history. The caller
	// then routes all facts at the current trust level with a deflection
	// instruction (CR-2 step 3).
	IsFallback bool
}

// EntityNameIndex maps entity display names to GameEntityIDs for entity
// reference detection (CR-2 step 1). It is an interface so the matcher
// stays pure: the production implementation wraps the game-layer entity
// registry (ADR-0001), while tests inject a stub.
type EntityNameIndex interface {
	// Lookup reports the GameEntityID registered under the given name. The
	// matcher passes the name folded to lower case; implementations must
	// match case-insensitively (fold names when building the index).
	Lookup(name string) (entity.GameEntityID, bool)
}

// QuestionRecord is one entry of the conversation history kept for CR-3
// question context: the last 3 questions (any NPC). The matcher resolves
// unresolved pronouns ("it", "that", "they") against these records.
type QuestionRecord struct {
	// Question is the raw text of the question the player asked.
	Question string
	// Domains are the domains that matched that question (its
	// MatchResult.MatchedDomains names), used for carry-forward resolution.
	Domains []string
	// Entities are the entity references detected in that question (its
	// MatchResult.EntityRefs), carried forward when the prior question was
	// itself resolved from context.
	Entities []entity.GameEntityID
}

// Config parameterizes the topic matcher. All values are tuning knobs
// (GDD tuning table); a zero-value Config selects the defaults from
// DefaultConfig, so callers override only the fields they care about.
type Config struct {
	// RelevanceThreshold is the minimum score a domain needs to be matched
	// (CR-2). Default 0.3; GDD tuning range 0.1–0.6.
	RelevanceThreshold float64
	// EntityBonus is the value granted to every domain when the question
	// mentions any known entity (CR-2). Default 1.0.
	EntityBonus float64
	// EntityBonusWeight scales the entity bonus in the score formula:
	// score = keyword_matches/total_keywords + EntityBonus × EntityBonusWeight.
	// Default 0.5.
	EntityBonusWeight float64
	// MaxEntityNameWords caps the n-gram scanning window for multi-word
	// entity names (CR-2). Default 4.
	MaxEntityNameWords int
	// Pronouns are the words that trigger CR-3 resolution against the
	// conversation history. Default: it, that, this, they, them, he, she,
	// him, her.
	Pronouns []string
	// MaxHistoryRecords is how many prior QuestionRecords are consulted for
	// pronoun resolution (CR-3: "last 3 questions"). Default 3.
	MaxHistoryRecords int
}

// DefaultConfig returns the canonical tuning values from the GDD (CR-2,
// CR-3 and the tuning table: relevance_threshold 0.3, range 0.1–0.6).
func DefaultConfig() Config {
	return Config{
		RelevanceThreshold: 0.3,
		EntityBonus:        1.0,
		EntityBonusWeight:  0.5,
		MaxEntityNameWords: 4,
		Pronouns:           []string{"it", "that", "this", "they", "them", "he", "she", "him", "her"},
		MaxHistoryRecords:  3,
	}
}

// normalized returns cfg with every zero field replaced by its DefaultConfig
// value, and Pronouns folded to lower case for whole-word comparison.
func (cfg Config) normalized() Config {
	def := DefaultConfig()
	if cfg.RelevanceThreshold == 0 {
		cfg.RelevanceThreshold = def.RelevanceThreshold
	}
	if cfg.EntityBonus == 0 {
		cfg.EntityBonus = def.EntityBonus
	}
	if cfg.EntityBonusWeight == 0 {
		cfg.EntityBonusWeight = def.EntityBonusWeight
	}
	if cfg.MaxEntityNameWords == 0 {
		cfg.MaxEntityNameWords = def.MaxEntityNameWords
	}
	if len(cfg.Pronouns) == 0 {
		cfg.Pronouns = def.Pronouns
	}
	if cfg.MaxHistoryRecords == 0 {
		cfg.MaxHistoryRecords = def.MaxHistoryRecords
	}
	folded := make([]string, len(cfg.Pronouns))
	for i, p := range cfg.Pronouns {
		folded[i] = strings.ToLower(p)
	}
	cfg.Pronouns = folded
	return cfg
}

// EventQuestionMatched fires when a player question has been topic-matched
// and routed to the AI Dialogue Engine (control manifest Feature Layer:
// question matched events via EventBus, ADR-0011). The question routing
// layer emits it after calling MatchTopics and querying the Knowledge Graph
// — the matcher itself is a pure function and never emits. The payload is
// defined by the routing story; it carries the question and MatchResult.
const EventQuestionMatched entity.EventType = "question.matched"
