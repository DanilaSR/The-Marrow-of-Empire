package investigation_journal

import (
	"fmt"
	"sort"

	"marrow-of-empire/thread_investigation"
)

// ThreadEntry is a display-ready thread row for the detail view.
// Extracted from thread_investigation.Thread with display formatting applied.
type ThreadEntry struct {
	ID           string
	Topic        string
	SourceNPC    uint64 // GameEntityID of the source NPC
	Status       thread_investigation.ThreadStatus
	StatusIcon   string // display icon character
	StatusLabel  string // human-readable status
	ChainID      string
	CreatedAt    float64
	AgeDisplay   string // formatted age string (computed by caller)
	HasLocation  bool   // true if target entity has a known position
}

// BuildThreadList extracts thread entries for a given chain from PlayerKnowledge
// and sorts them: prioritized first, then by CreatedAt descending.
func BuildThreadList(pk *thread_investigation.PlayerKnowledge, chainID string) []ThreadEntry {
	if pk == nil {
		return nil
	}

	// Collect threads belonging to this chain.
	entries := make([]ThreadEntry, 0)
	for _, t := range pk.ActiveThreads {
		if t.ChainID == chainID {
			entries = append(entries, threadToEntry(t))
		}
	}
	for _, t := range pk.ResolvedThreads {
		if t.ChainID == chainID {
			entries = append(entries, threadToEntry(t))
		}
	}

	// Sort: prioritized first, then by CreatedAt descending.
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Status == thread_investigation.ThreadPrioritized && entries[j].Status != thread_investigation.ThreadPrioritized {
			return true
		}
		if entries[j].Status == thread_investigation.ThreadPrioritized && entries[i].Status != thread_investigation.ThreadPrioritized {
			return false
		}
		return entries[i].CreatedAt > entries[j].CreatedAt
	})

	return entries
}

// threadToEntry converts a Thread to a display-ready ThreadEntry.
func threadToEntry(t thread_investigation.Thread) ThreadEntry {
	return ThreadEntry{
		ID:          t.ID,
		Topic:       t.Topic,
		SourceNPC:   0, // resolved from fact's SourceNPC via PlayerKnowledge by caller
		Status:      t.Status,
		StatusIcon:  ThreadStatusIcon(t.Status),
		StatusLabel: ThreadStatusLabel(t.Status),
		ChainID:     t.ChainID,
		CreatedAt:   t.CreatedAt,
	}
}

// ThreadStatusIcon returns the pen-and-ink icon character for a thread status.
// These are Unicode characters that serve as placeholders until proper
// pen-and-ink icon assets are created (per art bible Section 6.3).
func ThreadStatusIcon(status thread_investigation.ThreadStatus) string {
	switch status {
	case thread_investigation.ThreadPrioritized:
		return "★" // amber star
	case thread_investigation.ThreadResolved:
		return "✓" // green check
	case thread_investigation.ThreadDeadEnd:
		return "✗" // grey X
	default:
		return "●" // amber dot (active)
	}
}

// ThreadStatusLabel returns the human-readable status label for a thread.
func ThreadStatusLabel(status thread_investigation.ThreadStatus) string {
	switch status {
	case thread_investigation.ThreadPrioritized:
		return "Prioritized"
	case thread_investigation.ThreadResolved:
		return "Resolved"
	case thread_investigation.ThreadDeadEnd:
		return "Dead End"
	default:
		return "Active"
	}
}

// ThreadStatusColor returns the palette color for a thread status icon.
// Amber (#D9A441) for active/prioritized, green (#5A8A4A) for resolved,
// grey (#8A8A8A) for dead end. Never use amber for text — icons only.
func ThreadStatusColor(status thread_investigation.ThreadStatus) string {
	switch status {
	case thread_investigation.ThreadResolved:
		return "#5A8A4A"
	case thread_investigation.ThreadDeadEnd:
		return "#8A8A8A"
	default:
		return "#D9A441" // amber
	}
}

// ChainStatusLabel returns a human-readable label for a chain status.
func ChainStatusLabel(status string) string {
	switch status {
	case thread_investigation.ChainStatusRevelationReady:
		return "Revelation Available"
	case "resolved":
		return "Resolved"
	case thread_investigation.ChainStatusArchived:
		return "Archived"
	default:
		return "Active"
	}
}

// FormatThreadAge formats a game-time timestamp difference as a human-readable
// age string: "Just now" (<1 hour), "N hours ago" (<24 hours), "N days ago".
// gameMinutesAgo is the difference between current game time and the thread's CreatedAt.
func FormatThreadAge(gameMinutesAgo float64) string {
	if gameMinutesAgo < 0 {
		gameMinutesAgo = 0
	}
	hours := gameMinutesAgo / 60.0
	if hours < 1.0 {
		return "Just now"
	}
	if hours < 24.0 {
		h := int(hours)
		if h == 1 {
			return "1 hour ago"
		}
		return fmt.Sprintf("%d hours ago", h)
	}
	days := int(hours / 24.0)
	if days == 1 {
		return "1 day ago"
	}
	return fmt.Sprintf("%d days ago", days)
}

// FormatThreadCount returns a display string for thread counts.
// "1 active lead", "3 active leads", "0 active leads".
func FormatThreadCount(active int) string {
	if active == 1 {
		return "1 active lead"
	}
	return fmt.Sprintf("%d active leads", active)
}

// FormatFactCount returns a display string for fact counts.
// "1 fact discovered", "7 facts discovered", "0 facts discovered".
func FormatFactCount(count int) string {
	if count == 1 {
		return "1 fact discovered"
	}
	return fmt.Sprintf("%d facts discovered", count)
}
