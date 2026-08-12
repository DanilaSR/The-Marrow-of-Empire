package investigation_journal_test

import (
	"testing"

	"marrow-of-empire/investigation_journal"
	"marrow-of-empire/thread_investigation"
)

// ──────────────────────────────────────────────────────────────────────────────
// AC-1, AC-5: Chain List Building & Sorting
// ──────────────────────────────────────────────────────────────────────────────

func TestBuildChainList_EmptyPlayerKnowledge(t *testing.T) {
	entries := investigation_journal.BuildChainList(nil)
	if entries != nil {
		t.Errorf("BuildChainList(nil) = %v, want nil", entries)
	}

	entries = investigation_journal.BuildChainList(&thread_investigation.PlayerKnowledge{})
	if len(entries) != 0 {
		t.Errorf("BuildChainList(empty) has %d entries, want 0", len(entries))
	}
}

func TestBuildChainList_ExcludesArchivedChains(t *testing.T) {
	pk := &thread_investigation.PlayerKnowledge{
		Chains: []thread_investigation.Chain{
			{ID: "chain_1", Summary: "Active Chain", Status: "active"},
			{ID: "chain_2", Summary: "Archived Chain", Status: thread_investigation.ChainStatusArchived},
		},
	}
	entries := investigation_journal.BuildChainList(pk)
	if len(entries) != 1 {
		t.Fatalf("BuildChainList has %d entries, want 1 (archived excluded)", len(entries))
	}
	if entries[0].ID != "chain_1" {
		t.Errorf("remaining entry ID = %q, want chain_1", entries[0].ID)
	}
}

func TestBuildChainList_SortsActiveBeforeResolved(t *testing.T) {
	pk := &thread_investigation.PlayerKnowledge{
		ActiveThreads: []thread_investigation.Thread{
			{ID: "t1", ChainID: "chain_1", CreatedAt: 100, Status: thread_investigation.ThreadActive},
			{ID: "t2", ChainID: "chain_2", CreatedAt: 200, Status: thread_investigation.ThreadActive},
		},
		Chains: []thread_investigation.Chain{
			{ID: "chain_1", Summary: "Resolved Chain", Status: "resolved", Threads: []string{"t1"}},
			{ID: "chain_2", Summary: "Active Chain", Status: "active", Threads: []string{"t2"}},
		},
	}
	entries := investigation_journal.BuildChainList(pk)
	if len(entries) != 2 {
		t.Fatalf("BuildChainList has %d entries, want 2", len(entries))
	}
	// Active chain must come first regardless of timestamps.
	if entries[0].Status != "active" {
		t.Errorf("first entry status = %q, want active", entries[0].Status)
	}
	if entries[1].Status != "resolved" {
		t.Errorf("second entry status = %q, want resolved", entries[1].Status)
	}
}

func TestBuildChainList_WithinGroupSortsByLastUpdated(t *testing.T) {
	pk := &thread_investigation.PlayerKnowledge{
		ActiveThreads: []thread_investigation.Thread{
			{ID: "t1", ChainID: "chain_a", CreatedAt: 100, Status: thread_investigation.ThreadActive},
			{ID: "t2", ChainID: "chain_b", CreatedAt: 300, Status: thread_investigation.ThreadActive},
		},
		Chains: []thread_investigation.Chain{
			{ID: "chain_a", Summary: "Older", Status: "active", Threads: []string{"t1"}},
			{ID: "chain_b", Summary: "Newer", Status: "active", Threads: []string{"t2"}},
		},
	}
	entries := investigation_journal.BuildChainList(pk)
	if len(entries) != 2 {
		t.Fatalf("BuildChainList has %d entries, want 2", len(entries))
	}
	// Both active — newer (300) before older (100).
	if entries[0].ID != "chain_b" {
		t.Errorf("first entry = %q (CreatedAt ~300), want chain_b (newer)", entries[0].ID)
	}
	if entries[1].ID != "chain_a" {
		t.Errorf("second entry = %q (CreatedAt ~100), want chain_a (older)", entries[1].ID)
	}
}

func TestBuildChainList_CountsThreadsAndFacts(t *testing.T) {
	pk := &thread_investigation.PlayerKnowledge{
		ActiveThreads: []thread_investigation.Thread{
			{ID: "t1", ChainID: "chain_1", Status: thread_investigation.ThreadActive},
			{ID: "t2", ChainID: "chain_1", Status: thread_investigation.ThreadPrioritized},
		},
		Chains: []thread_investigation.Chain{
			{ID: "chain_1", Summary: "Test", Status: "active", Threads: []string{"t1", "t2"}, Facts: []string{"f1", "f2", "f3", "f4", "f5", "f6", "f7"}},
		},
	}
	entries := investigation_journal.BuildChainList(pk)
	if len(entries) != 1 {
		t.Fatalf("BuildChainList has %d entries, want 1", len(entries))
	}
	if entries[0].ThreadCount != 2 {
		t.Errorf("ThreadCount = %d, want 2", entries[0].ThreadCount)
	}
	if entries[0].ActiveCount != 2 {
		t.Errorf("ActiveCount = %d, want 2", entries[0].ActiveCount)
	}
	if entries[0].FactCount != 7 {
		t.Errorf("FactCount = %d, want 7", entries[0].FactCount)
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// AC-4: Thread List Building & Sorting
// ──────────────────────────────────────────────────────────────────────────────

func TestBuildThreadList_EmptyPlayerKnowledge(t *testing.T) {
	entries := investigation_journal.BuildThreadList(nil, "chain_1")
	if entries != nil {
		t.Errorf("BuildThreadList(nil) = %v, want nil", entries)
	}
}

func TestBuildThreadList_FiltersByChainID(t *testing.T) {
	pk := &thread_investigation.PlayerKnowledge{
		ActiveThreads: []thread_investigation.Thread{
			{ID: "t1", ChainID: "chain_a", Topic: "In chain A", Status: thread_investigation.ThreadActive},
			{ID: "t2", ChainID: "chain_b", Topic: "In chain B", Status: thread_investigation.ThreadActive},
		},
	}
	entries := investigation_journal.BuildThreadList(pk, "chain_a")
	if len(entries) != 1 {
		t.Fatalf("BuildThreadList for chain_a has %d entries, want 1", len(entries))
	}
	if entries[0].Topic != "In chain A" {
		t.Errorf("topic = %q, want 'In chain A'", entries[0].Topic)
	}
}

func TestBuildThreadList_PrioritizedSortsFirst(t *testing.T) {
	pk := &thread_investigation.PlayerKnowledge{
		ActiveThreads: []thread_investigation.Thread{
			{ID: "t1", ChainID: "chain_1", Topic: "Normal", Status: thread_investigation.ThreadActive, CreatedAt: 500},
			{ID: "t2", ChainID: "chain_1", Topic: "Star", Status: thread_investigation.ThreadPrioritized, CreatedAt: 100},
		},
	}
	entries := investigation_journal.BuildThreadList(pk, "chain_1")
	if len(entries) != 2 {
		t.Fatalf("BuildThreadList has %d entries, want 2", len(entries))
	}
	// Prioritized first, regardless of older CreatedAt.
	if entries[0].Topic != "Star" {
		t.Errorf("first entry topic = %q, want 'Star' (prioritized)", entries[0].Topic)
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// Status Icons & Labels
// ──────────────────────────────────────────────────────────────────────────────

func TestThreadStatusIcon_AllStatusesHaveIcon(t *testing.T) {
	statuses := []thread_investigation.ThreadStatus{
		thread_investigation.ThreadActive,
		thread_investigation.ThreadPrioritized,
		thread_investigation.ThreadResolved,
		thread_investigation.ThreadDeadEnd,
	}
	for _, s := range statuses {
		icon := investigation_journal.ThreadStatusIcon(s)
		if icon == "" {
			t.Errorf("ThreadStatusIcon(%q) is empty", s)
		}
	}
}

func TestThreadStatusIcon_DistinctByStatus(t *testing.T) {
	active := investigation_journal.ThreadStatusIcon(thread_investigation.ThreadActive)
	prioritized := investigation_journal.ThreadStatusIcon(thread_investigation.ThreadPrioritized)
	resolved := investigation_journal.ThreadStatusIcon(thread_investigation.ThreadResolved)
	deadEnd := investigation_journal.ThreadStatusIcon(thread_investigation.ThreadDeadEnd)

	// Active and resolved must differ (shape carries meaning, not just color).
	if active == resolved {
		t.Error("active and resolved icons must be distinct shapes")
	}
	if prioritized == deadEnd {
		t.Error("prioritized and dead-end icons must be distinct shapes")
	}
}

func TestThreadStatusLabel_AllStatusesHaveLabel(t *testing.T) {
	statuses := []thread_investigation.ThreadStatus{
		thread_investigation.ThreadActive,
		thread_investigation.ThreadPrioritized,
		thread_investigation.ThreadResolved,
		thread_investigation.ThreadDeadEnd,
	}
	for _, s := range statuses {
		label := investigation_journal.ThreadStatusLabel(s)
		if label == "" {
			t.Errorf("ThreadStatusLabel(%q) is empty", s)
		}
	}
}

func TestChainStatusLabel_AllStatusesCovered(t *testing.T) {
	labels := map[string]string{
		"active":                                "Active",
		thread_investigation.ChainStatusRevelationReady: "Revelation Available",
		"resolved":                              "Resolved",
		thread_investigation.ChainStatusArchived:        "Archived",
	}
	for status, expected := range labels {
		got := investigation_journal.ChainStatusLabel(status)
		if got != expected {
			t.Errorf("ChainStatusLabel(%q) = %q, want %q", status, got, expected)
		}
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// Time Formatting
// ──────────────────────────────────────────────────────────────────────────────

func TestFormatThreadAge_JustNow(t *testing.T) {
	if got := investigation_journal.FormatThreadAge(0); got != "Just now" {
		t.Errorf("FormatThreadAge(0) = %q, want 'Just now'", got)
	}
	if got := investigation_journal.FormatThreadAge(30); got != "Just now" {
		t.Errorf("FormatThreadAge(30 min) = %q, want 'Just now'", got)
	}
}

func TestFormatThreadAge_Hours(t *testing.T) {
	if got := investigation_journal.FormatThreadAge(60); got != "1 hour ago" {
		t.Errorf("FormatThreadAge(60 min) = %q, want '1 hour ago'", got)
	}
	if got := investigation_journal.FormatThreadAge(180); got != "3 hours ago" {
		t.Errorf("FormatThreadAge(180 min) = %q, want '3 hours ago'", got)
	}
	if got := investigation_journal.FormatThreadAge(23 * 60); got != "23 hours ago" {
		t.Errorf("FormatThreadAge(23h) = %q, want '23 hours ago'", got)
	}
}

func TestFormatThreadAge_Days(t *testing.T) {
	if got := investigation_journal.FormatThreadAge(24 * 60); got != "1 day ago" {
		t.Errorf("FormatThreadAge(24h) = %q, want '1 day ago'", got)
	}
	if got := investigation_journal.FormatThreadAge(72 * 60); got != "3 days ago" {
		t.Errorf("FormatThreadAge(72h) = %q, want '3 days ago'", got)
	}
}

func TestFormatThreadAge_NegativeClamped(t *testing.T) {
	// Negative game time (clock anomaly) should clamp to "Just now", not panic.
	if got := investigation_journal.FormatThreadAge(-100); got != "Just now" {
		t.Errorf("FormatThreadAge(-100) = %q, want 'Just now'", got)
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// Count Formatting
// ──────────────────────────────────────────────────────────────────────────────

func TestFormatThreadCount_Singular(t *testing.T) {
	if got := investigation_journal.FormatThreadCount(1); got != "1 active lead" {
		t.Errorf("FormatThreadCount(1) = %q, want '1 active lead'", got)
	}
}

func TestFormatThreadCount_Plural(t *testing.T) {
	if got := investigation_journal.FormatThreadCount(3); got != "3 active leads" {
		t.Errorf("FormatThreadCount(3) = %q, want '3 active leads'", got)
	}
}

func TestFormatThreadCount_Zero(t *testing.T) {
	if got := investigation_journal.FormatThreadCount(0); got != "0 active leads" {
		t.Errorf("FormatThreadCount(0) = %q, want '0 active leads'", got)
	}
}

func TestFormatFactCount_Singular(t *testing.T) {
	if got := investigation_journal.FormatFactCount(1); got != "1 fact discovered" {
		t.Errorf("FormatFactCount(1) = %q, want '1 fact discovered'", got)
	}
}

func TestFormatFactCount_Plural(t *testing.T) {
	if got := investigation_journal.FormatFactCount(7); got != "7 facts discovered" {
		t.Errorf("FormatFactCount(7) = %q, want '7 facts discovered'", got)
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// State: ChainSelected
// ──────────────────────────────────────────────────────────────────────────────

func TestJournalState_ChainSelectedIsOpen(t *testing.T) {
	// StateChainSelected is a sub-state of Open — IsOpen must return true.
	if !investigation_journal.StateChainSelected.IsOpen() {
		t.Error("StateChainSelected.IsOpen() = false, want true")
	}
}

func TestJournalState_ChainSelectedString(t *testing.T) {
	if got := investigation_journal.StateChainSelected.String(); got != "ChainSelected" {
		t.Errorf("StateChainSelected.String() = %q, want 'ChainSelected'", got)
	}
}

func TestJournalState_AllStatesAreDistinct(t *testing.T) {
	states := []investigation_journal.JournalState{
		investigation_journal.StateClosed,
		investigation_journal.StateOpen,
		investigation_journal.StateChainSelected,
	}
	seen := make(map[investigation_journal.JournalState]bool)
	for _, s := range states {
		if seen[s] {
			t.Errorf("duplicate state value: %v (%d)", s, int(s))
		}
		seen[s] = true
	}
	if len(seen) != 3 {
		t.Errorf("expected 3 distinct states, got %d", len(seen))
	}
}
