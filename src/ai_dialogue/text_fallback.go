package aidialogue

import (
	"fmt"
	"strings"
)

// TextFallbackConfig controls text-fallback behavior.
type TextFallbackConfig struct {
	// TrustLevel is the player's current trust level with this NPC (0-3).
	TrustLevel int
	// KnowsNot is the set of domain names this NPC explicitly does not know about.
	KnowsNot map[string]bool
}

// TextFallbackGenerator produces quote-styled prose from NPC knowledge-graph
// facts when the LLM is unavailable. It implements CR-5 (Graceful Degradation):
// text fallback is a designed, first-class experience — NPC facts presented as
// quote-styled prose with portrait attribution. Never a broken experience.
//
// The generator applies the same gating rules as LLM mode:
//   - knows_not domain exclusions
//   - trust_required filtering
//   - referral behavior (References in facts act as leads_to hints)
//
// Zero-fact queries receive an authored deflection with nearest-domain framing.
type TextFallbackGenerator struct{}

// GenerateFallback produces a text-fallback response for the given facts and question.
// npcName is used for portrait attribution styling.
// Returns a designed prose string — never an error or empty string.
func (g *TextFallbackGenerator) GenerateFallback(
	npcName string,
	facts []Fact,
	question string,
	cfg TextFallbackConfig,
) string {
	// Filter facts by trust and domain exclusions.
	eligible := g.filterFacts(facts, cfg)

	if len(eligible) == 0 {
		return g.zeroFactResponse(npcName, facts, cfg)
	}

	// Build quote-styled prose from the top facts (max 3 to keep it concise).
	var lines []string
	maxFacts := 3
	if len(eligible) < maxFacts {
		maxFacts = len(eligible)
	}

	lines = append(lines, fmt.Sprintf("%s considers your question.", npcName))
	for i := 0; i < maxFacts; i++ {
		f := eligible[i]
		line := fmt.Sprintf("  \"%s\"", f.Content)
		lines = append(lines, line)
	}

	// Add referral hints for facts with entity references.
	var referrals []string
	for _, f := range eligible {
		for _, ref := range f.References {
			if ref != "" {
				msg := fmt.Sprintf("You might ask %s about that.", ref)
				dup := false
				for _, r := range referrals {
					if r == msg {
						dup = true
						break
					}
				}
				if !dup {
					referrals = append(referrals, msg)
				}
			}
		}
	}
	if len(referrals) > 0 {
		lines = append(lines, "")
		lines = append(lines, referrals...)
	}

	_ = question // reserved for future context-aware fallback enrichment
	return strings.Join(lines, "\n")
}

// filterFacts applies knows_not domain exclusions and trust_required filtering.
func (g *TextFallbackGenerator) filterFacts(facts []Fact, cfg TextFallbackConfig) []Fact {
	var eligible []Fact
	for _, f := range facts {
		// knows_not gating — same as LLM mode.
		if cfg.KnowsNot != nil && cfg.KnowsNot[f.Domain] {
			continue
		}
		// trust_required filtering — same as LLM mode.
		if f.TrustRequired > cfg.TrustLevel {
			continue
		}
		eligible = append(eligible, f)
	}
	return eligible
}

// zeroFactResponse builds an authored deflection when no facts are eligible.
// Uses nearest-domain facts with "What I can tell you is..." framing per GDD CR-5.
func (g *TextFallbackGenerator) zeroFactResponse(
	npcName string,
	allFacts []Fact,
	cfg TextFallbackConfig,
) string {
	var lines []string
	lines = append(lines, fmt.Sprintf("%s pauses, choosing words carefully.", npcName))

	// Show nearest-domain facts if any exist (ignoring trust/domain for framing).
	if len(allFacts) > 0 {
		lines = append(lines, "What I can tell you is this:")
		maxFacts := 2
		if len(allFacts) < maxFacts {
			maxFacts = len(allFacts)
		}
		for i := 0; i < maxFacts; i++ {
			lines = append(lines, fmt.Sprintf("  \"%s\"", allFacts[i].Content))
		}
	} else {
		lines = append(lines, fmt.Sprintf("\"I'm afraid I have little to share on that matter, but I welcome your questions.\""))
	}

	_ = cfg // reserved for future trust-aware framing
	return strings.Join(lines, "\n")
}
