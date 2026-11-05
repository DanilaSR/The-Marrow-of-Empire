// Unit tests for Story 001 of the Persistent World State epic: World Flag
// Store & Query API.
//
// Evidence path: production/epics/persistent-world-state/
//
//	story-001-world-flag-store-query-api.md
//
// Tests map to the five acceptance criteria (AC-1 set/get round-trip,
// AC-2 GetEntitiesWithFlag region scoping, AC-3 type-schema rejection,
// AC-4 missing flag returns not-found, AC-5 GetWorldFlags returns all
// flags) and the QA edge cases: overwrite last-write-wins, write on an
// entity without a Persistent component (error), empty/unfiltered/never-set
// region queries, strict float-vs-int typing, unregistered flags accepting
// any type, flags set then cleared, the max_flags_per_entity soft cap
// (custom and default 32), and deterministic ascending result order.
// Requirements TR-pws-001 and TR-pws-006; governing ADR: ADR-0001.
package persistent_world_state_test

import (
	"errors"
	"fmt"
	"reflect"
	"testing"

	"marrow-of-empire/entity"
	"marrow-of-empire/persistent_world_state"
)

// fakeAccessor is an in-memory PersistentAccessor for unit tests: a plain
// map of Persistent components mirroring the Entity Framework component
// store contract (GetComponentData/SetComponentData semantics — absent
// entity = no component).
type fakeAccessor struct {
	components map[entity.GameEntityID]entity.Persistent
}

func newFakeAccessor() *fakeAccessor {
	return &fakeAccessor{components: make(map[entity.GameEntityID]entity.Persistent)}
}

func (f *fakeAccessor) GetPersistent(id entity.GameEntityID) (entity.Persistent, bool) {
	p, ok := f.components[id]
	return p, ok
}

func (f *fakeAccessor) SetPersistent(id entity.GameEntityID, p entity.Persistent) error {
	f.components[id] = p
	return nil
}

// newTestStore returns a store with a fresh fake accessor and GDD-default
// config.
func newTestStore() (*persistent_world_state.WorldFlagStore, *fakeAccessor) {
	acc := newFakeAccessor()
	store := persistent_world_state.NewWorldFlagStore(acc, persistent_world_state.WorldFlagStoreConfig{})
	return store, acc
}

// addEntity registers an entity with a Persistent component in the fake
// accessor, and optionally its region in the store.
func addEntity(acc *fakeAccessor, store *persistent_world_state.WorldFlagStore, id entity.GameEntityID, region string) {
	acc.components[id] = entity.Persistent{StateFlags: make(map[string]entity.StateFlagValue)}
	if region != "" {
		store.SetEntityRegion(id, region)
	}
}

// TestSetWorldFlagRoundTrip covers AC-1: a flag written via SetWorldFlag is
// returned by GetWorldFlag with the same typed value and found=true. int is
// normalized to int64 for storage (TR-pws-001 storage kind), so it reads
// back as int64.
func TestSetWorldFlagRoundTrip(t *testing.T) {
	store, acc := newTestStore()
	addEntity(acc, store, 1, "")

	tests := []struct {
		name  string
		value any
		want  any
	}{
		{name: "bool", value: true, want: true},
		{name: "int", value: 42, want: int64(42)},
		{name: "int64", value: int64(42), want: int64(42)},
		{name: "float", value: 0.5, want: 0.5},
		{name: "string", value: "bandits", want: "bandits"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := store.SetWorldFlag(1, "world.ruin_cleared", tt.value); err != nil {
				t.Fatalf("SetWorldFlag returned unexpected error: %v", err)
			}
			got, found := store.GetWorldFlag(1, "world.ruin_cleared")
			if !found {
				t.Fatal("GetWorldFlag found=false, want true")
			}
			if got != tt.want {
				t.Errorf("GetWorldFlag = %v (%T), want %v (%T)", got, got, tt.want, tt.want)
			}
		})
	}
}

// TestSetWorldFlagOverwriteLastWriteWins covers the AC-1 QA edge case:
// overwriting an existing flag keeps the latest value, and the entity stays
// indexed exactly once by the reverse index.
func TestSetWorldFlagOverwriteLastWriteWins(t *testing.T) {
	store, acc := newTestStore()
	addEntity(acc, store, 1, "")

	if err := store.SetWorldFlag(1, "world.ruin_cleared", true); err != nil {
		t.Fatalf("first write: %v", err)
	}
	if err := store.SetWorldFlag(1, "world.ruin_cleared", false); err != nil {
		t.Fatalf("overwrite: %v", err)
	}
	got, found := store.GetWorldFlag(1, "world.ruin_cleared")
	if !found || got != false {
		t.Errorf("GetWorldFlag after overwrite = (%v, %v), want (false, true)", got, found)
	}
	if ids := store.GetEntitiesWithFlag("world.ruin_cleared", ""); len(ids) != 1 {
		t.Errorf("GetEntitiesWithFlag after overwrite = %v, want exactly 1 id", ids)
	}
}

// TestSetWorldFlagEntityWithoutPersistentComponent covers the AC-1 QA edge
// case: writing a flag to an entity that has no Persistent component fails
// with ErrNoPersistentComponent and nothing is indexed.
func TestSetWorldFlagEntityWithoutPersistentComponent(t *testing.T) {
	store, _ := newTestStore()

	err := store.SetWorldFlag(7, "world.ruin_cleared", true)
	if !errors.Is(err, persistent_world_state.ErrNoPersistentComponent) {
		t.Fatalf("SetWorldFlag error = %v, want ErrNoPersistentComponent", err)
	}
	if _, found := store.GetWorldFlag(7, "world.ruin_cleared"); found {
		t.Error("GetWorldFlag found=true, want false")
	}
	if ids := store.GetEntitiesWithFlag("world.ruin_cleared", ""); len(ids) != 0 {
		t.Errorf("GetEntitiesWithFlag = %v, want empty", ids)
	}
}

// TestSetWorldFlagRejectsNonWorldPrefix covers CR-1: only the world.*
// namespace is writable. Other systems' namespaces (trust.*), bare names,
// and the bare prefix itself are rejected with ErrInvalidFlagName and
// nothing is stored. Registering a schema for a non-world.* name is also
// rejected (warning only — the namespace check still governs writes).
func TestSetWorldFlagRejectsNonWorldPrefix(t *testing.T) {
	store, acc := newTestStore()
	addEntity(acc, store, 1, "")

	for _, flag := range []string{"trust.some_flag", "ruin_cleared", "world."} {
		t.Run(flag, func(t *testing.T) {
			err := store.SetWorldFlag(1, flag, true)
			if !errors.Is(err, persistent_world_state.ErrInvalidFlagName) {
				t.Errorf("SetWorldFlag(%q) error = %v, want ErrInvalidFlagName", flag, err)
			}
		})
	}

	store.RegisterFlagType("trust.some_flag", reflect.TypeOf(true))
	if err := store.SetWorldFlag(1, "trust.some_flag", true); !errors.Is(err, persistent_world_state.ErrInvalidFlagName) {
		t.Errorf("write after schema registration on foreign namespace: error = %v, want ErrInvalidFlagName", err)
	}

	if flags := store.GetWorldFlags(1); len(flags) != 0 {
		t.Errorf("GetWorldFlags = %v, want empty", flags)
	}
}

// TestSetWorldFlagRejectsUnsupportedValueTypes covers strict typing
// (TR-pws-001): only bool, int, int64, float64, and string are writable.
// Every other Go type is rejected with ErrUnsupportedValueType and the flag
// is not mutated.
func TestSetWorldFlagRejectsUnsupportedValueTypes(t *testing.T) {
	store, acc := newTestStore()
	addEntity(acc, store, 1, "")

	values := []any{
		float32(1.5),
		int8(1),
		int32(1),
		uint(1),
		uint64(1),
		nil,
		struct{}{},
		[]string{"a"},
		map[string]int{"a": 1},
	}
	for _, value := range values {
		t.Run(fmt.Sprintf("%T", value), func(t *testing.T) {
			err := store.SetWorldFlag(1, "world.bad_value", value)
			if !errors.Is(err, persistent_world_state.ErrUnsupportedValueType) {
				t.Errorf("SetWorldFlag(%T) error = %v, want ErrUnsupportedValueType", value, err)
			}
			if v, found := store.GetWorldFlag(1, "world.bad_value"); found {
				t.Errorf("GetWorldFlag = (%v, true), want (nil, false) — flag must not be mutated", v)
			}
		})
	}
}

// TestRegisterFlagTypeMismatchedWriteRejected covers AC-3: a write whose
// value kind differs from the registered schema kind is rejected with
// ErrTypeMismatch, a warning is logged, and the flag is not mutated — an
// existing value survives the rejected write. Strict typing per QA: int
// schema rejects string, float64 schema rejects int, bool schema rejects
// float, string schema rejects bool.
func TestRegisterFlagTypeMismatchedWriteRejected(t *testing.T) {
	store, acc := newTestStore()
	for i := 1; i <= 4; i++ {
		addEntity(acc, store, entity.GameEntityID(i), "")
	}

	tests := []struct {
		name       string
		schemaType reflect.Type
		flag       string
		good       any
		want       any
		bad        any
	}{
		{name: "int schema rejects string", schemaType: reflect.TypeOf(int(0)), flag: "world.bandit_count", good: 5, want: int64(5), bad: "many"},
		{name: "float schema rejects int", schemaType: reflect.TypeOf(float64(0)), flag: "world.faction_influence", good: 0.25, want: 0.25, bad: 3},
		{name: "bool schema rejects float", schemaType: reflect.TypeOf(true), flag: "world.ruin_cleared", good: true, want: true, bad: 1.0},
		{name: "string schema rejects bool", schemaType: reflect.TypeOf(""), flag: "world.occupant", good: "hermit", want: "hermit", bad: true},
	}
	for i, tt := range tests {
		id := entity.GameEntityID(i + 1)
		store.RegisterFlagType(tt.flag, tt.schemaType)
		if err := store.SetWorldFlag(id, tt.flag, tt.good); err != nil {
			t.Fatalf("%s: seed write failed: %v", tt.name, err)
		}
		t.Run(tt.name, func(t *testing.T) {
			if err := store.SetWorldFlag(id, tt.flag, tt.bad); !errors.Is(err, persistent_world_state.ErrTypeMismatch) {
				t.Fatalf("SetWorldFlag error = %v, want ErrTypeMismatch", err)
			}
			got, found := store.GetWorldFlag(id, tt.flag)
			if !found || got != tt.want {
				t.Errorf("GetWorldFlag = (%v, %v), want (%v, true) — rejected write must not mutate", got, found, tt.want)
			}
		})
	}
}

// TestRegisterFlagTypeIntAndInt64Equivalent covers schema normalization: int
// and int64 schemas are the same kind (storage is int64), so a write of
// either integer type satisfies both.
func TestRegisterFlagTypeIntAndInt64Equivalent(t *testing.T) {
	store, acc := newTestStore()
	addEntity(acc, store, 1, "")

	store.RegisterFlagType("world.bandit_count", reflect.TypeOf(int(0)))
	if err := store.SetWorldFlag(1, "world.bandit_count", int64(7)); err != nil {
		t.Errorf("int64 write against int schema failed: %v", err)
	}
	store.RegisterFlagType("world.days_since", reflect.TypeOf(int64(0)))
	if err := store.SetWorldFlag(1, "world.days_since", 3); err != nil {
		t.Errorf("int write against int64 schema failed: %v", err)
	}
}

// TestRegisterFlagTypeUnregisteredFlagAcceptsAnyType covers the QA edge
// case: a flag with no registered schema accepts any of the four typed
// kinds.
func TestRegisterFlagTypeUnregisteredFlagAcceptsAnyType(t *testing.T) {
	store, acc := newTestStore()
	addEntity(acc, store, 1, "")

	for _, value := range []any{true, 3, 0.25, "hermit"} {
		if err := store.SetWorldFlag(1, "world.free_flag", value); err != nil {
			t.Fatalf("SetWorldFlag(%v) failed: %v", value, err)
		}
	}
	flags := store.GetWorldFlags(1)
	if len(flags) != 1 {
		t.Fatalf("GetWorldFlags = %v, want exactly 1 flag (last write wins)", flags)
	}
	if got := flags["world.free_flag"]; got != "hermit" {
		t.Errorf("world.free_flag = %v, want \"hermit\"", got)
	}
}

// TestRegisterFlagTypeUnsupportedSchemaTypeRejected: registering an
// unsupported reflect type (float32) or nil logs a warning and leaves the
// flag unregistered — later writes of any supported kind are accepted.
func TestRegisterFlagTypeUnsupportedSchemaTypeRejected(t *testing.T) {
	store, acc := newTestStore()
	addEntity(acc, store, 1, "")

	store.RegisterFlagType("world.odd_flag", reflect.TypeOf(float32(0)))
	store.RegisterFlagType("world.other_flag", nil)
	if err := store.SetWorldFlag(1, "world.odd_flag", 2.5); err != nil {
		t.Errorf("write after unsupported schema registration failed: %v", err)
	}
	if err := store.SetWorldFlag(1, "world.other_flag", true); err != nil {
		t.Errorf("write after nil schema registration failed: %v", err)
	}
	if v, found := store.GetWorldFlag(1, "world.odd_flag"); !found || v != 2.5 {
		t.Errorf("GetWorldFlag(world.odd_flag) = (%v, %v), want (2.5, true)", v, found)
	}
}

// TestGetWorldFlagMissingReturnsNotFound covers AC-4: a flag that was never
// set returns (nil, false), whether the entity has a Persistent component
// or not, and other systems' namespaces are not visible through the query
// API.
func TestGetWorldFlagMissingReturnsNotFound(t *testing.T) {
	store, acc := newTestStore()
	addEntity(acc, store, 1, "")

	if v, found := store.GetWorldFlag(1, "world.never_set"); found || v != nil {
		t.Errorf("GetWorldFlag(1, world.never_set) = (%v, %v), want (nil, false)", v, found)
	}
	if v, found := store.GetWorldFlag(999, "world.never_set"); found || v != nil {
		t.Errorf("GetWorldFlag(999, world.never_set) = (%v, %v), want (nil, false)", v, found)
	}
	if v, found := store.GetWorldFlag(1, "trust.some_flag"); found || v != nil {
		t.Errorf("GetWorldFlag(1, trust.some_flag) = (%v, %v), want (nil, false)", v, found)
	}
}

// TestGetEntitiesWithFlagRegionScoping covers AC-2: with five entities —
// three in region "A" carrying world.ruin_cleared, one in region "B"
// carrying it, one in region "A" without it — the region-scoped query
// returns exactly the three region-A ids in ascending order. Entities in
// other regions and flagless entities are excluded, an unknown region
// returns 0 results (QA "empty region"), and an unfiltered query returns
// all four carriers (QA "nil regionID" — empty string).
func TestGetEntitiesWithFlagRegionScoping(t *testing.T) {
	store, acc := newTestStore()
	addEntity(acc, store, 1, "A")
	addEntity(acc, store, 2, "A")
	addEntity(acc, store, 3, "A")
	addEntity(acc, store, 4, "B")
	addEntity(acc, store, 5, "A") // no flag

	for _, id := range []entity.GameEntityID{1, 2, 3, 4} {
		if err := store.SetWorldFlag(id, "world.ruin_cleared", true); err != nil {
			t.Fatalf("SetWorldFlag(%d): %v", id, err)
		}
	}

	got := store.GetEntitiesWithFlag("world.ruin_cleared", "A")
	want := []entity.GameEntityID{1, 2, 3}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("GetEntitiesWithFlag(region A) = %v, want %v", got, want)
	}

	got = store.GetEntitiesWithFlag("world.ruin_cleared", "")
	want = []entity.GameEntityID{1, 2, 3, 4}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("GetEntitiesWithFlag(all regions) = %v, want %v", got, want)
	}

	got = store.GetEntitiesWithFlag("world.ruin_cleared", "B")
	want = []entity.GameEntityID{4}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("GetEntitiesWithFlag(region B) = %v, want %v", got, want)
	}

	if got := store.GetEntitiesWithFlag("world.ruin_cleared", "C"); len(got) != 0 {
		t.Errorf("GetEntitiesWithFlag(unknown region) = %v, want empty", got)
	}
	if got := store.GetEntitiesWithFlag("world.never_set", ""); len(got) != 0 {
		t.Errorf("GetEntitiesWithFlag(never-set flag) = %v, want empty", got)
	}
}

// TestGetEntitiesWithFlagUnregisteredRegion: an entity with the flag but no
// registered region is excluded from region-scoped queries and included in
// unfiltered ones.
func TestGetEntitiesWithFlagUnregisteredRegion(t *testing.T) {
	store, acc := newTestStore()
	addEntity(acc, store, 1, "") // no region registered
	if err := store.SetWorldFlag(1, "world.ruin_cleared", true); err != nil {
		t.Fatalf("SetWorldFlag: %v", err)
	}

	if got := store.GetEntitiesWithFlag("world.ruin_cleared", "A"); len(got) != 0 {
		t.Errorf("region-scoped query = %v, want empty (entity has no region)", got)
	}
	if got := store.GetEntitiesWithFlag("world.ruin_cleared", ""); len(got) != 1 {
		t.Errorf("unfiltered query = %v, want [1]", got)
	}
}

// TestGetEntitiesWithFlagDeterministicOrder: results are ascending by
// GameEntityID regardless of write order (Go map iteration is
// non-deterministic, so the store sorts).
func TestGetEntitiesWithFlagDeterministicOrder(t *testing.T) {
	store, acc := newTestStore()
	for _, id := range []entity.GameEntityID{10, 2, 5} {
		addEntity(acc, store, id, "")
		if err := store.SetWorldFlag(id, "world.ruin_cleared", true); err != nil {
			t.Fatalf("SetWorldFlag(%d): %v", id, err)
		}
	}

	got := store.GetEntitiesWithFlag("world.ruin_cleared", "")
	want := []entity.GameEntityID{2, 5, 10}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("GetEntitiesWithFlag = %v, want %v", got, want)
	}
}

// TestGetWorldFlagsReturnsAllFlags covers AC-5: all world.* flags are
// returned in one map with their correct typed values (AC-5: world.a = true,
// world.b = 42, world.c = "hello"), and flags owned by other namespaces are
// not included.
func TestGetWorldFlagsReturnsAllFlags(t *testing.T) {
	store, acc := newTestStore()
	acc.components[1] = entity.Persistent{
		StateFlags: map[string]entity.StateFlagValue{
			"world.a":         {Kind: entity.StateFlagBool, Bool: true},
			"world.b":         {Kind: entity.StateFlagInt, Int: 42},
			"world.c":         {Kind: entity.StateFlagString, Str: "hello"},
			"trust.some_flag": {Kind: entity.StateFlagBool, Bool: true}, // another system's namespace
		},
	}

	flags := store.GetWorldFlags(1)
	if len(flags) != 3 {
		t.Fatalf("GetWorldFlags = %v, want exactly 3 world.* flags", flags)
	}
	if v := flags["world.a"]; v != true {
		t.Errorf("world.a = %v (%T), want true (bool)", v, v)
	}
	if v := flags["world.b"]; v != int64(42) {
		t.Errorf("world.b = %v (%T), want int64(42)", v, v)
	}
	if v := flags["world.c"]; v != "hello" {
		t.Errorf("world.c = %v (%T), want \"hello\"", v, v)
	}
	if _, ok := flags["trust.some_flag"]; ok {
		t.Error("trust.some_flag leaked into GetWorldFlags")
	}
}

// TestGetWorldFlagsEmptyMapNotNull covers the QA AC-5 edge case: an entity
// with a Persistent component but no flags yields an empty, non-nil map; the
// same holds for entities without a component. The returned map is a copy —
// mutating it does not corrupt the store.
func TestGetWorldFlagsEmptyMapNotNull(t *testing.T) {
	store, acc := newTestStore()
	addEntity(acc, store, 1, "")

	flags := store.GetWorldFlags(1)
	if flags == nil || len(flags) != 0 {
		t.Errorf("GetWorldFlags(no flags) = %#v, want empty non-nil map", flags)
	}
	flags = store.GetWorldFlags(999)
	if flags == nil || len(flags) != 0 {
		t.Errorf("GetWorldFlags(no component) = %#v, want empty non-nil map", flags)
	}

	flags["world.x"] = true
	if _, ok := store.GetWorldFlags(1)["world.x"]; ok {
		t.Error("mutating the returned map leaked into the store")
	}
}

// TestClearWorldFlag covers the AC-4 QA edge case "flag set then
// deleted/cleared": after clearing, the flag reads not-found and the reverse
// index no longer lists the entity, while other entities' entries survive.
// Clearing an absent flag, an absent entity's flag, and re-clearing are
// idempotent no-ops; non-world.* names are rejected.
func TestClearWorldFlag(t *testing.T) {
	store, acc := newTestStore()
	addEntity(acc, store, 1, "A")
	addEntity(acc, store, 2, "A")
	if err := store.SetWorldFlag(1, "world.ruin_cleared", true); err != nil {
		t.Fatalf("SetWorldFlag(1): %v", err)
	}
	if err := store.SetWorldFlag(2, "world.ruin_cleared", true); err != nil {
		t.Fatalf("SetWorldFlag(2): %v", err)
	}

	if err := store.ClearWorldFlag(1, "world.ruin_cleared"); err != nil {
		t.Fatalf("ClearWorldFlag: %v", err)
	}
	if v, found := store.GetWorldFlag(1, "world.ruin_cleared"); found || v != nil {
		t.Errorf("GetWorldFlag after clear = (%v, %v), want (nil, false)", v, found)
	}
	if got := store.GetEntitiesWithFlag("world.ruin_cleared", ""); !reflect.DeepEqual(got, []entity.GameEntityID{2}) {
		t.Errorf("GetEntitiesWithFlag after partial clear = %v, want [2]", got)
	}

	if err := store.ClearWorldFlag(1, "world.ruin_cleared"); err != nil {
		t.Errorf("re-clear returned error: %v", err)
	}
	if err := store.ClearWorldFlag(999, "world.ruin_cleared"); err != nil {
		t.Errorf("clear on entity without component returned error: %v", err)
	}
	if err := store.ClearWorldFlag(1, "trust.some_flag"); !errors.Is(err, persistent_world_state.ErrInvalidFlagName) {
		t.Errorf("ClearWorldFlag(trust.some_flag) error = %v, want ErrInvalidFlagName", err)
	}
}

// TestMaxFlagsPerEntitySoftLimit covers the GDD `max_flags_per_entity`
// tuning knob: writes beyond the cap log a warning but still succeed (soft
// limit, mirroring the Entity Framework's MaxEntitiesPerRegion), and the
// cap counts only world.* flags — other systems' flags do not count against
// it.
func TestMaxFlagsPerEntitySoftLimit(t *testing.T) {
	acc := newFakeAccessor()
	store := persistent_world_state.NewWorldFlagStore(acc,
		persistent_world_state.WorldFlagStoreConfig{MaxFlagsPerEntity: 2})
	addEntity(acc, store, 1, "")

	// A non-PWS flag on the same entity does not count toward the cap.
	acc.components[1].StateFlags["trust.some_flag"] = entity.StateFlagValue{Kind: entity.StateFlagBool, Bool: true}

	for i := 1; i <= 3; i++ {
		flag := fmt.Sprintf("world.flag_%d", i)
		if err := store.SetWorldFlag(1, flag, true); err != nil {
			t.Fatalf("SetWorldFlag(%s) beyond cap returned error: %v", flag, err)
		}
	}
	if flags := store.GetWorldFlags(1); len(flags) != 3 {
		t.Errorf("GetWorldFlags = %v, want all 3 world flags (soft limit)", flags)
	}
}

// TestDefaultMaxFlagsPerEntityAllows32Plus covers the QA AC-5 edge case:
// with the GDD default cap (32), an entity with 33 flags stores all of them
// ("32+ flags within max_flags_per_entity tuning knob").
func TestDefaultMaxFlagsPerEntityAllows32Plus(t *testing.T) {
	store, acc := newTestStore()
	addEntity(acc, store, 1, "")

	for i := 1; i <= 33; i++ {
		if err := store.SetWorldFlag(1, fmt.Sprintf("world.flag_%d", i), i); err != nil {
			t.Fatalf("SetWorldFlag(%d): %v", i, err)
		}
	}
	if flags := store.GetWorldFlags(1); len(flags) != 33 {
		t.Errorf("GetWorldFlags = %d flags, want 33", len(flags))
	}
}
