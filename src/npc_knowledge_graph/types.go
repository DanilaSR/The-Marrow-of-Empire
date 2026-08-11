// Package knowledge implements the NPC Knowledge Graph data model (ADR-0005):
// knowledge domains, fact types with presentation semantics, trust-gated
// facts, entity references, and knowledge boundaries.
//
// This package defines data types only. YAML .kg file parsing (Story 002) and
// the fact query engine with domain/trust/trigger filtering (Story 003) build
// on these types. The Knowledge component stores a KnowledgeGraph per NPC via
// map[GameEntityID]KnowledgeComponent in the Entity Framework (ADR-0001).
package knowledge

import "marrow-of-empire/entity"

// KnowledgeGraph is the parsed .kg file, stored in the Knowledge component
// (ADR-0005). It holds an NPC's knowledge domains, personality and speech
// style for prompt construction, explicit knowledge boundaries (knows_not),
// and the full set of facts the NPC knows.
type KnowledgeGraph struct {
	// NPCID is the unique identifier of the NPC this graph belongs to,
	// matching the .kg file name (e.g. "greta_blacksmith").
	NPCID string `yaml:"npc_id"`
	// NPCName is the display name of the NPC.
	NPCName string `yaml:"npc_name"`
	// Domains are the named categories of expertise this NPC can speak on
	// (e.g. "craft_metallurgy", "trade_commerce"). Domain matching maps a
	// player question to the NPC's relevant facts.
	Domains []string `yaml:"domains"`
	// Personality is a natural-language description of the NPC's character,
	// fed to the AI Dialogue Engine for prompt construction.
	Personality string `yaml:"personality"`
	// SpeechStyle is a natural-language description of how the NPC talks,
	// fed to the AI Dialogue Engine for prompt construction.
	SpeechStyle string `yaml:"speech_style"`
	// KnowsNot lists the knowledge domains the NPC explicitly does not know
	// (knowledge boundaries, ADR-0005 CR-5).
	KnowsNot []string `yaml:"knows_not"`
	// KnowsNotEntities lists GameEntityIDs the NPC explicitly does not know
	// (knowledge boundaries, ADR-0005 CR-5).
	KnowsNotEntities []string `yaml:"knows_not_entities"`
	// Facts are the individual pieces of knowledge the NPC holds, each with
	// its own type, domain, and trust gating.
	Facts []Fact `yaml:"facts"`
}

// FactType is the presentation category of a fact. The type determines how
// the AI Dialogue Engine presents the fact to the player (ADR-0005 CR-3):
// FACT is stated confidently, RUMOR is caveated ("I heard..."), OPINION is
// subjective, and SECRET is shared reluctantly and gated by trust.
type FactType string

const (
	// FactType_Fact is a verified, confident statement. The NPC presents it
	// as certain knowledge.
	FactType_Fact FactType = "FACT"
	// FactType_Rumor is second-hand or unverified. The NPC presents it
	// caveated, e.g. "I heard...".
	FactType_Rumor FactType = "RUMOR"
	// FactType_Opinion is subjective. The NPC presents it as their personal
	// view, not objective fact.
	FactType_Opinion FactType = "OPINION"
	// FactType_Secret is sensitive knowledge shared reluctantly and only when
	// the player's trust level satisfies the fact's TrustRequired gate.
	FactType_Secret FactType = "SECRET"
)

// Fact is a single piece of NPC knowledge (ADR-0005 CR-2). Content is
// authored as natural language so it can feed LLM prompts directly. Facts
// reference other entities by GameEntityID, enabling cross-referencing and
// investigation thread creation (ADR-0010).
type Fact struct {
	// FactID is the unique identifier of this fact within the NPC's graph
	// (e.g. "greta_ley_metal_001"). Contradiction and thread systems
	// reference facts by this ID.
	FactID string `yaml:"fact_id"`
	// Domain is the knowledge domain this fact belongs to; it must be one of
	// the graph's Domains.
	Domain string `yaml:"domain"`
	// Type determines how the AI Dialogue Engine presents the fact:
	// FACT, RUMOR, OPINION, or SECRET.
	Type FactType `yaml:"type"`
	// Content is the natural-language statement of the fact, fed to the LLM
	// prompt without transformation.
	Content string `yaml:"content"`
	// TrustRequired is the minimum trust level (0-3) needed to learn this
	// fact. Facts with TrustRequired greater than the current trust level
	// are excluded from queries. IsGated implements this check.
	TrustRequired int `yaml:"trust_required"` // 0-3
	// References are GameEntityIDs of entities this fact is about; enables
	// automatic cross-referencing and thread creation (ADR-0010).
	References []entity.GameEntityID `yaml:"references"`
	// DiscoveryTriggers are optional conditions that unlock this fact, e.g.
	// "player_has_item:X" or "player_visited:Y". A fact with unsatisfied
	// triggers is excluded from queries.
	DiscoveryTriggers []string `yaml:"discovery_triggers"`
	// Contradicts lists fact_ids in other NPCs' graphs that this fact
	// contradicts; discovered contradictions create investigation threads
	// (ADR-0010).
	Contradicts []string `yaml:"contradicts"`
	// LeadsTo are follow-up leads pointing the player to other entities
	// and the topics worth asking about (ADR-0010).
	LeadsTo []Lead `yaml:"leads_to"`
}

// Lead is a follow-up pointer from a fact to another entity, with a hint
// about what topic to pursue there (ADR-0005, ADR-0010).
type Lead struct {
	// EntityID is the GameEntityID of the entity the lead points to.
	EntityID entity.GameEntityID `yaml:"entity_id"`
	// TopicHint is a natural-language hint for what to ask about, e.g.
	// "Ask the hermit about Ley-touched metal".
	TopicHint string `yaml:"topic_hint"`
}

// IsGated reports whether fact is excluded from a query at the given
// trust level. Facts are accessible when TrustRequired <= trustLevel
// (ADR-0005 CR-4); a fact is gated when its TrustRequired exceeds the
// querying trust level. The fact query engine (Story 003) uses this to
// filter facts before prompt construction.
func IsGated(fact Fact, trustLevel int) bool {
	return fact.TrustRequired > trustLevel
}
