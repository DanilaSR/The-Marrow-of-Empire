// Package persistent_world_state implements Story 001 of the Persistent
// World State epic: the world.* flag store and query API.
//
// Story: production/epics/persistent-world-state/
//
//	story-001-world-flag-store-query-api.md
//
// GDD:  design/gdd/persistent-world-state.md (CR-1 World Flag Store, CR-4
//
//	State Queries)
//
// TR:   TR-pws-001 (world flag store: typed key-value per entity in the
//
//	world.* namespace), TR-pws-006 (query API: GetWorldFlags,
//	GetWorldFlag, GetEntitiesWithFlag, indexed on entity_id, flag-name,
//	region_id)
//
// ADR:  ADR-0001 (Entity Architecture) — PWS owns the world.* namespace of
//
//	the Entity Framework's Persistent component; flags are typed
//	key-value pairs stored per entity; PWS is the authoritative writer;
//	component access goes through a typed component store.
//
// Pure data structures, engine-agnostic: this package imports only the
// game-layer entity package for the component types (GameEntityID,
// Persistent, StateFlagValue) and never touches Kaiju engine APIs. Component
// store access is injected through the PersistentAccessor interface so the
// core logic is unit-testable without a running engine (ADR-0001: game code
// depends on interfaces, not engine internals).
//
// Concurrency: the store is not safe for concurrent use. The game loop is
// single-threaded and systems drive PWS from the main thread (GDD: events
// are processed in submission order, same-frame).
package persistent_world_state

import (
	"errors"
	"fmt"
	"log/slog"
	"reflect"
	"sort"
	"strings"

	"marrow-of-empire/entity"
)

// worldFlagPrefix is the namespace prefix owned by Persistent World State
// (GDD CR-1). Every flag this store accepts must begin with it; flags owned
// by other systems (trust.*, investigation.*, thread.*) are never touched.
const worldFlagPrefix = "world."

// DefaultMaxFlagsPerEntity is the GDD tuning knob `max_flags_per_entity`
// default (32): the soft per-entity cap on world.* flags. Exceeding it logs
// a warning and the flag is still written (mirrors the Entity Framework's
// MaxEntitiesPerRegion behavior — the cap is a memory-budget warning, not a
// correctness constraint).
const DefaultMaxFlagsPerEntity = 32

// Sentinel errors returned by the WorldFlagStore API. Callers may test with
// errors.Is; every error is wrapped from the underlying cause where one
// exists.
var (
	// ErrInvalidFlagName is returned when a flag name is not inside the
	// world.* namespace (GDD CR-1). PWS never reads or writes flags owned
	// by other systems.
	ErrInvalidFlagName = errors.New("persistent_world_state: flag name must be in the world.* namespace")

	// ErrUnsupportedValueType is returned when a flag value is not one of
	// the four typed kinds — bool, int, int64, float64, string (TR-pws-001).
	// The flag is not mutated.
	ErrUnsupportedValueType = errors.New("persistent_world_state: flag value must be bool, int, int64, float64, or string")

	// ErrTypeMismatch is returned when a flag value's kind does not match
	// the kind declared via RegisterFlagType (GDD CR-1 type schema). A
	// warning is logged and the flag is not mutated (AC-3).
	ErrTypeMismatch = errors.New("persistent_world_state: flag value type does not match the registered schema")

	// ErrNoPersistentComponent is returned when a write targets an entity
	// without a Persistent component (QA AC-1 edge case). PWS stores flags
	// in the Persistent component per ADR-0001; without it there is nowhere
	// to write.
	ErrNoPersistentComponent = errors.New("persistent_world_state: entity has no Persistent component")
)

// PersistentAccessor is the component-store surface Persistent World State
// requires (story implementation note: "component store integration via
// interface"). It reads and writes the entity's Persistent component. The
// Entity Framework's EntityStoreImpl satisfies it through a thin adapter
// (wired in Story 002 alongside the event pipeline); unit tests use an
// in-memory fake.
type PersistentAccessor interface {
	// GetPersistent returns the entity's Persistent component. The bool
	// reports whether the entity has one (false for entities without a
	// Persistent component, per ADR-0001 component store semantics).
	GetPersistent(id entity.GameEntityID) (entity.Persistent, bool)

	// SetPersistent stores the entity's Persistent component after a flag
	// mutation.
	SetPersistent(id entity.GameEntityID, p entity.Persistent) error
}

// WorldFlagStoreConfig carries the World Flag Store tuning knobs. Zero
// values fall back to the GDD defaults (mirrors the Entity Framework's
// EntityStoreConfig pattern; a later story wires these to external config
// files — gameplay values stay data-driven).
type WorldFlagStoreConfig struct {
	// MaxFlagsPerEntity is the GDD `max_flags_per_entity` soft cap (default
	// 32, safe range 8-64): the number of world.* flags on an entity beyond
	// which writes log a warning. The flag is still written — the cap is a
	// memory-budget warning, not a correctness constraint (GDD tuning knob
	// table).
	MaxFlagsPerEntity int
}

// WorldFlagStore owns the world.* namespace of the Persistent component
// (GDD CR-1, ADR-0001 write-ownership table). It is the authoritative
// writer of world.* flags; other systems read them through the CR-4 query
// API and request changes via PWS event submission (Story 002). No system
// writes world.* flags directly.
//
// The flag data itself lives in the entity's Persistent.StateFlags map
// (entity.StateFlagValue tagged union — bool/int64/float64/string,
// TR-pws-001, pod-safe per EF-005), so the store carries no per-flag
// payload of its own for live entities. The store additionally maintains
// five in-memory structures: the declared type schema
// (map[flag]StateFlagKind), the reverse index for GetEntitiesWithFlag
// (map[flagName]map[GameEntityID]bool, story implementation note), the
// entity -> region map for region scoping, the destroyed set
// (map[GameEntityID]bool, Story 004), and PWS-owned tombstone storage for
// destroyed entities' world.* flags
// (map[GameEntityID]map[string]StateFlagValue, Story 004 — the Entity
// Framework purges the Persistent component before entity.destroyed fires,
// so a destroyed entity's flags must live outside the component to stay
// queryable; Story 006 serializes the tombstones). The reverse index is
// maintained by SetWorldFlag and ClearWorldFlag (including tombstone
// writes); direct component mutation bypasses it (a load-time rebuild hook
// arrives with Story 006 save/load).
type WorldFlagStore struct {
	accessor          PersistentAccessor
	maxFlagsPerEntity int
	schema            map[string]entity.StateFlagKind
	reverseIndex      map[string]map[entity.GameEntityID]bool
	entityRegions     map[entity.GameEntityID]string
	destroyed         map[entity.GameEntityID]bool
	tombstones        map[entity.GameEntityID]map[string]entity.StateFlagValue
}

// NewWorldFlagStore creates a WorldFlagStore backed by the given component
// store accessor (dependency injection — no singletons). Zero-valued config
// fields fall back to GDD defaults (max_flags_per_entity = 32).
//
// Example usage:
//
//	store := persistent_world_state.NewWorldFlagStore(accessor,
//	    persistent_world_state.WorldFlagStoreConfig{})
func NewWorldFlagStore(accessor PersistentAccessor, config WorldFlagStoreConfig) *WorldFlagStore {
	if config.MaxFlagsPerEntity <= 0 {
		config.MaxFlagsPerEntity = DefaultMaxFlagsPerEntity
	}
	return &WorldFlagStore{
		accessor:          accessor,
		maxFlagsPerEntity: config.MaxFlagsPerEntity,
		schema:            make(map[string]entity.StateFlagKind),
		reverseIndex:      make(map[string]map[entity.GameEntityID]bool),
		entityRegions:     make(map[entity.GameEntityID]string),
		destroyed:         make(map[entity.GameEntityID]bool),
		tombstones:        make(map[entity.GameEntityID]map[string]entity.StateFlagValue),
	}
}

// RegisterFlagType declares the expected type for a flag (GDD CR-1 type
// schema, authored per region). expectedType must be one of bool, int,
// int64, float64, or string — the four TR-pws-001 kinds; int and int64 are
// equivalent because storage normalizes both to int64. Any other type
// (including nil) is rejected with a warning and the flag stays
// unregistered — unregistered flags accept any of the four kinds on write
// (QA edge case). A flag whose name is not in the world.* namespace is also
// rejected with a warning.
//
// Registering the same flag twice overwrites the schema.
func (s *WorldFlagStore) RegisterFlagType(flag string, expectedType reflect.Type) {
	if !isWorldFlagName(flag) {
		slog.Warn("world flag schema: flag name not in the world.* namespace; not registered",
			"flag", flag)
		return
	}
	if expectedType == nil {
		slog.Warn("world flag schema: nil expected type; flag not registered", "flag", flag)
		return
	}
	kind, ok := kindForType(expectedType)
	if !ok {
		slog.Warn("world flag schema: unsupported expected type; flag not registered",
			"flag", flag, "type", expectedType.String())
		return
	}
	s.schema[flag] = kind
}

// SetEntityRegion records the region an entity belongs to, for region
// scoping in GetEntitiesWithFlag (GDD CR-4: queries indexed on region_id).
// Idempotent: calling it repeatedly with the same region has no effect. An
// empty regionID unregisters the entity — it is then excluded from
// region-scoped queries and included in unfiltered ones. The store tracks
// regions itself; no entity component carries region data yet.
func (s *WorldFlagStore) SetEntityRegion(entityID entity.GameEntityID, regionID string) {
	if regionID == "" {
		delete(s.entityRegions, entityID)
		return
	}
	s.entityRegions[entityID] = regionID
}

// EntityRegion returns the region registered for the entity via
// SetEntityRegion, or "" when none was registered (GDD CR-4 region scoping).
// Story 004 uses it at destruction time to record where a quest-relevant
// entity was destroyed — the entity.destroyed event payload carries no
// region, only a position.
func (s *WorldFlagStore) EntityRegion(entityID entity.GameEntityID) string {
	return s.entityRegions[entityID]
}

// NoteEntityDestroyed records that the entity was destroyed, switching its
// world.* flag writes from the Persistent component to PWS-owned tombstone
// storage (Story 004, AC-1). The Entity Framework purges component stores
// before entity.destroyed fires, so a destroyed entity's flags cannot live in
// the component — tombstone storage keeps them queryable after destruction
// (AC-3).
//
// This is the only way an entity enters the destroyed set: tombstones are
// never created for entities the store has not seen destroyed, so entities
// that never existed never leak storage. Idempotent — safe to call more than
// once (the Entity Framework emits entity.destroyed exactly once).
//
// Story 006 (save/load) serializes the tombstone storage.
func (s *WorldFlagStore) NoteEntityDestroyed(entityID entity.GameEntityID) {
	s.destroyed[entityID] = true
}

// SetWorldFlag writes a typed flag on an entity's Persistent component
// (AC-1). Validation order:
//
//  1. The flag name must be in the world.* namespace (ErrInvalidFlagName).
//     Writes to flags owned by other systems are never forwarded — the
//     component is not touched.
//  2. The value must be one of the four typed kinds — bool, int, int64,
//     float64, string (ErrUnsupportedValueType). int is normalized to int64
//     for storage (TR-pws-001 storage kind).
//  3. When a schema was declared via RegisterFlagType, the value kind must
//     match it (ErrTypeMismatch, warning logged, flag not mutated — AC-3).
//  4. The entity must have a Persistent component — or be known destroyed
//     (Story 004): an entity with a component receives the write on the
//     component; a destroyed entity (NoteEntityDestroyed, component already
//     purged by the Entity Framework) receives the write on its PWS-owned
//     tombstone; any other entity gets ErrNoPersistentComponent (QA AC-1
//     edge case). Tombstone entries are created lazily on first write, so
//     destroyed entities that never had flags written hold no storage
//     beyond their destroyed-set entry.
//
// Exceeding max_flags_per_entity logs a warning but the write still
// succeeds (soft limit). Overwriting an existing flag is last-write-wins
// (QA AC-1 edge case).
//
// The reverse index entry is presence-based: an entity is listed by
// GetEntitiesWithFlag while it has the flag, regardless of its value (a
// false or zero value still counts — the query answers "which entities have
// this flag", e.g. "show cleared ruins"). Tombstone writes are indexed the
// same way, so destroyed entities stay discoverable (Story 004).
func (s *WorldFlagStore) SetWorldFlag(entityID entity.GameEntityID, flag string, value any) error {
	if !isWorldFlagName(flag) {
		return ErrInvalidFlagName
	}
	sfv, err := toStateFlagValue(value)
	if err != nil {
		return err
	}
	if kind, ok := s.schema[flag]; ok && kind != sfv.Kind {
		slog.Warn("world flag write skipped: value type does not match registered schema",
			"entity", entityID, "flag", flag,
			"expected", kind, "got", sfv.Kind)
		return ErrTypeMismatch
	}

	p, ok := s.accessor.GetPersistent(entityID)
	if !ok {
		// Story 004 (clue-path protection): the Entity Framework purges
		// component stores before entity.destroyed fires, so a destroyed
		// entity has no Persistent component. When the store has seen the
		// entity destroyed (NoteEntityDestroyed), the flag is written to
		// PWS-owned tombstone storage instead — never into the component.
		// Entities the store has never seen destroyed keep the Story 001
		// behavior (ErrNoPersistentComponent): tombstones never materialize
		// for entities that never existed.
		if !s.destroyed[entityID] {
			return ErrNoPersistentComponent
		}
		return s.setTombstoneFlag(entityID, flag, sfv)
	}
	if p.StateFlags == nil {
		p.StateFlags = make(map[string]entity.StateFlagValue)
	}

	if _, exists := p.StateFlags[flag]; !exists && s.worldFlagsCount(p) >= s.maxFlagsPerEntity {
		slog.Warn("max_flags_per_entity exceeded; flag still written",
			"entity", entityID, "flag", flag, "max", s.maxFlagsPerEntity)
	}

	p.StateFlags[flag] = sfv
	if err := s.accessor.SetPersistent(entityID, p); err != nil {
		return fmt.Errorf("persistent_world_state: failed to persist flag %q: %w", flag, err)
	}
	s.indexFlag(entityID, flag)
	return nil
}

// GetWorldFlag reads a single flag (GDD CR-4). Returns the typed value and
// found=true, or (nil, false) when the flag was never set, when the entity
// has no Persistent component and no tombstone entry, or when the flag name
// is outside the world.* namespace (AC-4). Integer flags read back as int64 —
// the TR-pws-001 storage kind.
//
// Lookup order (Story 004): the live Persistent component is checked first
// and is authoritative while the entity exists; the PWS-owned tombstone is
// the fallback for destroyed entities, whose component was purged before
// entity.destroyed fired. The two never conflict — tombstone entries exist
// only for destroyed entities — so the order only matters for destroyed
// entities, where the component is gone. This keeps the destroyed entity's
// flags queryable after destruction (AC-3: the Investigation Journal can
// discover world.<id>.destroyed_with_clues).
func (s *WorldFlagStore) GetWorldFlag(entityID entity.GameEntityID, flag string) (any, bool) {
	if !isWorldFlagName(flag) {
		return nil, false
	}
	if p, ok := s.accessor.GetPersistent(entityID); ok {
		if v, found := p.StateFlags[flag]; found {
			return fromStateFlagValue(v), true
		}
	}
	if tf, ok := s.tombstones[entityID]; ok {
		if v, found := tf[flag]; found {
			return fromStateFlagValue(v), true
		}
	}
	return nil, false
}

// GetWorldFlags reads every world.* flag of an entity (GDD CR-4: all world.*
// flags as map[string]interface{}). Returns a fresh map — callers may mutate
// it freely — that is never nil: an entity with no flags (or no Persistent
// component and no tombstone) yields an empty map (QA AC-5 edge case).
// Integer flags read back as int64. Flags owned by other namespaces
// (trust.*, investigation.*, thread.*) are not included.
//
// The live component's flags and the destroyed entity's tombstone flags
// (Story 004) are merged; a destroyed entity's flags remain readable after
// the component purge (AC-3).
func (s *WorldFlagStore) GetWorldFlags(entityID entity.GameEntityID) map[string]any {
	result := make(map[string]any)
	if p, ok := s.accessor.GetPersistent(entityID); ok {
		for name, v := range p.StateFlags {
			if strings.HasPrefix(name, worldFlagPrefix) {
				result[name] = fromStateFlagValue(v)
			}
		}
	}
	if tf, ok := s.tombstones[entityID]; ok {
		for name, v := range tf {
			if strings.HasPrefix(name, worldFlagPrefix) {
				result[name] = fromStateFlagValue(v)
			}
		}
	}
	return result
}

// GetEntitiesWithFlag returns every entity that has the flag set, in
// ascending GameEntityID order (deterministic, mirroring the Entity
// Framework's registration order) (GDD CR-4, AC-2). When regionID is
// non-empty, only entities registered to that region via SetEntityRegion
// are returned; an empty regionID disables the region filter (all regions —
// the QA "nil regionID" case; Go strings cannot be nil). Entities with no
// registered region are excluded from region-scoped queries. Flags never
// written return an empty slice. The lookup uses the store's reverse index,
// so it is O(entities indexed for the flag), not a scan of all entities.
//
// Destroyed entities are included: tombstone writes are reverse-indexed like
// live writes (Story 004), and region scoping uses the region registered via
// SetEntityRegion before destruction — so a destroyed clue carrier still
// answers "which entities have destroyed_with_clues in region X".
func (s *WorldFlagStore) GetEntitiesWithFlag(flag string, regionID string) []entity.GameEntityID {
	ids := make([]entity.GameEntityID, 0, len(s.reverseIndex[flag]))
	for id := range s.reverseIndex[flag] {
		if regionID != "" && s.entityRegions[id] != regionID {
			continue
		}
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	return ids
}

// ClearWorldFlag removes a flag from an entity and drops it from the reverse
// index (QA AC-4 edge case: flag set then cleared). Idempotent: clearing an
// absent flag, or clearing from an entity without a Persistent component and
// without a tombstone entry, is a no-op success (mirrors the Entity
// Framework's RemoveComponentData semantics). A tombstoned flag (Story 004)
// is cleared from the tombstone, which shrinks back to nothing when empty.
// Non-world.* names are rejected with ErrInvalidFlagName — PWS never mutates
// other systems' namespaces.
func (s *WorldFlagStore) ClearWorldFlag(entityID entity.GameEntityID, flag string) error {
	if !isWorldFlagName(flag) {
		return ErrInvalidFlagName
	}
	cleared := false
	if p, ok := s.accessor.GetPersistent(entityID); ok {
		if _, exists := p.StateFlags[flag]; exists {
			delete(p.StateFlags, flag)
			if err := s.accessor.SetPersistent(entityID, p); err != nil {
				return fmt.Errorf("persistent_world_state: failed to persist cleared flag %q: %w", flag, err)
			}
			cleared = true
		}
	}
	if tf, ok := s.tombstones[entityID]; ok {
		if _, exists := tf[flag]; exists {
			delete(tf, flag)
			if len(tf) == 0 {
				delete(s.tombstones, entityID)
			}
			cleared = true
		}
	}
	if cleared {
		if ids := s.reverseIndex[flag]; ids != nil {
			delete(ids, entityID)
			if len(ids) == 0 {
				delete(s.reverseIndex, flag)
			}
		}
	}
	return nil
}

// setTombstoneFlag writes a world.* flag into the destroyed entity's
// PWS-owned tombstone (Story 004, AC-1). Tombstone maps are created lazily
// on first write, so destroyed entities that never had world.* flags written
// hold no storage beyond their destroyed-set entry. The max_flags_per_entity
// soft cap applies to the tombstone exactly as it applies to the live
// component — a warning, never a rejection (len(tf) is the world.* count:
// SetWorldFlag validates the namespace before routing here, so the tombstone
// never holds foreign flags). Reverse-index entries are written so
// GetEntitiesWithFlag still finds destroyed entities.
func (s *WorldFlagStore) setTombstoneFlag(entityID entity.GameEntityID, flag string, sfv entity.StateFlagValue) error {
	tf := s.tombstones[entityID]
	if tf == nil {
		tf = make(map[string]entity.StateFlagValue)
		s.tombstones[entityID] = tf
	}
	if _, exists := tf[flag]; !exists && len(tf) >= s.maxFlagsPerEntity {
		slog.Warn("max_flags_per_entity exceeded for destroyed entity; flag still written",
			"entity", entityID, "flag", flag, "max", s.maxFlagsPerEntity)
	}
	tf[flag] = sfv
	s.indexFlag(entityID, flag)
	return nil
}

// isWorldFlagName reports whether flag is a valid world.* flag name: it must
// start with the namespace prefix and carry a non-empty remainder.
func isWorldFlagName(flag string) bool {
	return len(flag) > len(worldFlagPrefix) && strings.HasPrefix(flag, worldFlagPrefix)
}

// toStateFlagValue converts a public-API value to its typed storage form
// (entity.StateFlagValue, TR-pws-001). Exactly four kinds are accepted —
// bool, int, int64, float64, string — matching the GDD flag type list
// (bool/int/float/string). int is normalized to int64 (pod-safe storage,
// EF-005). Any other type (float32, unsigned integers, nil, structs) is
// rejected with ErrUnsupportedValueType: strict typing, per QA "schema
// declares float but int written (reject — strict typing)".
func toStateFlagValue(value any) (entity.StateFlagValue, error) {
	switch v := value.(type) {
	case bool:
		return entity.StateFlagValue{Kind: entity.StateFlagBool, Bool: v}, nil
	case int:
		return entity.StateFlagValue{Kind: entity.StateFlagInt, Int: int64(v)}, nil
	case int64:
		return entity.StateFlagValue{Kind: entity.StateFlagInt, Int: v}, nil
	case float64:
		return entity.StateFlagValue{Kind: entity.StateFlagFloat, Float: v}, nil
	case string:
		return entity.StateFlagValue{Kind: entity.StateFlagString, Str: v}, nil
	default:
		return entity.StateFlagValue{}, ErrUnsupportedValueType
	}
}

// fromStateFlagValue converts a stored value back to its public Go
// representation: bool, int64, float64, or string (TR-pws-001). Integer
// flags read back as int64 — the storage kind.
func fromStateFlagValue(v entity.StateFlagValue) any {
	switch v.Kind {
	case entity.StateFlagBool:
		return v.Bool
	case entity.StateFlagInt:
		return v.Int
	case entity.StateFlagFloat:
		return v.Float
	case entity.StateFlagString:
		return v.Str
	default:
		return nil
	}
}

// kindForType maps a registered schema type to its StateFlagKind. bool, int,
// int64, float64, and string are the supported schema types (int and int64
// are equivalent — storage normalizes both to int64); any other reflect
// type is rejected.
func kindForType(t reflect.Type) (entity.StateFlagKind, bool) {
	switch t {
	case reflect.TypeOf(true):
		return entity.StateFlagBool, true
	case reflect.TypeOf(int(0)), reflect.TypeOf(int64(0)):
		return entity.StateFlagInt, true
	case reflect.TypeOf(float64(0)):
		return entity.StateFlagFloat, true
	case reflect.TypeOf(""):
		return entity.StateFlagString, true
	default:
		return "", false
	}
}

// worldFlagsCount counts the world.* flags currently on the entity's
// Persistent component. The max_flags_per_entity soft cap applies to PWS's
// own namespace only — other systems' flags (trust.*, investigation.*) do
// not count against it.
func (s *WorldFlagStore) worldFlagsCount(p entity.Persistent) int {
	count := 0
	for name := range p.StateFlags {
		if strings.HasPrefix(name, worldFlagPrefix) {
			count++
		}
	}
	return count
}

// indexFlag adds the entity to the flag's reverse-index entry
// (map[flagName]map[GameEntityID]bool, story implementation note). Presence
// is what matters — the value is not stored here.
func (s *WorldFlagStore) indexFlag(entityID entity.GameEntityID, flag string) {
	ids, ok := s.reverseIndex[flag]
	if !ok {
		ids = make(map[entity.GameEntityID]bool)
		s.reverseIndex[flag] = ids
	}
	ids[entityID] = true
}
