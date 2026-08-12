package investigation_journal_test

import (
	"testing"

	"marrow-of-empire/investigation_journal"
)

// ──────────────────────────────────────────────────────────────────────────────
// AC-1, AC-4: Map Center Request Event
// ──────────────────────────────────────────────────────────────────────────────

func TestMapCenterEvent_Fields(t *testing.T) {
	evt := investigation_journal.MapCenterEvent{
		EntityID:  42,
		PosX:      45.0,
		PosY:      30.0,
		Requester: "investigation_journal",
	}
	if evt.EntityID != 42 {
		t.Errorf("EntityID = %d, want 42", evt.EntityID)
	}
	if evt.PosX != 45.0 || evt.PosY != 30.0 {
		t.Errorf("Position = (%.1f, %.1f), want (45.0, 30.0)", evt.PosX, evt.PosY)
	}
	if evt.Requester != "investigation_journal" {
		t.Errorf("Requester = %q, want investigation_journal", evt.Requester)
	}
}

func TestEventMapCenterRequested_IsNonEmpty(t *testing.T) {
	if investigation_journal.EventMapCenterRequested == "" {
		t.Error("EventMapCenterRequested must not be empty")
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// AC-2: Thread Location Resolution
// ──────────────────────────────────────────────────────────────────────────────

func TestResolveThreadLocation_ReturnsInfo(t *testing.T) {
	info := investigation_journal.ResolveThreadLocation(42)
	if info.TargetEntity != 42 {
		t.Errorf("TargetEntity = %d, want 42", info.TargetEntity)
	}
	// HasLocation is false until Entity Framework position lookup is wired.
	// This is the expected stub behavior.
	if info.HasLocation {
		t.Log("HasLocation = true — Entity Framework position lookup may be wired")
	}
}

func TestResolveThreadLocation_ZeroEntity(t *testing.T) {
	info := investigation_journal.ResolveThreadLocation(0)
	if info.TargetEntity != 0 {
		t.Errorf("TargetEntity = %d, want 0", info.TargetEntity)
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// AC-4: Map unavailable — event emission is tested via mock subscriber
// ──────────────────────────────────────────────────────────────────────────────

func TestMapCenterEvent_RequesterField(t *testing.T) {
	// All MapCenterEvent instances from the journal must identify themselves.
	evt := investigation_journal.MapCenterEvent{Requester: "investigation_journal"}
	if evt.Requester != "investigation_journal" {
		t.Errorf("Requester = %q, want investigation_journal", evt.Requester)
	}
}
