// Serialization for Story EF-005 (TR-ef-006, ADR-0001, ADR-0002, GDD CR-8).
//
// The Entity Framework's serialization boundary: EntityStoreImpl.Serialize
// and Deserialize encode and restore the complete game-layer state — the
// entity table and every typed component store — through Kaiju's pod
// reflection system. Component types are registered with pod at init time
// (components.go); each typed store implements EntityData
// (component_store.go); this file orchestrates the pass.
//
// Save format (single pod blob, GDD CR-8):
//
//	snapshot { NextEntityID, Table []entityRecord, Stores map[string][]byte }
//
// The entity table is serialized first, then each non-empty component store
// in registration order. Empty stores are omitted entirely and contribute
// zero bytes: pod cannot round-trip an empty top-level map (the encoder
// writes only the empty header tables, and decoding them EOFs), so skipping
// them is both the AC-3 requirement and a correctness requirement.
package entity

import (
	"bytes"
	"fmt"

	"kaijuengine.com/engine"
	"kaijuengine.com/engine/encoding/pod"
	"kaijuengine.com/matrix"
)

// entityRecord is one serialized entity table entry (GDD CR-8 save set):
// identity, type, display name, the authoritative transform position, and
// the lifecycle state. Only Active and Suspended entities are serialized
// (GDD CR-8: Pending and Destroyed are excluded).
type entityRecord struct {
	// ID is the GameEntityID, stable across save/load and never reused.
	ID GameEntityID
	// Type is the entity category.
	Type EntityType
	// DisplayName is the human-readable name supplied at creation.
	DisplayName string
	// Position is the world position read from the Kaiju Transform (the
	// authoritative position source, ADR-0001).
	Position matrix.Vec3
	// State is EntityStateActive or EntityStateSuspended.
	State EntityState
}

// snapshot is the pod-encodable full state of the entity framework.
type snapshot struct {
	// NextEntityID is the monotonic GameEntityID counter at save time (GDD
	// CR-8: restored on load so post-load entity creation never collides
	// with restored IDs; also covers entities destroyed before the save,
	// whose IDs must still never be reused — GDD CR-1).
	NextEntityID GameEntityID
	// Table holds one entityRecord per live entity in registration order.
	Table []entityRecord
	// Stores holds the pod-encoded contents of every non-empty component
	// store, keyed by component type name. Empty stores are omitted (AC-3:
	// zero bytes).
	Stores map[string][]byte
}

func init() {
	// Framework types reachable from the snapshot must be registered with
	// pod or encoding fails ("never registered with pod").
	registerPodType(GameEntityID(0))
	registerPodType(EntityType(0))
	registerPodType(EntityState(0))
	registerPodType(entityRecord{})
	registerPodType(snapshot{})
}

// Serialize implements EntityStore. See the interface doc for the contract.
func (s *EntityStoreImpl) Serialize() ([]byte, error) {
	snap := snapshot{
		NextEntityID: s.entityCounter,
		Table:        make([]entityRecord, 0, len(s.order)),
		Stores:       make(map[string][]byte),
	}
	for _, id := range s.order {
		switch s.state[id] {
		case EntityStateActive, EntityStateSuspended:
		default:
			// Pending and Destroyed entities are never serialized (GDD
			// CR-8). Pending is not externally observable (CreateEntity
			// transitions to Active atomically); the filter is defensive.
			continue
		}
		snap.Table = append(snap.Table, entityRecord{
			ID:          id,
			Type:        s.entityTypes[id],
			DisplayName: s.displayNames[id],
			Position:    s.kaijuPosition(id),
			State:       s.state[id],
		})
	}
	for _, name := range s.storeOrder {
		st := s.storesByName[name]
		if st.entityCount() == 0 {
			// AC-3: an empty store encodes as zero bytes. Omitting it is
			// also required for round-trip correctness: pod cannot encode
			// an empty top-level map.
			continue
		}
		blob, err := st.Encode()
		if err != nil {
			return nil, err
		}
		snap.Stores[name] = blob
	}

	if len(snap.Table) == 0 && len(snap.Stores) == 0 && snap.NextEntityID == 0 {
		// AC-3: a fresh store with no entities and no component data
		// encodes as zero bytes, no error. (A store with history but no
		// live entities still encodes the snapshot so the GameEntityID
		// counter survives the round-trip, GDD CR-1.)
		return []byte{}, nil
	}

	var buf bytes.Buffer
	if err := pod.NewEncoder(&buf).Encode(snap); err != nil {
		return nil, fmt.Errorf("entity: failed to encode snapshot: %w", err)
	}
	return buf.Bytes(), nil
}

// Deserialize implements EntityStore. See the interface doc for the
// contract.
func (s *EntityStoreImpl) Deserialize(data []byte) error {
	if len(data) == 0 {
		// An empty blob is a valid empty snapshot (AC-3).
		return nil
	}
	var snap snapshot
	if err := pod.NewDecoder(bytes.NewReader(data)).Decode(&snap); err != nil {
		return fmt.Errorf("entity: failed to decode snapshot: %w", err)
	}

	// Validate before touching live state: unknown store names and
	// non-serializable states indicate a corrupt or foreign blob.
	reuse := make(map[GameEntityID]bool, len(snap.Table))
	for _, rec := range snap.Table {
		switch rec.State {
		case EntityStateActive, EntityStateSuspended:
		default:
			return fmt.Errorf("entity: snapshot contains entity %d with non-serializable state %s", rec.ID, rec.State)
		}
		reuse[rec.ID] = true
	}
	for name := range snap.Stores {
		if _, ok := storeFactories[name]; !ok {
			return fmt.Errorf("entity: snapshot contains unknown component type %q", name)
		}
	}

	// 1. Clear current state (no lifecycle events, GDD CR-8).
	s.clearAll(reuse)

	// 2. Restore the entity table: recreate Kaiju entities in registration
	// order (story EF-005: restore entity table -> recreate entities).
	s.entityCounter = snap.NextEntityID
	for _, rec := range snap.Table {
		if err := s.restoreEntity(rec); err != nil {
			return err
		}
	}

	// 3. Distribute component data to stores.
	for name, blob := range snap.Stores {
		st := storeFactories[name](s)
		if err := st.Decode(blob); err != nil {
			return err
		}
		st.register()
	}
	return nil
}

// clearAll resets the store to its initial empty state without emitting
// lifecycle events (GDD CR-8: lifecycle events are suppressed during
// load/teardown). Native entities whose IDs the incoming snapshot reuses are
// kept — Deserialize reuses them (restoreEntity), which avoids SetEntityId
// collisions with still-registered entities. All other native entities are
// destroyed; engine.Host.DestroyEntity defers cleanup (ForceCleanup and the
// OnDestroy bridge hook) to the next frame's EndUpdate, and the hook is a
// no-op for them: their IDs are absent from the restored state, so
// handleKaijuDestroy's state lookup misses.
func (s *EntityStoreImpl) clearAll(reuse map[GameEntityID]bool) {
	for _, id := range s.order {
		if reuse[id] {
			continue
		}
		if kaiju := s.host.EntityById(s.idToEntityMap[id]); kaiju != nil {
			s.host.DestroyEntity(kaiju)
		}
	}
	s.order = nil
	s.idToEntityMap = make(map[GameEntityID]engine.EntityId)
	s.state = make(map[GameEntityID]EntityState)
	s.entityTypes = make(map[GameEntityID]EntityType)
	s.displayNames = make(map[GameEntityID]string)
	s.stores = make(map[any]any)
	s.storesByName = make(map[string]componentStore)
	s.storeOrder = nil
	s.index = newSpatialIndex(s.config.SpatialBucketSize, s.config.SpatialOverflowThreshold)
	s.bridgeSuppress = make(map[GameEntityID]bool)
	s.lastEvent = make(map[GameEntityID]EventType)
	s.entityCounter = 0
}

// restoreEntity recreates one entity from the serialized table: it reuses
// the live native entity when the host still registers the ID (the entity
// survived the previous life of this store) and creates a fresh one
// otherwise, then restores identity, transform, spatial index entry, and
// lifecycle state. The native activation state is synced to the restored
// state with the bridge suppressed, so no lifecycle events fire and no
// double transitions occur (GDD CR-8: events suppressed during load).
func (s *EntityStoreImpl) restoreEntity(rec entityRecord) error {
	entityID := engine.EntityId(fmt.Sprintf("game_%d", rec.ID))
	kaiju := s.host.EntityById(entityID)
	switch {
	case kaiju != nil && !kaiju.IsDestroyed():
		// Reuse: the native entity is still registered and live. Its
		// lifecycle bridge closures capture the same GameEntityID, so no
		// re-attachment is needed.
	case kaiju != nil && kaiju.IsDestroyed():
		// The native entity was destroyed but host.DestroyEntity defers
		// cleanup (ForceCleanup + OnDestroy) to the next frame's EndUpdate,
		// and its EntityId remains registered until then. Rebuilding it now
		// would collide on SetEntityId; the caller must run Deserialize
		// after the host flush (frame boundary).
		return fmt.Errorf("entity: engine entity %q is pending destruction; deserialize must run after the host's frame-end cleanup", entityID)
	default:
		kaiju = engine.NewEntity(s.host.WorkGroup())
		if !s.host.SetEntityId(kaiju, entityID) {
			s.host.DestroyEntity(kaiju)
			return fmt.Errorf("entity: failed to register engine entity id %q during deserialization", entityID)
		}
		s.attachBridge(kaiju, rec.ID)
	}
	kaiju.SetName(rec.DisplayName)
	kaiju.Transform.SetPosition(rec.Position)
	s.index.insert(rec.ID, rec.Position)
	s.order = append(s.order, rec.ID)
	s.idToEntityMap[rec.ID] = entityID
	s.state[rec.ID] = rec.State
	s.entityTypes[rec.ID] = rec.Type
	s.displayNames[rec.ID] = rec.DisplayName

	switch rec.State {
	case EntityStateSuspended:
		s.bridgeSuppress[rec.ID] = true
		kaiju.Deactivate()
		delete(s.bridgeSuppress, rec.ID)
	case EntityStateActive:
		s.bridgeSuppress[rec.ID] = true
		kaiju.Activate()
		delete(s.bridgeSuppress, rec.ID)
	}
	return nil
}
