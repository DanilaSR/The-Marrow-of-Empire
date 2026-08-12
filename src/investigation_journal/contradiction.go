package investigation_journal

import "marrow-of-empire/thread_investigation"

// ContradictionPair represents two facts that contradict each other.
type ContradictionPair struct {
	FactA    FactEntry
	FactB    FactEntry
	Topic    string // e.g. "ore quality"
}

// FindContradictions scans a chain's facts and returns pairs that contradict
// each other. Contradicted facts are flagged in PlayerKnowledge.ContradictedFacts.
// Pairs are formed between the first contradicted fact and each subsequent
// contradicted fact in the same chain.
func FindContradictions(pk *thread_investigation.PlayerKnowledge, chainID string) []ContradictionPair {
	if pk == nil {
		return nil
	}

	// Collect contradicted facts in this chain.
	var facts []FactEntry
	for _, f := range pk.DiscoveredFacts {
		for _, cid := range pk.ContradictedFacts {
			if f.FactID == cid {
				facts = append(facts, FactEntry{
					FactID:     f.FactID,
					SourceNPC:  uint64(f.SourceNPC),
					DiscoveredAt: f.DiscoveredAt,
				})
				break
			}
		}
	}
	if len(facts) < 2 {
		return nil
	}

	// Group contradicted facts by chain membership.
	chainFacts := make(map[string]bool)
	for _, ch := range pk.Chains {
		if ch.ID == chainID {
			for _, fid := range ch.Facts {
				chainFacts[fid] = true
			}
			break
		}
	}

	var chainContradicted []FactEntry
	for _, f := range facts {
		if chainFacts[f.FactID] {
			chainContradicted = append(chainContradicted, f)
		}
	}
	if len(chainContradicted) < 2 {
		return nil
	}

	// Form pairs: first contradicted fact paired with each subsequent one.
	pairs := make([]ContradictionPair, 0, len(chainContradicted)-1)
	for i := 1; i < len(chainContradicted); i++ {
		pairs = append(pairs, ContradictionPair{
			FactA: chainContradicted[0],
			FactB: chainContradicted[i],
			Topic: "contradiction", // resolved from fact content in later story
		})
	}
	return pairs
}

// RevelationPrompt holds the data for a revelation prompt (CR-6).
type RevelationPrompt struct {
	ChainID      string
	Conclusion   string // e.g. "Aldric was murdered by the guild"
	TargetNPC    uint64 // GameEntityID of NPC to confront
	TargetName   string // resolved by caller
	Dismissed    bool   // true if dismissed this session
}

// CheckRevelationReady returns true if the chain has reached revelation threshold
// (3+ connected facts, status = revelation_ready per ADR-0010).
func CheckRevelationReady(pk *thread_investigation.PlayerKnowledge, chainID string) bool {
	if pk == nil {
		return false
	}
	for _, ch := range pk.Chains {
		if ch.ID == chainID && ch.Status == thread_investigation.ChainStatusRevelationReady {
			return true
		}
	}
	return false
}

// BuildRevelationPrompt creates a revelation prompt for a chain that has reached
// the threshold. Returns nil if the chain is not revelation_ready.
func BuildRevelationPrompt(pk *thread_investigation.PlayerKnowledge, chainID string) *RevelationPrompt {
	if !CheckRevelationReady(pk, chainID) {
		return nil
	}
	// Find the revelation record for this chain.
	for _, rev := range pk.Revelations {
		if rev.ChainID == chainID && rev.ConfrontedAt == 0 {
			return &RevelationPrompt{
				ChainID:    chainID,
				Conclusion: "evidence linking a conclusion", // placeholder; real text from Revelation
			}
		}
	}
	return nil
}

// DismissedRevelations tracks which chains have had their revelation prompt
// dismissed in the current journal session. Dismissals reset when the journal
// closes (not persisted).
type DismissedRevelations struct {
	dismissed map[string]bool
}

// NewDismissedRevelations returns an empty dismissal tracker.
func NewDismissedRevelations() *DismissedRevelations {
	return &DismissedRevelations{dismissed: make(map[string]bool)}
}

// Dismiss marks a chain's revelation prompt as dismissed for this session.
func (d *DismissedRevelations) Dismiss(chainID string) {
	d.dismissed[chainID] = true
}

// IsDismissed returns true if the chain's revelation was dismissed this session.
func (d *DismissedRevelations) IsDismissed(chainID string) bool {
	return d.dismissed[chainID]
}

// Reset clears all dismissals (called when journal closes).
func (d *DismissedRevelations) Reset() {
	d.dismissed = make(map[string]bool)
}
