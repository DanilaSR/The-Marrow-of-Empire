package investigation_journal

// EntityChip is a display-ready entity reference chip for the quick-ref bar
// or inline fact references.
type EntityChip struct {
	EntityID   uint64
	Name       string // resolved from Entity Framework by caller
	LastSeenAt float64 // game timestamp of most recent reference
}

// EntityRefSet tracks recently referenced entities for the quick-ref bar.
// Max 8 entries, most recent first. Thread-safe for single-goroutine UI use.
type EntityRefSet struct {
	chips []EntityChip
}

// NewEntityRefSet returns an empty entity reference set.
func NewEntityRefSet() *EntityRefSet {
	return &EntityRefSet{
		chips: make([]EntityChip, 0, 8),
	}
}

// RecordEntity records a reference to an entity, moving it to the front.
// If the entity is already tracked, its LastSeenAt is updated and it moves to
// position 0. Max 8 entries; oldest falls off when exceeded.
func (s *EntityRefSet) RecordEntity(entityID uint64, name string, seenAt float64) {
	// Remove existing entry if present.
	for i, c := range s.chips {
		if c.EntityID == entityID {
			s.chips = append(s.chips[:i], s.chips[i+1:]...)
			break
		}
	}
	// Prepend new entry.
	s.chips = append([]EntityChip{{EntityID: entityID, Name: name, LastSeenAt: seenAt}}, s.chips...)
	// Trim to max 8.
	if len(s.chips) > 8 {
		s.chips = s.chips[:8]
	}
}

// Chips returns a copy of the current chip list (most recent first).
func (s *EntityRefSet) Chips() []EntityChip {
	out := make([]EntityChip, len(s.chips))
	copy(out, s.chips)
	return out
}

// Len returns the number of tracked entities.
func (s *EntityRefSet) Len() int {
	return len(s.chips)
}
