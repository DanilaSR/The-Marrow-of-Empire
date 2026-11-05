package aidialogue

import (
	"strings"
	"unicode"
)

// QueryClassifier decides whether a prompt is simple enough for the tiny
// CGo model. Per ADR-0008 revised Decision: "When uncertain, return false —
// routing to primary. The cost of a false-positive 'simple' is a thin answer;
// the cost of a false-positive 'complex' is 2s latency. Err toward primary."
type QueryClassifier struct{}

// ClassifyResult carries the optional output of the Question System's
// topic matcher. All fields are optional — the classifier works on the
// prompt alone when no topic-matcher data is available.
type ClassifyResult struct {
	// IsFallback is true when the topic matcher found no domain above threshold.
	IsFallback bool
	// EntityRefCount is the number of entity references detected in the question.
	EntityRefCount int
	// DomainCount is the number of domains that matched above threshold.
	DomainCount int
}

// IsSimple reports whether the given prompt and optional topic-matcher data
// represent a simple conversational turn suitable for the tiny model.
//
// Simple queries (ADR-0008 revised Decision):
//   - IsFallback == true → no domains matched (player asked outside NPC knowledge)
//   - No entity references AND no domains matched → greeting / conversational opener
//   - Short prompt (<5 tokens) with no entity references → pronoun-only follow-up
//
// Everything else routes to the primary model.
func (qc *QueryClassifier) IsSimple(prompt string, cr ClassifyResult) bool {
	// No domains matched — player asked outside NPC knowledge.
	if cr.IsFallback {
		return true
	}

	// No domains AND no entities — greeting or conversational opener.
	// When no topic-matcher metadata is available (IsFallback is false and
	// all counts are zero), require the prompt to be short. A long sentence
	// with zero metadata is unknown territory — be conservative.
	if cr.DomainCount == 0 && cr.EntityRefCount == 0 {
		if estimateTokenCount(prompt) < 8 {
			return true
		}
		return false
	}

	// Short follow-up with no entity references (pronoun-only).
	tokenCount := estimateTokenCount(prompt)
	if tokenCount < 5 && cr.EntityRefCount == 0 {
		return true
	}

	return false
}

// estimateTokenCount returns a rough token count for the prompt.
// Approximation: 1 token ≈ 4 characters for English text.
// This is sufficient for the <5-token short-query check; precision is
// not required — when uncertain, the classifier defaults to complex.
func estimateTokenCount(s string) int {
	// Count word-like sequences (alphanumeric runs).
	wordCount := 0
	inWord := false
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			if !inWord {
				wordCount++
				inWord = true
			}
		} else {
			inWord = false
		}
	}

	// Fallback: character-based estimate.
	charEstimate := len(strings.TrimSpace(s)) / 4
	if charEstimate < 1 {
		charEstimate = 1
	}

	// Use the higher estimate (conservative — prefer complex routing).
	if wordCount > charEstimate {
		return wordCount
	}
	return charEstimate
}
