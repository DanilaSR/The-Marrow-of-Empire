package investigation_journal

import "marrow-of-empire/entity"

// Map integration event type (Story 005).
// Emitted when the player clicks a thread referencing a known location.
// The Map System (or a mock subscriber in tests) handles centering.
const EventMapCenterRequested entity.EventType = "map.center_requested"

// MapCenterEvent is the payload for EventMapCenterRequested.
type MapCenterEvent struct {
	EntityID   uint64  // target entity (location)
	PosX       float32 // known X position (0 if unknown)
	PosY       float32 // known Y position (0 if unknown)
	Requester  string  // "investigation_journal"
}

// ThreadLocationInfo bundles a thread's location data for map integration.
type ThreadLocationInfo struct {
	ThreadID    string
	TargetEntity uint64
	HasLocation bool
	PosX        float32
	PosY        float32
}

// ResolveThreadLocation checks whether a thread's target entity has a known
// position. Returns location info with HasLocation=true if position data exists.
// This is a stub — real position lookup needs Entity Framework Transform component.
func ResolveThreadLocation(targetEntity uint64) ThreadLocationInfo {
	// VERIFY: Entity Framework — Transform component lookup for position data.
	// For now, all entities report HasLocation=false (fallback to entity journal).
	return ThreadLocationInfo{
		TargetEntity: targetEntity,
		HasLocation:  false,
	}
}
