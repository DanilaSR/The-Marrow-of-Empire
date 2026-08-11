// Query engine for the NPC Knowledge Graph (Story KG-003, TR-kg-003).
// QueryFacts combines the three fact filters — trust gating, domain, and
// discovery triggers — into a single query for the AI Dialogue Engine's
// prompt construction, capped at a prompt-budget maximum.
package knowledge

// DefaultMaxResults is the cap on facts returned by QueryFacts — the AI
// Dialogue Engine's prompt budget (ADR-0005): at most 5 matching facts feed
// an LLM prompt so the 2048-token context window is not overflowed.
const DefaultMaxResults = 5

// QueryConfig parameterizes a QueryFacts call (Story KG-003, TR-kg-003).
type QueryConfig struct {
	// TrustLevel is the NPC's current trust toward the player (0-3,
	// TR-trust-001). Facts whose TrustRequired exceeds it are excluded.
	TrustLevel int
	// Domain optionally restricts results to one knowledge domain (exact
	// match on Fact.Domain). Empty means no restriction — facts from all
	// domains are returned.
	Domain string
	// SatisfiedTriggers lists the discovery triggers the player has
	// satisfied (e.g. "player_has_item:ley_metal_fragment"). A fact is
	// returned only when every one of its DiscoveryTriggers appears here.
	SatisfiedTriggers []string
	// MaxResults caps how many facts are returned, in authoring order.
	// Zero uses DefaultMaxResults (the 5-fact prompt budget).
	MaxResults int
}

// QueryFacts returns the facts an NPC will share under the given query
// (Story KG-003, TR-kg-003). A fact is included only when all of:
//
//   - trust gating: fact.TrustRequired <= cfg.TrustLevel (via IsGated)
//   - domain: cfg.Domain == "" || fact.Domain == cfg.Domain
//   - discovery: every fact.DiscoveryTriggers entry is in
//     cfg.SatisfiedTriggers (a fact with no triggers always passes)
//
// Results preserve authoring order and are capped at cfg.MaxResults facts
// (DefaultMaxResults when zero) — the AI Dialogue Engine's prompt budget
// (ADR-0005). Returns nil when no fact matches, or when graph is nil.
func QueryFacts(graph *KnowledgeGraph, cfg QueryConfig) []Fact {
	if graph == nil {
		return nil
	}
	limit := cfg.MaxResults
	if limit == 0 {
		limit = DefaultMaxResults
	}
	satisfied := make(map[string]struct{}, len(cfg.SatisfiedTriggers))
	for _, t := range cfg.SatisfiedTriggers {
		satisfied[t] = struct{}{}
	}

	var out []Fact
	for _, f := range graph.Facts {
		if IsGated(f, cfg.TrustLevel) {
			continue
		}
		if cfg.Domain != "" && f.Domain != cfg.Domain {
			continue
		}
		if !triggersSatisfied(f.DiscoveryTriggers, satisfied) {
			continue
		}
		out = append(out, f)
		if len(out) >= limit {
			break
		}
	}
	return out
}

// triggersSatisfied reports whether every required trigger is present in the
// satisfied set. A fact with no required triggers is always satisfied
// (vacuous truth).
func triggersSatisfied(required []string, satisfied map[string]struct{}) bool {
	for _, t := range required {
		if _, ok := satisfied[t]; !ok {
			return false
		}
	}
	return true
}
