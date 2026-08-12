package investigation_journal

import (
	"sort"

	"marrow-of-empire/thread_investigation"
)

// ChainEntry is a display-ready chain row for the sidebar.
// Extracted from thread_investigation.Chain + computed sort key.
type ChainEntry struct {
	ID           string
	Summary      string
	ThreadCount  int
	ActiveCount  int
	FactCount    int
	Status       string // "active", "revelation_ready", "resolved", "archived"
	LastUpdated  float64 // most recent thread CreatedAt in this chain
}

// BuildChainList extracts chain entries from PlayerKnowledge and sorts them:
// active first (by last_updated desc), then resolved/archived (by last_updated desc).
// Archived chains are excluded from display.
func BuildChainList(pk *thread_investigation.PlayerKnowledge) []ChainEntry {
	if pk == nil {
		return nil
	}

	// Index threads by ID for O(1) lookup of CreatedAt.
	threadByID := make(map[string]*thread_investigation.Thread, len(pk.ActiveThreads)+len(pk.ResolvedThreads))
	for i := range pk.ActiveThreads {
		threadByID[pk.ActiveThreads[i].ID] = &pk.ActiveThreads[i]
	}
	for i := range pk.ResolvedThreads {
		threadByID[pk.ResolvedThreads[i].ID] = &pk.ResolvedThreads[i]
	}

	entries := make([]ChainEntry, 0, len(pk.Chains))
	for _, ch := range pk.Chains {
		if ch.Status == thread_investigation.ChainStatusArchived {
			continue // archived chains are not displayed
		}

		entry := ChainEntry{
			ID:      ch.ID,
			Summary: ch.Summary,
			Status:  ch.Status,
		}

		// Count facts and threads; compute most recent update.
		entry.FactCount = len(ch.Facts)
		entry.ThreadCount = len(ch.Threads)
		for _, tid := range ch.Threads {
			if t, ok := threadByID[tid]; ok {
				if t.Status == thread_investigation.ThreadActive || t.Status == thread_investigation.ThreadPrioritized {
					entry.ActiveCount++
				}
				if t.CreatedAt > entry.LastUpdated {
					entry.LastUpdated = t.CreatedAt
				}
			}
		}

		entries = append(entries, entry)
	}

	// Sort: active/revelation_ready first, then resolved, then dead_end chains.
	// Within each group, most recently updated first.
	sort.Slice(entries, func(i, j int) bool {
		ai := chainStatusGroup(entries[i].Status)
		aj := chainStatusGroup(entries[j].Status)
		if ai != aj {
			return ai < aj
		}
		return entries[i].LastUpdated > entries[j].LastUpdated
	})

	return entries
}

// chainStatusGroup returns a sort group: 0 = active/revelation_ready (top),
// 1 = resolved, 2 = everything else.
func chainStatusGroup(status string) int {
	switch status {
	case "active", thread_investigation.ChainStatusRevelationReady:
		return 0
	case "resolved":
		return 1
	default:
		return 2
	}
}
