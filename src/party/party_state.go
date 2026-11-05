// Package party implements the Party System's roster store (Story 001:
// Party Roster & Active Slots; TR-party-001; ADR-0006, ADR-0011).
//
// The Party System owns a single roster per save — one PartyState, not a
// per-entity component: the player character (PCID, fixed at character
// creation and never removable), the active formation (Active, PC first then
// companions in marching order), the recruitment pool (Pool — recruited
// companions waiting at the current rest point), and the current rest point
// (RestPoint, where the pool is recorded as present).
//
// Architecture (ADR-0006/ADR-0001): the store follows the typed
// component-store pattern — pure Go data with no Kaiju engine APIs — and
// implements entity.EntityData (Encode/Decode via pod) with PartyState
// registered via pod.Register, so Save/Load can serialize the roster
// (ADR-0001, ADR-0002). The pc_controlled tag lives on a party-owned
// component (PCControlled, component.go) registered with the Entity
// Framework via entity.RegisterComponent, so the tag persists through the
// Entity Framework's own save/load pass (GDD CR-8); the store writes it
// through the injected PCTagAccessor, whose production adapter (EFTagAdapter)
// goes through the Entity Framework's write arbitration (ADR-0001).
//
// All state mutations go through this store's interface; roster changes are
// notified on the EventBus (ADR-0011, synchronous dispatch):
// party.member_added, party.member_removed, party.pool_changed (events.go).
// Consuming systems set flags and never write the roster.
//
// GameEntityIDs are never created or reused here: the Entity Framework owns
// the monotonic ID counter (ADR-0001: "never reuse a GameEntityID" — the
// party store only references IDs given to it).
//
// Story: production/epics/party-system/story-001-party-roster-active-slots.md
// GDD:  design/gdd/party-system.md (CR-1 Party Size and Composition)
package party

import (
	"bytes"
	"fmt"
	"log/slog"

	"kaijuengine.com/engine/encoding/pod"

	"marrow-of-empire/entity"
)

// Default roster tuning knobs from the party GDD CR-1 tuning knob table.
// Story 001 injects them via PartyStoreConfig; gameplay values stay
// data-driven — the numbers are never hardcoded in the store.
const (
	// DefaultVSPartySize is the GDD `vs_party_size` knob (4): the active
	// formation capacity at Vertical Slice — the player character plus three
	// companions.
	DefaultVSPartySize = 4
	// DefaultFullVisionPartySize is the GDD `full_vision_party_size` knob
	// (6): the active formation capacity at Full Vision — the player
	// character plus five companions.
	DefaultFullVisionPartySize = 6
)

// PartyState is the serializable roster state (story implementation note:
// "one PartyState per save, not per entity"). It is stored on the
// PartyStore and pod-encoded as a single value.
//
// Invariants maintained by the store's mutation API: Active[0] is always
// the player character whenever PCID is set; Active and Pool are disjoint
// and never contain duplicates. Decode (the load path) replaces the state
// wholesale and trusts the decoded value to satisfy the invariants.
type PartyState struct {
	// PCID is the player character's GameEntityID, fixed at character
	// creation and never removable (AC-1: the PC slot is locked).
	PCID entity.GameEntityID
	// Active lists the active party members in marching order: the player
	// character first, then companions.
	Active []entity.GameEntityID
	// Pool lists the recruited companions waiting at the current rest point
	// (AC-5: pool companions automatically relocate with the party).
	Pool []entity.GameEntityID
	// RestPoint is the rest point the pool is recorded as present at; zero
	// means the party has not rested yet.
	RestPoint entity.GameEntityID
}

// PartyStoreConfig carries the party roster tuning knobs (GDD CR-1 tuning
// knob table). Zero-valued fields fall back to the GDD defaults
// (vs_party_size 4, full_vision_party_size 6) — mirroring the Entity
// Framework's EntityStoreConfig pattern; a later story wires these to
// external config files.
type PartyStoreConfig struct {
	// VSPartySize is the GDD `vs_party_size` knob (default 4): the active
	// formation capacity at Vertical Slice. The store starts with it as its
	// active capacity.
	VSPartySize int
	// FullVisionPartySize is the GDD `full_vision_party_size` knob
	// (default 6): the active formation capacity at Full Vision. The Full
	// Vision milestone switches the store to it via SetMaxActiveSize.
	FullVisionPartySize int
}

// PartyStore is the Party System's roster store: one per save (story
// implementation note), not per entity. It owns the PartyState, the active
// formation capacity read from config, and the pc_controlled tag writes on
// companion entities (PCTagAccessor — the Party System is the tag's
// authoritative writer, ADR-0001 write-ownership).
//
// All state mutations go through this store's methods; roster changes emit
// events on the injected EventBus (ADR-0011) so downstream systems (TCE,
// UI) update in the same frame — they never write the roster.
//
// The store is not safe for concurrent use: the game loop is
// single-threaded and systems drive it from the main thread.
type PartyStore struct {
	state    PartyState
	vsSize   int
	fullSize int
	capacity int
	tags     PCTagAccessor
	bus      entity.EventBus
}

// NewPartyStore creates the roster store (dependency injection — no
// singletons). config zero values fall back to the GDD defaults
// (vs_party_size 4, full_vision_party_size 6); the store starts with
// vs_party_size as its active capacity. tags is the pc_controlled write
// surface (nil skips tag writes — headless operation); bus receives the
// roster notification events (nil skips emission, mirroring the character
// sheet's nil-bus pattern).
//
// Example usage:
//
//	store := party.NewPartyStore(party.PartyStoreConfig{}, tagAdapter, bus)
//	if err := store.SetPC(pcID); err != nil { /* handle */ }
func NewPartyStore(config PartyStoreConfig, tags PCTagAccessor, bus entity.EventBus) *PartyStore {
	if config.VSPartySize <= 0 {
		config.VSPartySize = DefaultVSPartySize
	}
	if config.FullVisionPartySize <= 0 {
		config.FullVisionPartySize = DefaultFullVisionPartySize
	}
	return &PartyStore{
		vsSize:   config.VSPartySize,
		fullSize: config.FullVisionPartySize,
		capacity: config.VSPartySize,
		tags:     tags,
		bus:      bus,
	}
}

// PC returns the player character's GameEntityID; zero until SetPC.
func (s *PartyStore) PC() entity.GameEntityID { return s.state.PCID }

// ActiveMembers returns a copy of the active formation in marching order
// (player character first, AC-3: the party screen's active slots render
// from this). The copy is defensive — callers may never mutate the store's
// internal slices.
func (s *PartyStore) ActiveMembers() []entity.GameEntityID {
	return append([]entity.GameEntityID(nil), s.state.Active...)
}

// PoolMembers returns a copy of the recruitment pool (AC-5: pool
// companions are present at RestPoint).
func (s *PartyStore) PoolMembers() []entity.GameEntityID {
	return append([]entity.GameEntityID(nil), s.state.Pool...)
}

// RestPoint returns the rest point the pool is recorded as present at
// (AC-5); zero when the party has not rested yet.
func (s *PartyStore) RestPoint() entity.GameEntityID { return s.state.RestPoint }

// Capacity returns the current active formation capacity — the player
// character counts against it (GDD vs_party_size 4 = PC plus three
// companions). Starts at vs_party_size; SetMaxActiveSize switches it (Full
// Vision milestone).
func (s *PartyStore) Capacity() int { return s.capacity }

// IsActive reports whether id is in the active formation.
func (s *PartyStore) IsActive(id entity.GameEntityID) bool {
	for _, m := range s.state.Active {
		if m == id {
			return true
		}
	}
	return false
}

// InPool reports whether id is in the recruitment pool.
func (s *PartyStore) InPool(id entity.GameEntityID) bool {
	for _, m := range s.state.Pool {
		if m == id {
			return true
		}
	}
	return false
}

// Encode implements entity.EntityData (ADR-0001, ADR-0002): pod-encodes the
// store's full PartyState value. Save/Load serializes the roster through
// this method — the component-store pattern's Encode/Decode contract, with
// the roster stored as one value per save rather than a
// map[GameEntityID]T store. The blob is a valid input to Decode.
func (s *PartyStore) Encode() ([]byte, error) {
	var buf bytes.Buffer
	if err := pod.NewEncoder(&buf).Encode(s.state); err != nil {
		return nil, fmt.Errorf("party: failed to encode party state: %w", err)
	}
	return buf.Bytes(), nil
}

// Decode implements entity.EntityData: replaces the store's state with the
// pod-decoded PartyState — the load path. It restores PCID, Active, Pool,
// and RestPoint wholesale and bypasses the mutation API (no events fire
// during load, mirroring the Entity Framework's event-suppressed
// deserialization, GDD CR-8), and trusts the decoded value to satisfy the
// roster invariants. A decode failure leaves the state partially replaced;
// the caller (Save/Load) treats that as a corrupt blob — the same honesty
// as the entity framework's typed-store contract
// (component_store.go Decode).
func (s *PartyStore) Decode(data []byte) error {
	if err := pod.NewDecoder(bytes.NewReader(data)).Decode(&s.state); err != nil {
		return fmt.Errorf("party: failed to decode party state: %w", err)
	}
	return nil
}

// Compile-time assertion: PartyStore satisfies the EntityData serialization
// contract.
var _ entity.EntityData = (*PartyStore)(nil)

func init() {
	// PartyState must be registered with pod or encoding fails with "never
	// registered with pod" (ADR-0001, ADR-0002). A registration collision is
	// warned and skipped — the type is registered either way (the entity
	// framework's registerPodType and the save manager's metadata layout use
	// the same convention). GameEntityID is registered by the entity package,
	// which party imports, so its init runs first.
	if err := pod.Register(PartyState{}); err != nil {
		slog.Warn("pod registration skipped",
			"type", pod.QualifiedNameForLayout(PartyState{}), "err", err)
	}
}
