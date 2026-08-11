// Integration tests for Story SAVE-001: Save/Load Orchestration.
//
// Evidence path: production/epics/save-load-system/story-001-save-load-orchestration.md
// Required evidence: this file must exist and pass.
//
// The acceptance criteria map to test functions 1:1 (Go only executes
// Test-prefixed functions, so each story name is exported with the Go
// prefix):
//
//	AC-1  test_save_100entities_completes_within_2_seconds -> TestSave100EntitiesCompletesWithin2Seconds
//	AC-2  test_load_roundtrip_identical_state              -> TestLoadRoundtripIdenticalState
//	AC-3  test_version_migration_runs_and_loads            -> TestVersionMigrationRunsAndLoads
//	AC-4  test_slot_management_max_slots                   -> TestSlotManagementMaxSlots
//	AC-5  test_quicksave_indicator                         -> TestQuicksaveIndicator
//	AC-6  test_quickload_restores_state                    -> TestQuickloadRestoresState
//
// Edge cases from the QA plan: no quicksave exists -> F9 no-op error (no
// crash); corrupted .sav file -> CRC32 checksum rejection (the .bak restore
// path is SAVE-003, out of scope); save from a newer version rejected; save
// from an older version with no registered migration rejected.
//
// The mock KaijuHost and newTestStore helpers are mirrored from
// tests/integration/entity (Go cannot import test helpers across packages).
package save_load_test

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"kaijuengine.com/engine"
	"kaijuengine.com/matrix"
	"kaijuengine.com/platform/concurrent"

	"marrow-of-empire/entity"
	"marrow-of-empire/save"
)

// mockKaijuHost is a test double for entity.KaijuHost (mirrored from
// tests/integration/entity). It registers engine entities by EntityId and
// records native destroy calls.
type mockKaijuHost struct {
	entities  map[engine.EntityId]*engine.Entity
	destroyed []engine.EntityId
	workGroup concurrent.WorkGroup
}

func (m *mockKaijuHost) EntityById(id engine.EntityId) *engine.Entity {
	return m.entities[id]
}

func (m *mockKaijuHost) SetEntityId(entity *engine.Entity, id engine.EntityId) bool {
	if id == "" {
		return false
	}
	if _, exists := m.entities[id]; exists {
		return false
	}
	m.entities[id] = entity
	return true
}

func (m *mockKaijuHost) DestroyEntity(entity *engine.Entity) {
	for id, e := range m.entities {
		if e == entity {
			delete(m.entities, id)
			m.destroyed = append(m.destroyed, id)
			return
		}
	}
}

func (m *mockKaijuHost) WorkGroup() *concurrent.WorkGroup { return &m.workGroup }

func newMockHost() *mockKaijuHost {
	m := &mockKaijuHost{entities: make(map[engine.EntityId]*engine.Entity)}
	m.workGroup.Init()
	return m
}

// newTestStore wires a store to a mock host and a fresh event bus (mirrored
// from tests/integration/entity). Config zero values fall back to the GDD
// defaults.
func newTestStore() (*entity.EntityStoreImpl, *mockKaijuHost) {
	host := newMockHost()
	store := entity.NewEntityStore(host, entity.NewEventBus(), entity.EntityStoreConfig{})
	return store, host
}

// newTestManager wires a SaveManager to a fresh store and a per-test save
// directory (t.TempDir removes it after the test — no stray save files).
func newTestManager(t *testing.T) (*save.SaveManagerImpl, *entity.EntityStoreImpl) {
	t.Helper()
	store, _ := newTestStore()
	manager := save.NewSaveManager(store, save.Config{SaveDir: t.TempDir()})
	return manager, store
}

func createTestEntity(t *testing.T, store *entity.EntityStoreImpl, etype entity.EntityType, name string, pos matrix.Vec3) entity.GameEntityID {
	t.Helper()
	id, err := store.CreateEntity(etype, name, pos)
	if err != nil {
		t.Fatalf("CreateEntity(%q) error: %v", name, err)
	}
	return id
}

func mustSetComponent[T any](t *testing.T, store *entity.EntityStoreImpl, id entity.GameEntityID, component T) {
	t.Helper()
	if err := entity.SetComponent(store, id, component); err != nil {
		t.Fatalf("SetComponent on entity %d error: %v", id, err)
	}
}

func samePosition(a, b matrix.Vec3) bool {
	return a.Distance(b) == 0
}

// kaijuPosition reads the entity position from the mock host's engine entity.
func kaijuPosition(host *mockKaijuHost, id entity.GameEntityID) (matrix.Vec3, bool) {
	kaiju := host.EntityById(engine.EntityId(fmt.Sprintf("game_%d", id)))
	if kaiju == nil {
		return matrix.Vec3{}, false
	}
	return kaiju.Transform.Position(), true
}

// TestSave100EntitiesCompletesWithin2Seconds is AC-1: saving 100 entities
// with components collects the component stores, pod-encodes them, and writes
// the .sav file within 2 seconds (the ADR-0002 guardrail is 500 entities in
// 2s; the QA case exercises 100).
func TestSave100EntitiesCompletesWithin2Seconds(t *testing.T) {
	manager, store := newTestManager(t)
	for i := 0; i < 100; i++ {
		id := createTestEntity(t, store, entity.EntityTypeNPC, fmt.Sprintf("Guard%d", i), matrix.NewVec3(float32(i)*2, 0, 0))
		mustSetComponent(t, store, id, entity.Health{Current: 100 - i, Max: 100, Temporary: i})
		if i%2 == 0 {
			mustSetComponent(t, store, id, entity.Stats{Strength: 10 + i%9, Dexterity: 11, Charisma: 12})
		}
	}

	start := time.Now()
	if err := manager.Save(0, save.SaveMetadata{CharacterName: "Test Hero"}); err != nil {
		t.Fatalf("Save error: %v", err)
	}
	if elapsed := time.Since(start); elapsed >= 2*time.Second {
		t.Errorf("save of 100 entities took %v, want < 2s", elapsed)
	}

	slots := manager.ListSaves()
	if len(slots) != 1 || slots[0].SlotID != 0 {
		t.Fatalf("ListSaves = %+v, want exactly one save in slot 0", slots)
	}
	if _, err := os.Stat(slots[0].FilePath); err != nil {
		t.Fatalf("save file not written: %v", err)
	}
}

// TestLoadRoundtripIdenticalState is AC-2: loading a .sav file reconstructs
// entities with identical GameEntityIDs, positions, and component values, and
// returns the saved metadata.
func TestLoadRoundtripIdenticalState(t *testing.T) {
	saveDir := t.TempDir()
	storeA, _ := newTestStore()
	managerA := save.NewSaveManager(storeA, save.Config{SaveDir: saveDir})

	const count = 50
	wantHealth := make(map[entity.GameEntityID]entity.Health, count)
	wantStats := make(map[entity.GameEntityID]entity.Stats, count)
	wantPos := make(map[entity.GameEntityID]matrix.Vec3, count)
	for i := 0; i < count; i++ {
		pos := matrix.NewVec3(float32(i)*5, float32(i%7), 0)
		id := createTestEntity(t, storeA, entity.EntityTypeCharacter, fmt.Sprintf("Hero%d", i), pos)
		wantPos[id] = pos
		wantHealth[id] = entity.Health{Current: 50 + i, Max: 120, Temporary: i % 3}
		mustSetComponent(t, storeA, id, wantHealth[id])
		if i%2 == 0 {
			wantStats[id] = entity.Stats{Strength: 8 + i%11, Dexterity: 12, Constitution: 10, Intelligence: 9, Wisdom: 11, Charisma: 14, Evasion: 12}
			mustSetComponent(t, storeA, id, wantStats[id])
		}
	}
	storeA.UpdateSuspension(matrix.NewVec3(0, 0, 0))

	wantMeta := save.SaveMetadata{
		CharacterName: "Marrow Hero",
		PlayTime:      2 * time.Hour,
		RegionID:      "thornwell",
		Thumbnail:     []byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10}, // tiny JPEG header
	}
	if err := managerA.Save(1, wantMeta); err != nil {
		t.Fatalf("Save error: %v", err)
	}

	// Load into a fresh store: the manager wraps the store it restores.
	storeB, hostB := newTestStore()
	managerB := save.NewSaveManager(storeB, save.Config{SaveDir: saveDir})
	meta, err := managerB.Load(1)
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}

	if meta.CharacterName != wantMeta.CharacterName || meta.PlayTime != wantMeta.PlayTime ||
		meta.RegionID != wantMeta.RegionID || !reflect.DeepEqual(meta.Thumbnail, wantMeta.Thumbnail) {
		t.Errorf("metadata mismatch:\n got %+v\nwant %+v", meta, wantMeta)
	}
	if meta.Version != save.DefaultSaveVersion {
		t.Errorf("metadata version = %d, want %d", meta.Version, save.DefaultSaveVersion)
	}
	if meta.Timestamp.IsZero() {
		t.Error("metadata timestamp not set by save/load")
	}

	gotIDs := storeB.QueryEntities(entity.Query{})
	if len(gotIDs) != count {
		t.Fatalf("loaded %d entities, want %d", len(gotIDs), count)
	}
	for _, id := range gotIDs {
		pos, ok := kaijuPosition(hostB, id)
		if !ok || !samePosition(pos, wantPos[id]) {
			t.Errorf("entity %d: position not restored, got %v ok=%v want %v", id, pos, ok, wantPos[id])
		}
		hp, ok := entity.GetComponent[entity.Health](storeB, id)
		if !ok || hp != wantHealth[id] {
			t.Errorf("entity %d: Health mismatch, got %+v want %+v ok=%v", id, hp, wantHealth[id], ok)
		}
		stats, hasStats := wantStats[id]
		gotStats, okStats := entity.GetComponent[entity.Stats](storeB, id)
		if hasStats != okStats || (okStats && gotStats != stats) {
			t.Errorf("entity %d: Stats mismatch, got %+v want %+v ok=%v", id, gotStats, stats, okStats)
		}
	}
}

// TestVersionMigrationRunsAndLoads is AC-3: a save from version N loaded by a
// version N+1 manager with a registered migration runs the migration and the
// data loads correctly.
func TestVersionMigrationRunsAndLoads(t *testing.T) {
	saveDir := t.TempDir()
	storeA, _ := newTestStore()
	managerA := save.NewSaveManager(storeA, save.Config{SaveDir: saveDir, CurrentVersion: 1})
	id := createTestEntity(t, storeA, entity.EntityTypeNPC, "Veteran", matrix.NewVec3(4, 5, 6))
	mustSetComponent(t, storeA, id, entity.Health{Current: 77, Max: 100})
	if err := managerA.Save(2, save.SaveMetadata{CharacterName: "Old Hero"}); err != nil {
		t.Fatalf("Save (v1) error: %v", err)
	}

	storeB, _ := newTestStore()
	managerB := save.NewSaveManager(storeB, save.Config{SaveDir: saveDir, CurrentVersion: 2})
	migrationsRun := 0
	managerB.RegisterMigration(1, 2, func(payload []byte) ([]byte, error) {
		migrationsRun++
		return payload, nil // the v2 format keeps the payload layout; a real migration transforms it
	})

	meta, err := managerB.Load(2)
	if err != nil {
		t.Fatalf("Load (v2) error: %v", err)
	}
	if migrationsRun != 1 {
		t.Errorf("migration ran %d times, want exactly 1", migrationsRun)
	}
	if meta.Version != 2 {
		t.Errorf("loaded metadata version = %d, want 2 (effective post-migration version)", meta.Version)
	}

	// The migrated data loads correctly: the entity and its components survive.
	state, ok := storeB.EntityState(id)
	if !ok || state != entity.EntityStateActive {
		t.Fatalf("entity %d missing or wrong state after migrated load: %v ok=%v", id, state, ok)
	}
	hp, ok := entity.GetComponent[entity.Health](storeB, id)
	if !ok || hp != (entity.Health{Current: 77, Max: 100}) {
		t.Errorf("entity %d: Health mismatch after migrated load, got %+v ok=%v", id, hp, ok)
	}
}

// TestSlotManagementMaxSlots is AC-4: 20 manual slots plus the auto and
// quicksave reserved slots are managed without exceeding the max — every
// valid slot writes, ListSaves reports exactly 22, and out-of-range IDs are
// rejected.
func TestSlotManagementMaxSlots(t *testing.T) {
	manager, _ := newTestManager(t)
	meta := save.SaveMetadata{CharacterName: "Slot Tester"}

	for slotID := 0; slotID < save.ManualSlotCount; slotID++ {
		if err := manager.Save(slotID, meta); err != nil {
			t.Fatalf("Save(%d) error: %v", slotID, err)
		}
	}
	if err := manager.Save(save.AutoSlotID, meta); err != nil {
		t.Fatalf("Save(auto) error: %v", err)
	}
	if err := manager.Quicksave(meta); err != nil {
		t.Fatalf("Quicksave error: %v", err)
	}

	slots := manager.ListSaves()
	if len(slots) != save.ManualSlotCount+2 {
		t.Fatalf("ListSaves = %d slots, want %d (20 manual + 1 auto + 1 quicksave)", len(slots), save.ManualSlotCount+2)
	}
	// Slots are listed in ID order: -2 (quicksave), -1 (auto), 0..19.
	if slots[0].SlotID != save.QuicksaveSlotID || slots[1].SlotID != save.AutoSlotID ||
		slots[2].SlotID != 0 || slots[len(slots)-1].SlotID != save.ManualSlotCount-1 {
		t.Errorf("unexpected slot order/ids: %v", slots)
	}

	// The slot set is never exceeded: out-of-range IDs are rejected.
	if err := manager.Save(save.ManualSlotCount, meta); !errors.Is(err, save.ErrSlotOutOfRange) {
		t.Errorf("Save(%d) error = %v, want ErrSlotOutOfRange", save.ManualSlotCount, err)
	}
	if err := manager.Save(-3, meta); !errors.Is(err, save.ErrSlotOutOfRange) {
		t.Errorf("Save(-3) error = %v, want ErrSlotOutOfRange", err)
	}
	if _, err := manager.Load(-3); !errors.Is(err, save.ErrSlotOutOfRange) {
		t.Errorf("Load(-3) error = %v, want ErrSlotOutOfRange", err)
	}
}

// TestQuicksaveIndicator is AC-5: quicksave completes and emits the
// save-completed signal the UI observes for the "Game Saved" indicator. The
// 1.5-second display timer is a UI concern and is not asserted here.
func TestQuicksaveIndicator(t *testing.T) {
	manager, store := newTestManager(t)
	_ = createTestEntity(t, store, entity.EntityTypeNPC, "Wanderer", matrix.NewVec3(1, 2, 3))

	var signals []int
	unsubscribe := manager.SubscribeSaveCompleted(func(slotID int, metadata save.SaveMetadata) {
		signals = append(signals, slotID)
	})

	if err := manager.Quicksave(save.SaveMetadata{CharacterName: "Quick Hero"}); err != nil {
		t.Fatalf("Quicksave error: %v", err)
	}
	if len(signals) != 1 || signals[0] != save.QuicksaveSlotID {
		t.Fatalf("save-completed signals = %v, want exactly one for the quicksave slot (%d)", signals, save.QuicksaveSlotID)
	}

	// The signal fires for manual saves too, and unsubscribing stops it.
	unsubscribe()
	if err := manager.Save(0, save.SaveMetadata{CharacterName: "Manual"}); err != nil {
		t.Fatalf("Save error: %v", err)
	}
	if len(signals) != 1 {
		t.Errorf("signals after unsubscribe = %v, want only the quicksave signal", signals)
	}
}

// TestQuickloadRestoresState is AC-6: F9 with a valid quicksave restores the
// game state to the quicksave moment — entities, positions, and component
// values — and returns the quicksave's metadata.
func TestQuickloadRestoresState(t *testing.T) {
	manager, store := newTestManager(t)
	id1 := createTestEntity(t, store, entity.EntityTypeNPC, "Saved", matrix.NewVec3(10, 20, 0))
	id2 := createTestEntity(t, store, entity.EntityTypeNPC, "SavedToo", matrix.NewVec3(30, 40, 0))
	mustSetComponent(t, store, id1, entity.Health{Current: 60, Max: 100})
	mustSetComponent(t, store, id2, entity.Health{Current: 90, Max: 100})
	if err := manager.Quicksave(save.SaveMetadata{CharacterName: "Quicksave Hero"}); err != nil {
		t.Fatalf("Quicksave error: %v", err)
	}

	// Mutate the live state after the quicksave moment: destroy an entity,
	// change a component, create a new entity.
	if err := store.DestroyEntity(id1); err != nil {
		t.Fatalf("DestroyEntity error: %v", err)
	}
	mustSetComponent(t, store, id2, entity.Health{Current: 1, Max: 100})
	createTestEntity(t, store, entity.EntityTypeItem, "PostSave", matrix.NewVec3(5, 5, 5))

	meta, err := manager.Quickload()
	if err != nil {
		t.Fatalf("Quickload error: %v", err)
	}
	if meta.CharacterName != "Quicksave Hero" {
		t.Errorf("quickload metadata = %+v, want Quicksave Hero", meta)
	}

	// The game state matches the quicksave moment: id1 is back, both Health
	// values are restored, and the post-save entity is gone.
	got := store.QueryEntities(entity.Query{})
	if len(got) != 2 || got[0] != id1 || got[1] != id2 {
		t.Fatalf("entities after quickload = %v, want [%d %d]", got, id1, id2)
	}
	hp, ok := entity.GetComponent[entity.Health](store, id1)
	if !ok || hp != (entity.Health{Current: 60, Max: 100}) {
		t.Errorf("entity %d: Health mismatch after quickload, got %+v ok=%v", id1, hp, ok)
	}
	hp, ok = entity.GetComponent[entity.Health](store, id2)
	if !ok || hp != (entity.Health{Current: 90, Max: 100}) {
		t.Errorf("entity %d: Health mismatch after quickload, got %+v ok=%v", id2, hp, ok)
	}
}

// TestQuickloadWithoutQuicksaveReturnsError covers the AC-6 edge case: F9
// with no quicksave is a no-op error (ErrSaveNotFound), never a crash. The
// same holds for loading an empty manual slot.
func TestQuickloadWithoutQuicksaveReturnsError(t *testing.T) {
	manager, _ := newTestManager(t) // no saves at all

	meta, err := manager.Quickload()
	if meta != nil {
		t.Errorf("Quickload returned metadata %+v, want nil", meta)
	}
	if !errors.Is(err, save.ErrSaveNotFound) {
		t.Errorf("Quickload error = %v, want ErrSaveNotFound", err)
	}
	if _, err := manager.Load(0); !errors.Is(err, save.ErrSaveNotFound) {
		t.Errorf("Load(0) on empty slot error = %v, want ErrSaveNotFound", err)
	}
}

// TestCorruptedSaveDetectedByChecksum covers the QA edge case: a flipped byte
// in the payload is detected by the CRC32 checksum (ADR-0002). The .bak
// restore path is SAVE-002 and out of scope; the live state must be untouched
// by the rejected load.
func TestCorruptedSaveDetectedByChecksum(t *testing.T) {
	manager, store := newTestManager(t)
	id := createTestEntity(t, store, entity.EntityTypeNPC, "Target", matrix.NewVec3(1, 1, 1))
	if err := manager.Save(3, save.SaveMetadata{CharacterName: "Corrupt Me"}); err != nil {
		t.Fatalf("Save error: %v", err)
	}

	slots := manager.ListSaves()
	if len(slots) != 1 {
		t.Fatalf("ListSaves = %d slots, want 1", len(slots))
	}
	data, err := os.ReadFile(slots[0].FilePath)
	if err != nil {
		t.Fatalf("ReadFile error: %v", err)
	}
	data[16] ^= 0xFF // flip one byte in the payload (after the 16-byte header)
	if err := os.WriteFile(slots[0].FilePath, data, 0o644); err != nil {
		t.Fatalf("WriteFile error: %v", err)
	}

	_, err = manager.Load(3)
	if !errors.Is(err, save.ErrChecksumMismatch) {
		t.Fatalf("Load error = %v, want ErrChecksumMismatch", err)
	}
	if state, ok := store.EntityState(id); !ok || state != entity.EntityStateActive {
		t.Errorf("entity %d state changed by rejected load: %v ok=%v", id, state, ok)
	}
}

// TestNewerVersionSaveRejected covers the forward-compatibility direction:
// a save written by a newer build cannot load on an older build
// (ErrUnsupportedVersion).
func TestNewerVersionSaveRejected(t *testing.T) {
	saveDir := t.TempDir()
	storeA, _ := newTestStore()
	managerA := save.NewSaveManager(storeA, save.Config{SaveDir: saveDir, CurrentVersion: 2})
	_ = createTestEntity(t, storeA, entity.EntityTypeNPC, "Future", matrix.NewVec3(1, 0, 0))
	if err := managerA.Save(0, save.SaveMetadata{CharacterName: "From the Future"}); err != nil {
		t.Fatalf("Save (v2) error: %v", err)
	}

	storeB, _ := newTestStore()
	managerB := save.NewSaveManager(storeB, save.Config{SaveDir: saveDir, CurrentVersion: 1})
	_, err := managerB.Load(0)
	if !errors.Is(err, save.ErrUnsupportedVersion) {
		t.Fatalf("Load error = %v, want ErrUnsupportedVersion (newer save)", err)
	}
}

// TestLoadWithoutRegisteredMigrationRejected covers the missing-chain branch
// of TR-save-002: an older save cannot load on a newer build when no
// migration function is registered for the transition.
func TestLoadWithoutRegisteredMigrationRejected(t *testing.T) {
	saveDir := t.TempDir()
	storeA, _ := newTestStore()
	managerA := save.NewSaveManager(storeA, save.Config{SaveDir: saveDir, CurrentVersion: 1})
	if err := managerA.Save(0, save.SaveMetadata{CharacterName: "Lonely"}); err != nil {
		t.Fatalf("Save error: %v", err)
	}

	storeB, _ := newTestStore()
	managerB := save.NewSaveManager(storeB, save.Config{SaveDir: saveDir, CurrentVersion: 2})
	_, err := managerB.Load(0)
	if !errors.Is(err, save.ErrUnsupportedVersion) {
		t.Fatalf("Load error = %v, want ErrUnsupportedVersion (no migration path)", err)
	}
}

// TestZeroComponentEntityRoundTrip covers the QA edge case "entity with zero
// components": a componentless entity survives a save/load round-trip with its
// ID and active state intact.
func TestZeroComponentEntityRoundTrip(t *testing.T) {
	// Use an explicit shared save dir so two managers see the same file.
	saveDir := t.TempDir()
	storeA, _ := newTestStore()
	mgrA := save.NewSaveManager(storeA, save.Config{SaveDir: saveDir})
	bareID := createTestEntity(t, storeA, entity.EntityTypeProp, "BareProp", matrix.NewVec3(7, 8, 9))
	if err := mgrA.Save(0, save.SaveMetadata{CharacterName: "Bare"}); err != nil {
		t.Fatalf("Save error: %v", err)
	}

	storeB, _ := newTestStore()
	mgrB := save.NewSaveManager(storeB, save.Config{SaveDir: saveDir})
	if _, err := mgrB.Load(0); err != nil {
		t.Fatalf("Load error: %v", err)
	}

	state, ok := storeB.EntityState(bareID)
	if !ok || state != entity.EntityStateActive {
		t.Errorf("zero-component entity %d after load: state=%v ok=%v, want Active", bareID, state, ok)
	}
	// The entity has no components — verify QueryEntities returns it.
	ids := storeB.QueryEntities(entity.Query{})
	found := false
	for _, id := range ids {
		if id == bareID {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("zero-component entity %d missing from QueryEntities after load", bareID)
	}
}

// TestAllComponentTypesRoundTrip covers the QA edge case "maximum component
// count": an entity with all 12 attachable component types round-trips
// correctly.
func TestAllComponentTypesRoundTrip(t *testing.T) {
	saveDir := t.TempDir()
	storeA, _ := newTestStore()
	mgrA := save.NewSaveManager(storeA, save.Config{SaveDir: saveDir})

	id := createTestEntity(t, storeA, entity.EntityTypeNPC, "Maxed", matrix.NewVec3(1, 2, 3))

	mustSetComponent(t, storeA, id, entity.Health{Current: 80, Max: 100, Temporary: 5})
	mustSetComponent(t, storeA, id, entity.Stats{Strength: 12, Dexterity: 14, Constitution: 10, Intelligence: 16, Wisdom: 13, Charisma: 8, MaxHealth: 100, ActionPoints: 6, Initiative: 4, Evasion: 12})
	mustSetComponent(t, storeA, id, entity.Inventory{Items: []entity.GameEntityID{1, 2}, EquipmentSlots: map[string]entity.GameEntityID{"weapon": 1}, Gold: 42})
	mustSetComponent(t, storeA, id, entity.Knowledge{Domains: []string{"history"}, KnownFacts: []string{"f1", "f2"}, TrustThreshold: 2})
	mustSetComponent(t, storeA, id, entity.Faction{FactionID: "thornwell", Standing: 50})
	mustSetComponent(t, storeA, id, entity.Dialogue{Greeting: "Hello", Topics: []string{"war"}, PersonalityProfile: "scholar"})
	mustSetComponent(t, storeA, id, entity.Combatant{Initiative: 10, ActionPoints: 4, MovementPoints: 6, MovementSpeed: 3.0, Actions: []string{"slash"}, StatusEffects: nil, ReactionAvailable: true})
	mustSetComponent(t, storeA, id, entity.Interactable{InteractionType: entity.InteractionTalk, PromptText: "Speak", Range: 2.0, IsOnceOnly: false})
	mustSetComponent(t, storeA, id, entity.Persistent{StateFlags: map[string]entity.StateFlagValue{"world.test": {Kind: entity.StateFlagBool, Bool: true}}, MemoryLog: nil, ModifiedAt: 100.0})
	mustSetComponent(t, storeA, id, entity.Visual{ModelRef: "npc_01", AnimationState: "idle", Tint: matrix.NewVec3(1, 1, 1), VisibilityFlag: 3, RenderLayer: 1})
	mustSetComponent(t, storeA, id, entity.Audio{AmbientSound: "wind", FootstepType: "stone", OcclusionFlag: false})
	mustSetComponent(t, storeA, id, entity.Schedule{Entries: []entity.ScheduleEntry{{StartTime: 480, EndTime: 720, Activity: "working", TargetPosition: matrix.NewVec3(10, 10, 0)}}, CurrentActivity: "working"})

	if err := mgrA.Save(0, save.SaveMetadata{CharacterName: "MaxedOut"}); err != nil {
		t.Fatalf("Save error: %v", err)
	}

	storeB, _ := newTestStore()
	mgrB := save.NewSaveManager(storeB, save.Config{SaveDir: saveDir})
	if _, err := mgrB.Load(0); err != nil {
		t.Fatalf("Load error: %v", err)
	}

	// All 12 components must be present after load.
	if _, ok := entity.GetComponent[entity.Health](storeB, id); !ok {
		t.Error("Health component missing after load")
	}
	if _, ok := entity.GetComponent[entity.Stats](storeB, id); !ok {
		t.Error("Stats component missing after load")
	}
	if _, ok := entity.GetComponent[entity.Inventory](storeB, id); !ok {
		t.Error("Inventory component missing after load")
	}
	if _, ok := entity.GetComponent[entity.Knowledge](storeB, id); !ok {
		t.Error("Knowledge component missing after load")
	}
	if _, ok := entity.GetComponent[entity.Faction](storeB, id); !ok {
		t.Error("Faction component missing after load")
	}
	if _, ok := entity.GetComponent[entity.Dialogue](storeB, id); !ok {
		t.Error("Dialogue component missing after load")
	}
	if _, ok := entity.GetComponent[entity.Combatant](storeB, id); !ok {
		t.Error("Combatant component missing after load")
	}
	if _, ok := entity.GetComponent[entity.Interactable](storeB, id); !ok {
		t.Error("Interactable component missing after load")
	}
	if _, ok := entity.GetComponent[entity.Persistent](storeB, id); !ok {
		t.Error("Persistent component missing after load")
	}
	if _, ok := entity.GetComponent[entity.Visual](storeB, id); !ok {
		t.Error("Visual component missing after load")
	}
	if _, ok := entity.GetComponent[entity.Audio](storeB, id); !ok {
		t.Error("Audio component missing after load")
	}
	if _, ok := entity.GetComponent[entity.Schedule](storeB, id); !ok {
		t.Error("Schedule component missing after load")
	}
}

// TestDeleteSaveRemovesFileAndNoOpOnEmpty verifies DeleteSave removes the .sav
// file from disk and that deleting an empty slot is a no-op (nil error).
func TestDeleteSaveRemovesFileAndNoOpOnEmpty(t *testing.T) {
	manager, store := newTestManager(t)
	createTestEntity(t, store, entity.EntityTypeNPC, "Target", matrix.NewVec3(1, 2, 3))

	if err := manager.Save(3, save.SaveMetadata{CharacterName: "Delete Me"}); err != nil {
		t.Fatalf("Save error: %v", err)
	}
	slots := manager.ListSaves()
	if len(slots) != 1 {
		t.Fatalf("expected 1 save, got %d", len(slots))
	}

	if err := manager.DeleteSave(3); err != nil {
		t.Fatalf("DeleteSave error: %v", err)
	}
	if _, err := os.Stat(slots[0].FilePath); !os.IsNotExist(err) {
		t.Errorf("save file still exists after DeleteSave: %v", err)
	}
	if len(manager.ListSaves()) != 0 {
		t.Error("ListSaves not empty after DeleteSave")
	}

	// Deleting an empty slot is a no-op.
	if err := manager.DeleteSave(5); err != nil {
		t.Errorf("DeleteSave on empty slot error: %v, want nil", err)
	}
	// Out-of-range slot is still rejected.
	if err := manager.DeleteSave(-3); !errors.Is(err, save.ErrSlotOutOfRange) {
		t.Errorf("DeleteSave(-3) error = %v, want ErrSlotOutOfRange", err)
	}
}

// TestTruncatedSaveFileRejected verifies that a file shorter than the 16-byte
// header returns a distinct error (not a panic, not a misleading checksum or
// version error).
func TestTruncatedSaveFileRejected(t *testing.T) {
	saveDir := t.TempDir()
	store, _ := newTestStore()
	manager := save.NewSaveManager(store, save.Config{SaveDir: saveDir})

	path := filepath.Join(saveDir, "save_0.sav")
	if err := os.WriteFile(path, []byte{0x01, 0x00}, 0o644); err != nil {
		t.Fatalf("WriteFile error: %v", err)
	}

	_, err := manager.Load(0)
	if err == nil {
		t.Fatal("expected error for truncated file, got nil")
	}
	if errors.Is(err, save.ErrChecksumMismatch) {
		t.Errorf("truncated file returned ErrChecksumMismatch; want a size-related error")
	}
	if errors.Is(err, save.ErrUnsupportedVersion) {
		t.Errorf("truncated file returned ErrUnsupportedVersion; want a size-related error")
	}
}

// TestMigrationChainRunsAllSteps verifies that a multi-step migration chain
// (v1→v2→v3) runs each step exactly once in order.
func TestMigrationChainRunsAllSteps(t *testing.T) {
	saveDir := t.TempDir()
	storeA, _ := newTestStore()
	mgrA := save.NewSaveManager(storeA, save.Config{SaveDir: saveDir, CurrentVersion: 1})
	id := createTestEntity(t, storeA, entity.EntityTypeNPC, "ChainTest", matrix.NewVec3(1, 1, 1))
	mustSetComponent(t, storeA, id, entity.Health{Current: 50, Max: 100})
	if err := mgrA.Save(0, save.SaveMetadata{CharacterName: "v1 Hero"}); err != nil {
		t.Fatalf("Save (v1) error: %v", err)
	}

	storeB, _ := newTestStore()
	mgrB := save.NewSaveManager(storeB, save.Config{SaveDir: saveDir, CurrentVersion: 3})

	stepOrder := make([]string, 0)
	mgrB.RegisterMigration(1, 2, func(payload []byte) ([]byte, error) {
		stepOrder = append(stepOrder, "1->2")
		return payload, nil
	})
	mgrB.RegisterMigration(2, 3, func(payload []byte) ([]byte, error) {
		stepOrder = append(stepOrder, "2->3")
		return payload, nil
	})

	meta, err := mgrB.Load(0)
	if err != nil {
		t.Fatalf("Load (v3) error: %v", err)
	}
	if meta.Version != 3 {
		t.Errorf("loaded version = %d, want 3", meta.Version)
	}
	if len(stepOrder) != 2 || stepOrder[0] != "1->2" || stepOrder[1] != "2->3" {
		t.Errorf("migration chain order = %v, want [1->2 2->3]", stepOrder)
	}
	hp, ok := entity.GetComponent[entity.Health](storeB, id)
	if !ok || hp != (entity.Health{Current: 50, Max: 100}) {
		t.Errorf("entity %d: Health mismatch after chain migration, got %+v ok=%v", id, hp, ok)
	}
}

// TestCorruptedOldSaveChecksumPrecedesMigration verifies that a corrupted
// OLD-version save is rejected by checksum before any migration function
// touches its payload (ADR-0002).
func TestCorruptedOldSaveChecksumPrecedesMigration(t *testing.T) {
	saveDir := t.TempDir()
	storeA, _ := newTestStore()
	mgrA := save.NewSaveManager(storeA, save.Config{SaveDir: saveDir, CurrentVersion: 1})
	createTestEntity(t, storeA, entity.EntityTypeNPC, "OldCorrupt", matrix.NewVec3(1, 1, 1))
	if err := mgrA.Save(0, save.SaveMetadata{CharacterName: "Old"}); err != nil {
		t.Fatalf("Save (v1) error: %v", err)
	}

	// Corrupt the payload of the v1 save.
	slots := mgrA.ListSaves()
	data, err := os.ReadFile(slots[0].FilePath)
	if err != nil {
		t.Fatalf("ReadFile error: %v", err)
	}
	data[20] ^= 0xFF // flip a byte deep in the payload
	if err := os.WriteFile(slots[0].FilePath, data, 0o644); err != nil {
		t.Fatalf("WriteFile error: %v", err)
	}

	storeB, _ := newTestStore()
	mgrB := save.NewSaveManager(storeB, save.Config{SaveDir: saveDir, CurrentVersion: 2})
	migrationRan := false
	mgrB.RegisterMigration(1, 2, func(payload []byte) ([]byte, error) {
		migrationRan = true
		return payload, nil
	})

	_, err = mgrB.Load(0)
	if !errors.Is(err, save.ErrChecksumMismatch) {
		t.Fatalf("Load error = %v, want ErrChecksumMismatch (checksum must precede migration)", err)
	}
	if migrationRan {
		t.Error("migration ran on a corrupted save — checksum check must precede migration")
	}
}

// TestListSavesIgnoresForeignFiles verifies that non-.sav files in the save
// directory are ignored by ListSaves.
func TestListSavesIgnoresForeignFiles(t *testing.T) {
	manager, store := newTestManager(t)
	createTestEntity(t, store, entity.EntityTypeNPC, "Real", matrix.NewVec3(1, 2, 3))
	if err := manager.Save(0, save.SaveMetadata{CharacterName: "Real Save"}); err != nil {
		t.Fatalf("Save error: %v", err)
	}

	// Write foreign files into the save directory.
	saveDir := manager.ListSaves()[0].FilePath[:len(manager.ListSaves()[0].FilePath)-len("save_0.sav")]
	foreignFiles := []string{"readme.txt", ".hidden", "save_99.bak", "save_abc.sav", "config.json"}
	for _, name := range foreignFiles {
		if err := os.WriteFile(filepath.Join(saveDir, name), []byte("junk"), 0o644); err != nil {
			t.Fatalf("WriteFile(%s) error: %v", name, err)
		}
	}

	slots := manager.ListSaves()
	if len(slots) != 1 {
		t.Errorf("ListSaves = %d slots (with foreign files), want exactly 1", len(slots))
	}
	if len(slots) > 0 && slots[0].SlotID != 0 {
		t.Errorf("unexpected slot ID %d in ListSaves, want 0", slots[0].SlotID)
	}
}

// TestNoNotificationOnFailedSave verifies that save-completed subscribers are
// NOT notified when the save fails (e.g. invalid slot).
func TestNoNotificationOnFailedSave(t *testing.T) {
	manager, store := newTestManager(t)
	createTestEntity(t, store, entity.EntityTypeNPC, "Notify", matrix.NewVec3(1, 2, 3))

	var signals []int
	manager.SubscribeSaveCompleted(func(slotID int, metadata save.SaveMetadata) {
		signals = append(signals, slotID)
	})

	// A successful save fires the notification.
	if err := manager.Save(0, save.SaveMetadata{CharacterName: "Good"}); err != nil {
		t.Fatalf("Save error: %v", err)
	}
	if len(signals) != 1 {
		t.Fatalf("expected 1 notification after successful save, got %d", len(signals))
	}

	// A failed save (invalid slot) must NOT fire the notification.
	if err := manager.Save(-3, save.SaveMetadata{CharacterName: "Bad"}); !errors.Is(err, save.ErrSlotOutOfRange) {
		t.Fatalf("Save(-3) error = %v, want ErrSlotOutOfRange", err)
	}
	if len(signals) != 1 {
		t.Errorf("notification fired on failed save: got %d signals, want exactly 1", len(signals))
	}
}
