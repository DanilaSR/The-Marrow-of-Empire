// response_processor.go implements the LLM response post-processor (Story 003;
// TR-ai-004; GDD CR-4; ADR-0008): a deterministic pure function that strips
// stop tokens, resolves entity names with fuzzy matching, detects hallucinations,
// buffers sentences for safe display, truncates at sentence boundaries, and
// detects repetition against the immediate predecessor.
//
// Note: multi-word n-gram matching and pronoun/role substitution (GDD CR-4
// implementation notes) are deferred to a later story. This module handles
// single-word proper nouns with Levenshtein ≤2 fuzzy correction.
package aidialogue

import (
	"fmt"
	"regexp"
	"strings"
)

// EntityRegistry provides entity-name lookup for hallucination detection and
// fuzzy correction (GDD CR-4 step 2: Entity Name Resolution).
type EntityRegistry interface {
	IsRegistered(name string) bool
	AllNames() []string
	KnowsNotEntities() []string // entities the NPC does NOT know about (ADR-0005)
}

// ProcessConfig holds the tunables for a single ProcessResponse call.
type ProcessConfig struct {
	MaxResponseTokens int            // soft cap; 0 → DefaultMaxResponseTokens (150)
	EntityRegistry    EntityRegistry // nil disables entity validation
	PlayerNames       []string       // player-introduced names, exempt from stripping (Story 002)
	PreviousResponse  string         // preceding processed response for repetition detection
	FallbackFact      string         // highest-relevance fact for empty/all-stripped fallback
	FallbackNPCName   string         // NPC name for quote-styled fallback attribution
}

// ProcessResult is the outcome of a single ProcessResponse call.
type ProcessResult struct {
	Text              string   // final processed response, ready for typewriter display
	Hallucinations    []string // unknown terms that triggered stripping
	NearMisses        []string // fuzzy corrections in "candidate→canonical" form
	IsFallback        bool     // fallback fact used (empty/garbage/all-stripped)
	NeedsRetry        bool     // >85% Jaccard → caller should re-query once with rep penalty
	SentencesStripped []string // sentences removed before display (AC-30)
}

const (
	DefaultMaxResponseTokens = 150
	HardMaxTokens            = 200

	minTokensForRepetition = 10
	jaccardRetryThreshold  = 0.85
	levenshteinMaxDist     = 2
)

// ---- regex & stopword data ----

var (
	stopTokenRE  = regexp.MustCompile(`<\|eot_id\|>|<\|end_of_text\|>|<\|begin_of_text\|>`)
	chatTokenRE  = regexp.MustCompile(`<\|[a-z_]+(?:\|[a-z_]+)*\|>`)
	markdownRE   = regexp.MustCompile(`\*{1,3}|_{1,3}|` + "`" + `{1,3}`)
	properNounRE = regexp.MustCompile(`\b[A-Z][a-z]{1,}\b`)
)

var nounStopwords = map[string]bool{
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
	"well": true, "oh": true, "ah": true, "aye": true,
	"look": true, "listen": true, "come": true, "go": true, "get": true,
	"see": true, "know": true, "think": true, "want": true, "need": true,
	"let": true, "us": true, "now": true, "here": true,
}

// ---- ProcessResponse (public API) ----

// ProcessResponse post-processes raw LLM output through the full pipeline
// (GDD CR-4): stripping → entity resolution → sentence-buffered hallucination
// removal → truncation → repetition detection. Pure function — no I/O.
func ProcessResponse(raw string, cfg ProcessConfig) ProcessResult {
	cleaned := stripResponseTokens(raw)
	cleaned = strings.TrimSpace(cleaned)
	if cleaned == "" {
		return buildFallback(cfg)
	}

	sentences := splitSentences(cleaned)
	if len(sentences) == 0 {
		return buildFallback(cfg)
	}

	playerSet := strSet(cfg.PlayerNames)
	knowsNotSet := strSet(knowsNotList(cfg.EntityRegistry))
	allNames := allRegNames(cfg.EntityRegistry)

	var result ProcessResult
	kept := processSentenceList(sentences, cfg.EntityRegistry, allNames, playerSet, knowsNotSet, &result)
	if len(kept) == 0 {
		result.IsFallback = true
		result.Text = fallbackText(cfg)
		return result
	}

	result.Text = strings.Join(kept, " ")
	if cfg.MaxResponseTokens <= 0 {
		result.Text = truncateAtSentence(result.Text, DefaultMaxResponseTokens)
	} else {
		result.Text = truncateAtSentence(result.Text, cfg.MaxResponseTokens)
	}

	if cfg.PreviousResponse != "" && wordCount(result.Text) >= minTokensForRepetition {
		if tokenJaccard(result.Text, cfg.PreviousResponse) > jaccardRetryThreshold {
			result.NeedsRetry = true
		}
	}

	return result
}

// ---- pipeline helpers ----

func strSet(ss []string) map[string]bool {
	if len(ss) == 0 {
		return nil
	}
	out := make(map[string]bool, len(ss))
	for _, s := range ss {
		out[strings.ToLower(s)] = true
	}
	return out
}

func knowsNotList(reg EntityRegistry) []string {
	if reg == nil {
		return nil
	}
	return reg.KnowsNotEntities()
}

func allRegNames(reg EntityRegistry) []string {
	if reg == nil {
		return nil
	}
	return reg.AllNames()
}

func processSentenceList(
	sentences []string,
	reg EntityRegistry,
	allNames []string,
	playerSet, knowsNotSet map[string]bool,
	result *ProcessResult,
) []string {
	var kept []string
	for _, s := range sentences {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		processed, sr := processSentence(s, reg, allNames, playerSet, knowsNotSet)
		if sr.stripped {
			result.SentencesStripped = append(result.SentencesStripped, s)
			result.Hallucinations = append(result.Hallucinations, sr.hallucinations...)
		} else {
			result.NearMisses = append(result.NearMisses, sr.nearMisses...)
			kept = append(kept, processed)
		}
	}
	return kept
}

// ---- stripping ----

func stripResponseTokens(s string) string {
	s = stopTokenRE.ReplaceAllString(s, "")
	s = chatTokenRE.ReplaceAllString(s, "")
	s = markdownRE.ReplaceAllString(s, "")
	return strings.Join(strings.Fields(s), " ")
}

// ---- sentence splitting ----

func splitSentences(text string) []string {
	if strings.TrimSpace(text) == "" {
		return nil
	}
	var out []string
	start := 0
	for i := 0; i < len(text); i++ {
		b := text[i]
		if b != '.' && b != '!' && b != '?' {
			continue
		}
		j := i + 1
		for j < len(text) && (text[j] == ' ' || text[j] == '\t' || text[j] == '\n') {
			j++
		}
		if j < len(text) && text[j] >= 'A' && text[j] <= 'Z' {
			out = append(out, text[start:i+1])
			start = j
			i = j - 1
		}
	}
	if start < len(text) {
		out = append(out, text[start:])
	}
	return out
}

// ---- Levenshtein distance ----

func levenshtein(a, b string) int {
	ar, br := []rune(a), []rune(b)
	n, m := len(ar), len(br)
	if n == 0 {
		return m
	}
	if m == 0 {
		return n
	}
	// Length-difference gate: if the absolute length difference exceeds
	// maxDist, the distance can never be ≤ maxDist.
	if abs(n-m) > levenshteinMaxDist {
		return levenshteinMaxDist + 1
	}
	prev := make([]int, m+1)
	cur := make([]int, m+1)
	for j := 0; j <= m; j++ {
		prev[j] = j
	}
	for i := 1; i <= n; i++ {
		cur[0] = i
		for j := 1; j <= m; j++ {
			cost := 0
			if ar[i-1] != br[j-1] {
				cost = 1
			}
			cur[j] = min3(prev[j]+1, cur[j-1]+1, prev[j-1]+cost)
		}
		prev, cur = cur, prev
	}
	return prev[m]
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func min3(a, b, c int) int {
	if b < a {
		a = b
	}
	if c < a {
		return c
	}
	return a
}

// ---- Jaccard similarity ----

func tokenJaccard(a, b string) float64 {
	ta := tokenSet(a)
	tb := tokenSet(b)
	if len(ta) == 0 && len(tb) == 0 {
		return 1.0
	}
	intersection := 0
	for t := range ta {
		if tb[t] {
			intersection++
		}
	}
	union := len(ta) + len(tb) - intersection
	return float64(intersection) / float64(union)
}

func tokenSet(s string) map[string]bool {
	out := make(map[string]bool)
	for _, w := range strings.Fields(s) {
		out[strings.ToLower(w)] = true
	}
	return out
}

func wordCount(s string) int { return len(strings.Fields(s)) }

// ---- proper noun extraction ----

func extractProperNouns(sentence string, playerNames map[string]bool) []string {
	matches := properNounRE.FindAllString(sentence, -1)
	seen := make(map[string]bool)
	var out []string
	for _, m := range matches {
		lower := strings.ToLower(m)
		if nounStopwords[lower] || (playerNames != nil && playerNames[lower]) || seen[lower] {
			continue
		}
		seen[lower] = true
		out = append(out, m)
	}
	return out
}

// ---- entity resolution ----

// fuzzyMatch returns the closest registry name and its distance.
// When multiple candidates tie, returns ("", dist) to signal ambiguity.
func fuzzyMatch(name string, candidates []string) (string, int) {
	best := ""
	bestDist := levenshteinMaxDist + 1
	ambiguous := false
	for _, c := range candidates {
		d := levenshtein(name, c)
		if d < bestDist {
			bestDist = d
			best = c
			ambiguous = false
		} else if d == bestDist {
			ambiguous = true
		}
	}
	if ambiguous {
		return "", bestDist
	}
	return best, bestDist
}

// replaceWord replaces whole-word occurrences of old with replacement in s.
// Uses a word-boundary regex to avoid corrupting longer words (e.g. replacing
// "Gretta"→"Greta" must not turn "Grettabelle" into "Gretabelle").
func replaceWord(s, old, replacement string) string {
	re := regexp.MustCompile(`\b` + regexp.QuoteMeta(old) + `\b`)
	return re.ReplaceAllString(s, replacement)
}

// ---- knows-not gating ----

func isDeflectionOrNegation(s string) bool {
	lower := strings.ToLower(s)
	for _, p := range []string{
		"don't know", "do not know", "i'm not sure", "i am not sure",
		"never heard", "can't say", "cannot say", "no idea",
		"don't deal", "not my", "beyond me", "above my",
		"haven't heard", "have not heard", "nothing to do with",
		"not something i", "i don't", "wish i could", "if only i knew",
		"can't help", "cannot help", "ask", "talk to", "speak to",
	} {
		if strings.Contains(lower, p) {
			return true
		}
	}
	return false
}

// ---- per-sentence processing ----

type sentenceResult struct {
	stripped       bool
	hallucinations []string
	nearMisses     []string
}

// processSentence applies entity resolution to one sentence and reports
// whether it should be stripped before display. allNames is the pre-fetched
// result of EntityRegistry.AllNames() (hoisted to avoid per-noun calls).
func processSentence(
	sentence string,
	reg EntityRegistry,
	allNames []string,
	playerNames, knowsNot map[string]bool,
) (string, sentenceResult) {
	var sr sentenceResult
	if reg == nil {
		return sentence, sr
	}
	nouns := extractProperNouns(sentence, playerNames)
	resolved := sentence
	for _, n := range nouns {
		lower := strings.ToLower(n)

		// Knows-not: strip positive claims, preserve negations/deflections.
		if knowsNot != nil && knowsNot[lower] && !isDeflectionOrNegation(sentence) {
			sr.stripped = true
			sr.hallucinations = append(sr.hallucinations, n)
			return sentence, sr
		}

		if reg.IsRegistered(n) {
			continue
		}

		// Pre-fetched allNames; nil means no registry configured.
		if allNames == nil {
			continue
		}
		best, dist := fuzzyMatch(n, allNames)
		if dist <= levenshteinMaxDist && best != "" {
			sr.nearMisses = append(sr.nearMisses, fmt.Sprintf("%s→%s", n, best))
			resolved = replaceWord(resolved, n, best)
			continue
		}
		// Ambiguous match → leave unchanged (implementation notes).
		if dist <= levenshteinMaxDist {
			continue
		}

		// No match → hallucination → strip.
		sr.stripped = true
		sr.hallucinations = append(sr.hallucinations, n)
		return sentence, sr
	}
	return resolved, sr
}

// ---- truncation ----

func truncateAtSentence(text string, maxTokens int) string {
	if maxTokens > HardMaxTokens {
		maxTokens = HardMaxTokens
	}
	if wordCount(text) <= maxTokens {
		return text
	}
	lastKeep := 0
	for i := 0; i < len(text); i++ {
		b := text[i]
		if b != '.' && b != '!' && b != '?' {
			continue
		}
		j := i + 1
		for j < len(text) && (text[j] == ' ' || text[j] == '\t' || text[j] == '\n') {
			j++
		}
		if j < len(text) && text[j] >= 'A' && text[j] <= 'Z' {
			end := i + 1
			if wordCount(text[:end]) > maxTokens {
				break
			}
			lastKeep = end
			i = j - 1
		}
	}
	if lastKeep > 0 {
		return text[:lastKeep]
	}
	return truncateByWords(text, maxTokens)
}

func truncateByWords(text string, maxTokens int) string {
	words := strings.Fields(text)
	if len(words) <= maxTokens {
		return text
	}
	return strings.Join(words[:maxTokens], " ")
}

// ---- fallback ----

func buildFallback(cfg ProcessConfig) ProcessResult {
	return ProcessResult{IsFallback: true, Text: fallbackText(cfg)}
}

func fallbackText(cfg ProcessConfig) string {
	if cfg.FallbackFact == "" {
		return "..."
	}
	name := cfg.FallbackNPCName
	if name == "" {
		name = "NPC"
	}
	return fmt.Sprintf("\"%s\" — %s", cfg.FallbackFact, name)
}
