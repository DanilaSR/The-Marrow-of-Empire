// Topic matching algorithm (GDD CR-2, CR-3; Story Q-001):
//
//  1. Entity reference detection — the question text is scanned for entity
//     names (NPCs, locations, items, factions) via case-insensitive,
//     whole-word, n-gram probing against the EntityNameIndex. Detected
//     entities become EntityRefs and add the entity bonus to every domain's
//     score.
//  2. Domain keyword matching — each NPC domain scores
//     (matched_keywords / total_keywords) + entity_bonus × 0.5.
//  3. Threshold — domains at ≥ 0.3 are matched, ranked by score descending.
//     When none qualify, the result falls back to all domains at low
//     relevance with IsFallback set.
//  4. Question context (CR-3) — when the question contains a pronoun
//     ("it", "that", "they", ...) and direct scoring matched nothing, the
//     last 3 history questions are consulted: the question is re-scored
//     together with the prior question texts, and domains that matched a
//     prior question are carried forward (marked FromContext).
package questions

import (
	"sort"
	"strings"
	"unicode"

	"marrow-of-empire/entity"
)

// MatchTopics scores the player's question against an NPC's domain keyword
// sets (CR-2) with default tuning and no conversation history. It is a pure
// function — no I/O, no global state; all data is passed in. See
// MatchTopicsWithContext for configurable tuning and CR-3 resolution.
func MatchTopics(question string, npcDomains []Domain, entityIndex EntityNameIndex) MatchResult {
	return MatchTopicsWithContext(question, npcDomains, entityIndex, nil, Config{})
}

// MatchTopicsWithContext scores the question like MatchTopics, additionally
// applying CR-3 pronoun resolution against history (the last 3 questions,
// any NPC) and cfg tuning. A zero-value cfg selects the DefaultConfig
// values; pass nil history when no conversation context exists.
func MatchTopicsWithContext(question string, npcDomains []Domain, entityIndex EntityNameIndex, history []QuestionRecord, cfg Config) MatchResult {
	cfg = cfg.normalized()

	tokens := tokenize(question)
	refs := detectEntities(tokens, entityIndex, cfg.MaxEntityNameWords)
	scores := scoreDomains(tokens, npcDomains, refs, cfg)

	// CR-3: unresolved pronouns inherit the prior topic context when
	// direct scoring matched nothing above the threshold.
	if !anyAboveThreshold(scores, cfg) && hasPronoun(tokens, cfg.Pronouns) && len(history) > 0 {
		return resolveFromHistory(question, npcDomains, entityIndex, history, cfg)
	}

	return finalize(scores, refs, cfg)
}

// tokenize splits text into lower-cased word tokens. Any non-letter rune
// (unicode-aware) is a word boundary, so punctuation and digits split words
// and "Ley-touched" becomes "ley", "touched".
func tokenize(text string) []string {
	var tokens []string
	var cur []rune
	flush := func() {
		if len(cur) > 0 {
			tokens = append(tokens, strings.ToLower(string(cur)))
			cur = cur[:0]
		}
	}
	for _, r := range text {
		if unicode.IsLetter(r) {
			cur = append(cur, r)
		} else {
			flush()
		}
	}
	flush()
	return tokens
}

// detectEntities finds entity names in the question text by n-gram probing
// (CR-2 step 1). At each token position the longest phrase (up to maxWords)
// the index knows wins, and scanning skips past it — so overlapping names
// never double-report and the longest name is preferred. IDs are
// deduplicated, in order of first mention. A nil index is an empty index.
func detectEntities(tokens []string, index EntityNameIndex, maxWords int) []entity.GameEntityID {
	if index == nil || len(tokens) == 0 {
		return nil
	}
	var refs []entity.GameEntityID
	seen := make(map[entity.GameEntityID]bool)
	for i := 0; i < len(tokens); {
		matched := false
		maxN := maxWords
		if remaining := len(tokens) - i; maxN > remaining {
			maxN = remaining
		}
		for n := maxN; n >= 1; n-- {
			id, ok := index.Lookup(strings.Join(tokens[i:i+n], " "))
			if !ok {
				continue
			}
			if !seen[id] {
				seen[id] = true
				refs = append(refs, id)
			}
			i += n
			matched = true
			break
		}
		if !matched {
			i++
		}
	}
	return refs
}

// scoreDomains scores every NPC domain against the question tokens (CR-2
// step 2): score = (matched_keywords / total_keywords) + entity_bonus ×
// entity_bonus_weight, where the entity bonus is granted once when any
// entity reference was detected. A domain with an empty keyword set scores
// the entity bonus alone. Keywords are matched case-insensitively, whole
// word, and each keyword counts at most once.
func scoreDomains(tokens []string, npcDomains []Domain, refs []entity.GameEntityID, cfg Config) []DomainScore {
	hasEntity := len(refs) > 0
	scores := make([]DomainScore, 0, len(npcDomains))
	for _, d := range npcDomains {
		matched := matchKeywords(tokens, d.Keywords)
		score := 0.0
		if len(d.Keywords) > 0 {
			score = float64(len(matched)) / float64(len(d.Keywords))
		}
		if hasEntity {
			score += cfg.EntityBonus * cfg.EntityBonusWeight
		}
		scores = append(scores, DomainScore{Domain: d.Name, Score: score, Keywords: matched})
	}
	return scores
}

// matchKeywords returns the keywords whose folded word sequence appears as
// a contiguous run of question tokens. Single-word keywords match as whole
// tokens ("smith" does not match "blacksmith"); multi-word keywords
// ("how long") match as consecutive tokens.
func matchKeywords(tokens []string, keywords []string) []string {
	var matched []string
	for _, kw := range keywords {
		seq := strings.Fields(strings.ToLower(kw))
		if len(seq) > 0 && containsRun(tokens, seq) {
			matched = append(matched, kw)
		}
	}
	return matched
}

// containsRun reports whether seq appears as a contiguous run in tokens
// (both are already folded to lower case).
func containsRun(tokens, seq []string) bool {
	if len(seq) > len(tokens) {
		return false
	}
outer:
	for i := 0; i+len(seq) <= len(tokens); i++ {
		for j := range seq {
			if tokens[i+j] != seq[j] {
				continue outer
			}
		}
		return true
	}
	return false
}

// hasPronoun reports whether any token is one of the CR-3 pronoun words
// (folded). A question that names its topic directly ("that forge") still
// contains a pronoun but never reaches resolution when it scores above
// threshold — see MatchTopicsWithContext.
func hasPronoun(tokens []string, pronouns []string) bool {
	for _, t := range tokens {
		for _, p := range pronouns {
			if t == p {
				return true
			}
		}
	}
	return false
}

// anyAboveThreshold reports whether any domain score reaches the threshold.
func anyAboveThreshold(scores []DomainScore, cfg Config) bool {
	for _, s := range scores {
		if s.Score >= cfg.RelevanceThreshold {
			return true
		}
	}
	return false
}

// finalize ranks the scores (descending, stable so authoring order breaks
// ties) and splits them into matched vs. fallback (CR-2 step 3). Domains
// matched via conversation history (FromContext) count as matched even
// though their direct score is below the threshold.
func finalize(scores []DomainScore, refs []entity.GameEntityID, cfg Config) MatchResult {
	sorted := make([]DomainScore, len(scores))
	copy(sorted, scores)
	sort.SliceStable(sorted, func(i, j int) bool { return sorted[i].Score > sorted[j].Score })

	matched := make([]DomainScore, 0, len(sorted))
	for _, s := range sorted {
		if s.Score >= cfg.RelevanceThreshold || s.FromContext {
			matched = append(matched, s)
		}
	}
	if len(matched) > 0 {
		return MatchResult{MatchedDomains: matched, EntityRefs: refs}
	}
	return MatchResult{MatchedDomains: sorted, EntityRefs: refs, IsFallback: true}
}

// resolveFromHistory answers a question whose pronouns ("it", "that",
// "they", ...) refer back to the prior conversation (CR-3). The question
// text is re-scored together with the last MaxHistoryRecords question texts
// (most recent first), entity names are re-scanned across that combined
// text, and domains that matched a prior question are carried forward —
// only when the NPC actually has that domain. Carried domains are marked
// FromContext. When nothing resolves, the result falls back like any other
// unmatched question.
func resolveFromHistory(question string, npcDomains []Domain, entityIndex EntityNameIndex, history []QuestionRecord, cfg Config) MatchResult {
	start := len(history) - cfg.MaxHistoryRecords
	if start < 0 {
		start = 0
	}

	// Re-score over the question plus the prior question texts, most
	// recent first.
	tokens := tokenize(question)
	for i := len(history) - 1; i >= start; i-- {
		tokens = append(tokens, tokenize(history[i].Question)...)
	}
	refs := detectEntities(tokens, entityIndex, cfg.MaxEntityNameWords)

	// Carry entity references from records whose entity names do not
	// appear in their text (records resolved from an earlier context).
	seen := make(map[entity.GameEntityID]bool, len(refs))
	for _, id := range refs {
		seen[id] = true
	}
	for i := len(history) - 1; i >= start; i-- {
		for _, id := range history[i].Entities {
			if !seen[id] {
				seen[id] = true
				refs = append(refs, id)
			}
		}
	}

	scores := scoreDomains(tokens, npcDomains, refs, cfg)

	// Carry forward prior matched domains, most recent first, only when
	// the NPC has the domain and the augmented re-score did not already
	// match it.
	have := make(map[string]bool, len(npcDomains))
	for _, s := range scores {
		if s.Score >= cfg.RelevanceThreshold {
			have[s.Domain] = true
		}
	}
	for i := len(history) - 1; i >= start; i-- {
		for _, name := range history[i].Domains {
			if have[name] {
				continue
			}
			have[name] = true
			for j := range scores {
				if scores[j].Domain == name {
					scores[j].FromContext = true
					break
				}
			}
		}
	}

	return finalize(scores, refs, cfg)
}
