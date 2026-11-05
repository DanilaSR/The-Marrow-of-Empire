package aidialogue

import (
	"slices"
	"strings"
)

// MaxSummaryTokens is the soft cap on the rolling summary length in tokens.
// Older bullet points are dropped when the summary exceeds this limit.
const MaxSummaryTokens = 150

// ConversationMemory tracks the rolling summary and disclosed facts for a
// single NPC conversation. It implements CR-6 (Conversation Memory) from
// the AI Dialogue Engine GDD: after each NPC response, a brief summary is
// appended; previously disclosed fact IDs are tracked so the NPC can
// acknowledge known information.
//
// The memory supports snapshot/restore for save/load round-trip (AC-27).
type ConversationMemory struct {
	// RollingSummary is a bullet-point summary of the conversation so far,
	// formatted for prompt injection. Max ~150 tokens; older bullets are
	// dropped when the limit is exceeded.
	RollingSummary string

	// DisclosedFacts is the set of fact IDs that have been included in
	// prompts during this conversation. Prevents NPCs from re-introducing
	// facts the player already knows.
	DisclosedFacts []string

	// ExchangeCount is the number of question-response pairs.
	ExchangeCount int
}

// RecordExchange appends a new exchange to the conversation memory.
// After recording, the rolling summary is trimmed to MaxSummaryTokens.
func (m *ConversationMemory) RecordExchange(question, response string) {
	m.ExchangeCount++

	// Build a brief summary bullet: first ~60 chars of the question,
	// followed by the key point from the response.
	qSummary := truncateToChars(question, 60)
	rSummary := truncateToChars(response, 80)
	bullet := "- Q: " + qSummary + " → " + rSummary

	if m.RollingSummary == "" {
		m.RollingSummary = bullet
	} else {
		m.RollingSummary = m.RollingSummary + "\n" + bullet
	}

	// Trim to token budget. Approximate: 1 token ≈ 4 chars.
	m.trimSummary()
}

// RecordDisclosed adds a fact ID to the disclosed set. Duplicates are ignored.
func (m *ConversationMemory) RecordDisclosed(factID string) {
	if slices.Contains(m.DisclosedFacts, factID) {
		return
	}
	m.DisclosedFacts = append(m.DisclosedFacts, factID)
}

// HasDisclosed reports whether a fact ID has already been disclosed.
func (m *ConversationMemory) HasDisclosed(factID string) bool {
	return slices.Contains(m.DisclosedFacts, factID)
}

// SummaryForPrompt returns the rolling summary formatted for inclusion in
// the LLM prompt. Returns an empty string if no exchanges have occurred.
func (m *ConversationMemory) SummaryForPrompt() string {
	if m.RollingSummary == "" {
		return ""
	}
	return "Previous conversation:\n" + m.RollingSummary
}

// DisclosedSummary returns a compact list of disclosed fact labels for
// prompt injection. Returns empty if none disclosed.
func (m *ConversationMemory) DisclosedSummary() string {
	if len(m.DisclosedFacts) == 0 {
		return ""
	}
	return "Already discussed: " + strings.Join(m.DisclosedFacts, ", ")
}

// Snapshot returns a deep copy for save serialization (AC-27).
func (m *ConversationMemory) Snapshot() ConversationMemory {
	facts := make([]string, len(m.DisclosedFacts))
	copy(facts, m.DisclosedFacts)
	return ConversationMemory{
		RollingSummary: m.RollingSummary,
		DisclosedFacts: facts,
		ExchangeCount:  m.ExchangeCount,
	}
}

// Restore replaces the current state with a saved snapshot (AC-27).
func (m *ConversationMemory) Restore(snapshot ConversationMemory) {
	facts := make([]string, len(snapshot.DisclosedFacts))
	copy(facts, snapshot.DisclosedFacts)
	m.RollingSummary = snapshot.RollingSummary
	m.DisclosedFacts = facts
	m.ExchangeCount = snapshot.ExchangeCount
}

// GapAcknowledgement returns text the NPC should prepend when resuming
// a restored conversation (AC-27). Empty string if ExchangeCount is 0.
func (m *ConversationMemory) GapAcknowledgement() string {
	if m.ExchangeCount == 0 {
		return ""
	}
	return "It's been a while since we last spoke..."
}

// trimSummary drops the oldest bullet points until the summary fits within
// MaxSummaryTokens (approximated as chars/4).
func (m *ConversationMemory) trimSummary() {
	maxChars := MaxSummaryTokens * 4
	for len(m.RollingSummary) > maxChars {
		idx := strings.IndexByte(m.RollingSummary, '\n')
		if idx < 0 {
			// Single bullet but still too long: truncate hard.
			if len(m.RollingSummary) > maxChars {
				m.RollingSummary = m.RollingSummary[:maxChars-3] + "..."
			}
			return
		}
		m.RollingSummary = m.RollingSummary[idx+1:]
	}
}

// truncateToChars truncates s to at most n characters, adding "…" if truncated.
func truncateToChars(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n-1] + "…"
}
