package investigation_journal

import (
	"sort"

	"marrow-of-empire/thread_investigation"
)

// FactEntry is a display-ready fact row for the fact log tab.
type FactEntry struct {
	FactID        string   // fact identifier from NPC Knowledge Graph
	Content       string   // display text
	SourceNPC     uint64   // GameEntityID of the NPC who revealed this fact
	TrustLevel    int      // trust level at time of discovery (0-3)
	TrustLabel    string   // human-readable trust level
	DiscoveredAt  float64  // game timestamp
	AgeDisplay    string   // formatted age (computed by caller with game clock)
	Contradicts   bool     // true if this fact contradicts another
	EntityRefs    []uint64 // GameEntityIDs referenced in this fact
}

// FactGroup is a group of facts from the same source NPC.
type FactGroup struct {
	SourceNPC  uint64
	NPCName    string // resolved from Entity Framework by caller
	TrustLevel int    // highest trust level among facts in this group
	Facts      []FactEntry
}

// BuildFactLog extracts facts for a given chain from PlayerKnowledge, groups
// them by source NPC, and sorts within each group by discovery time (oldest first).
// Groups are sorted by NPC name for stable display order.
func BuildFactLog(pk *thread_investigation.PlayerKnowledge, chainID string) []FactGroup {
	if pk == nil {
		return nil
	}

	// Collect fact IDs belonging to this chain.
	chainFactIDs := make(map[string]bool)
	for _, ch := range pk.Chains {
		if ch.ID == chainID {
			for _, fid := range ch.Facts {
				chainFactIDs[fid] = true
			}
			break
		}
	}
	if len(chainFactIDs) == 0 {
		return nil
	}

	// Build a fact lookup: FactID → DiscoveredFact.
	factLookup := make(map[string]*thread_investigation.DiscoveredFact, len(pk.DiscoveredFacts))
	for i := range pk.DiscoveredFacts {
		factLookup[pk.DiscoveredFacts[i].FactID] = &pk.DiscoveredFacts[i]
	}

	// Check which facts are contradicted.
	contradicted := make(map[string]bool, len(pk.ContradictedFacts))
	for _, fid := range pk.ContradictedFacts {
		contradicted[fid] = true
	}

	// Group facts by source NPC.
	groupMap := make(map[uint64][]FactEntry)
	for fid := range chainFactIDs {
		df, ok := factLookup[fid]
		if !ok {
			continue
		}
		entry := FactEntry{
			FactID:       fid,
			SourceNPC:    uint64(df.SourceNPC),
			DiscoveredAt: df.DiscoveredAt,
			Contradicts:  contradicted[fid],
		}
		groupMap[entry.SourceNPC] = append(groupMap[entry.SourceNPC], entry)
	}

	// Build sorted groups.
	groups := make([]FactGroup, 0, len(groupMap))
	for npcID, facts := range groupMap {
		// Sort facts within group by discovery time (oldest first).
		sort.Slice(facts, func(i, j int) bool {
			return facts[i].DiscoveredAt < facts[j].DiscoveredAt
		})

		// Determine highest trust level in group.
		maxTrust := 0
		for _, f := range facts {
			if f.TrustLevel > maxTrust {
				maxTrust = f.TrustLevel
			}
		}

		groups = append(groups, FactGroup{
			SourceNPC:  npcID,
			TrustLevel: maxTrust,
			Facts:      facts,
		})
	}

	// Sort groups by SourceNPC for stable display order.
	sort.Slice(groups, func(i, j int) bool {
		return groups[i].SourceNPC < groups[j].SourceNPC
	})

	return groups
}

// TrustLevelLabel returns a human-readable trust level string.
func TrustLevelLabel(level int) string {
	switch level {
	case 0:
		return "Stranger"
	case 1:
		return "Acquaintance"
	case 2:
		return "Trusted"
	case 3:
		return "Confidant"
	default:
		return "Unknown"
	}
}
