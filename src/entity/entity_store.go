package entity

import (
	"fmt"
	"log/slog"
	"math"
	"sort"

	"kaijuengine.com/engine"
	"kaijuengine.com/matrix"
	"kaijuengine.com/platform/concurrent"
)

// Default tuning values from the Entity Framework GDD tuning knob table.
// Story EF-001 injects them via EntityStoreConfig; a later story wires the
// config to external config files (gameplay values stay data-driven).
const (
	// DefaultSuspensionDistance is the GDD `suspension_distance` (100 units).
	DefaultSuspensionDistance float64 = 100
	// DefaultMaxEntitiesPerRegion is the GDD `max_entities_per_region` (500).
	DefaultMaxEntitiesPerRegion uint64 = 500
	// DefaultSpatialBucketSize is the GDD `spatial_bucket_size` (32 world
	// units): the cell edge length of the uniform grid spatial index.
	DefaultSpatialBucketSize float64 = 32
	// DefaultSpatialOverflowThreshold is the bucket population above which a
	// range query degrades to a linear scan with a warning (Story EF-003
	// AC-5: >500 entities in one bucket).
	DefaultSpatialOverflowThreshold int = 500
)

// Query defines a component + spatial query (ADR-0001).
//
// Tags are accepted for interface compatibility with ADR-0001 but are not
// filtered yet: tag storage and filtering arrive with the typed component
// stores (Story EF-002). Components are all required; tags would be any-match.
type Query struct {
	// Components lists required component type names; all must be present.
	Components []string
	// Tags lists required tags; any match qualifies. Not yet filtered (EF-002).
	Tags []string
	// Position is the center of the spatial query.
	Position matrix.Vec3
	// Range is the search radius in world units; 0 disables the range filter.
	Range float64
}

// EntityStore is the primary interface exposed by the Entity Framework
// (ADR-0001). Game code depends on this interface, never on Kaiju internals.
type EntityStore interface {
	// CreateEntity assembles a game entity with the given type, display name
	// and position: it creates the Kaiju engine entity with EntityId
	// "game_{GameEntityID}", transitions Pending -> Active, and emits
	// entity.created. Returns the new GameEntityID (> 0, never reused).
	CreateEntity(entityType EntityType, displayName string, position matrix.Vec3) (GameEntityID, error)

	// DestroyEntity removes an entity: it calls the Kaiju native destroy,
	// removes the GameEntityID -> EntityId mapping, transitions to Destroyed,
	// and emits entity.destroyed. The GameEntityID counter never decrements
	// and the ID is never reused. Destroying an already-destroyed or unknown
	// ID returns an error.
	DestroyEntity(id GameEntityID) error

	// GetComponentData retrieves a component by type name. Returns
	// (nil, false) for destroyed or unknown entities, for component types
	// never written, and for missing components. This is the type-erased
	// interface method behind the generic GetComponent helper: Go interfaces
	// cannot declare generic methods, so ADR-0001's GetComponent[T]
	// signature is provided as a package-level function (see GetComponent,
	// SetComponent, and RemoveComponent).
	GetComponentData(id GameEntityID, componentType string) (any, bool)

	// SetComponentData stores a component by type name. Components are
	// preserved across suspension/reactivation. The type must already have a
	// typed store (any entity must have been written via the generic
	// SetComponent helper first); otherwise an error is returned. Returns an
	// error if the value is not the store's component type or the entity is
	// destroyed or unknown.
	SetComponentData(id GameEntityID, componentType string, component any) error

	// RemoveComponentData removes a component by type name. Removing an
	// absent component is a no-op (nil, no event); removing from a destroyed
	// or unknown entity returns an error. The generic RemoveComponent helper
	// is the preferred type-safe path.
	RemoveComponentData(id GameEntityID, componentType string) error

	// QueryEntities returns GameEntityIDs matching the query, in registration
	// order. Destroyed entities are never returned; suspended entities are
	// visible to spatial queries.
	QueryEntities(q Query) []GameEntityID

	// UpdatePosition syncs the spatial index after an entity moves (Story
	// EF-003). The Kaiju Transform remains the authoritative position source:
	// systems that move an entity's transform must call UpdatePosition with
	// the new position so range queries index the entity in the correct
	// bucket. Range queries re-verify exact distance against the transform,
	// so a stale bucket can only omit an entity from results — never return
	// one that is out of range. Returns an error for unknown or destroyed
	// IDs.
	//
	// Example usage:
	//
	//	kaiju.Transform.SetPosition(newPos)
	//	store.UpdatePosition(id, newPos)
	UpdatePosition(id GameEntityID, newPosition matrix.Vec3) error

	// EntityState returns the current lifecycle state of an entity.
	// Returns (state, false) for unknown IDs.
	EntityState(id GameEntityID) (EntityState, bool)
}

// KaijuHost abstracts the subset of the Kaiju engine Host API required by the
// entity framework. *engine.Host satisfies this interface; unit tests use a
// mock (ADR-0001: the mapping layer is the only code that touches the engine,
// so the engine can be swapped by rewriting this boundary).
type KaijuHost interface {
	// EntityById returns the engine entity registered with the given ID.
	EntityById(id engine.EntityId) *engine.Entity
	// SetEntityId registers an engine entity for host-wide lookup.
	SetEntityId(entity *engine.Entity, id engine.EntityId) bool
	// DestroyEntity marks an engine entity for destruction (cleanup at the
	// beginning of the next frame).
	DestroyEntity(entity *engine.Entity)
	// WorkGroup returns the host's concurrent work group, used to initialize
	// engine entities.
	WorkGroup() *concurrent.WorkGroup
}

// EntityStoreConfig carries the tuning knobs for the entity framework.
// Zero-valued fields fall back to the GDD defaults (suspension_distance = 100,
// max_entities_per_region = 500, spatial_bucket_size = 32,
// spatial_overflow_threshold = 500).
type EntityStoreConfig struct {
	// SuspensionDistance is the distance beyond which an Active entity is
	// suspended and within which a Suspended entity is reactivated.
	SuspensionDistance float64
	// MaxEntitiesPerRegion warns once entity count exceeds it; entities are
	// still created (QA: warning logged, entity still created).
	MaxEntitiesPerRegion uint64
	// SpatialBucketSize is the cell edge length of the uniform grid spatial
	// index in world units (GDD spatial_bucket_size, default 32).
	SpatialBucketSize float64
	// SpatialOverflowThreshold is the bucket population above which a range
	// query degrades to a linear scan with a warning (Story EF-003 AC-5,
	// default 500).
	SpatialOverflowThreshold int
}

// EntityStoreImpl is the concrete EntityStore (ADR-0001).
//
// GameEntityID (uint64, monotonic counter) maps to Kaiju EntityId (string,
// "game_{ID}") through idToEntityMap. The lifecycle state machine
// (Pending -> Active -> Suspended -> Destroyed) is stored in state, and every
// transition is broadcast synchronously through the EventBus. Kaiju lifecycle
// hooks (OnActivate/OnDeactivate/OnDestroy) are bridged to game-layer events
// (TR-ef-004); the store suppresses its own hook-triggered emissions while it
// drives a transition, so each transition emits exactly once.
type EntityStoreImpl struct {
	host   KaijuHost
	events EventBus
	config EntityStoreConfig

	// entityCounter is the monotonic GameEntityID source. It never
	// decrements and destroyed IDs are never reused.
	entityCounter GameEntityID

	// order is the registration-ordered list of live entity IDs. Iteration
	// over maps is non-deterministic, so all traversal (suspension checks,
	// queries) uses this slice (GDD: ordered registration list).
	order []GameEntityID

	idToEntityMap map[GameEntityID]engine.EntityId
	state         map[GameEntityID]EntityState
	entityTypes   map[GameEntityID]EntityType
	displayNames  map[GameEntityID]string
	// stores holds one typedStore[T] per component type, keyed by the type
	// descriptor pointer (*T)(nil) — a compile-time value unique per type —
	// so typed access (GetComponent[T]) never uses reflection. storesByName
	// mirrors the same stores keyed by type name for the string-keyed
	// interface methods and QueryEntities. Both are populated lazily: a
	// store is allocated on the first write of its component type
	// (TR-ef-002, GDD CR-2).
	stores       map[any]any
	storesByName map[string]componentStore

	// index is the uniform grid spatial index (Story EF-003, TR-ef-003).
	// CreateEntity inserts, retire removes, and UpdatePosition migrates
	// entities between buckets; QueryEntities uses it for Range > 0 queries.
	index *spatialIndex

	// bridgeSuppress prevents the Kaiju lifecycle hooks from emitting when
	// the store itself drives the transition (avoids double emission).
	bridgeSuppress map[GameEntityID]bool
	// lastEvent is the audit record written by the store's priority-1
	// subscription (ADR-0011: Entity Framework processes lifecycle first).
	lastEvent map[GameEntityID]EventType
}

// NewEntityStore creates the entity store wired to a Kaiju host and an
// EventBus (dependency injection — no singletons). Config zero values fall
// back to GDD defaults.
//
// Example usage:
//
//	bus := entity.NewEventBus()
//	store := entity.NewEntityStore(host, bus, entity.EntityStoreConfig{})
func NewEntityStore(host KaijuHost, eventBus EventBus, config EntityStoreConfig) *EntityStoreImpl {
	if config.SuspensionDistance <= 0 {
		config.SuspensionDistance = DefaultSuspensionDistance
	}
	if config.MaxEntitiesPerRegion == 0 {
		config.MaxEntitiesPerRegion = DefaultMaxEntitiesPerRegion
	}
	if config.SpatialBucketSize <= 0 {
		config.SpatialBucketSize = DefaultSpatialBucketSize
	}
	if config.SpatialOverflowThreshold <= 0 {
		config.SpatialOverflowThreshold = DefaultSpatialOverflowThreshold
	}
	s := &EntityStoreImpl{
		host:           host,
		events:         eventBus,
		config:         config,
		idToEntityMap:  make(map[GameEntityID]engine.EntityId),
		state:          make(map[GameEntityID]EntityState),
		entityTypes:    make(map[GameEntityID]EntityType),
		displayNames:   make(map[GameEntityID]string),
		stores:         make(map[any]any),
		storesByName:   make(map[string]componentStore),
		index:          newSpatialIndex(config.SpatialBucketSize, config.SpatialOverflowThreshold),
		bridgeSuppress: make(map[GameEntityID]bool),
		lastEvent:      make(map[GameEntityID]EventType),
	}
	s.subscribeSelf()
	return s
}

// CreateEntity implements EntityStore. The engine entity starts active in
// Kaiju terms (engine.Entity.Init sets isActive), so the Pending -> Active
// transition and the entity.created emission are store-driven; the
// OnActivate bridge handles later external activations (reactivation path).
func (s *EntityStoreImpl) CreateEntity(entityType EntityType, displayName string, position matrix.Vec3) (GameEntityID, error) {
	s.entityCounter++
	id := s.entityCounter
	entityID := engine.EntityId(fmt.Sprintf("game_%d", id))

	if uint64(s.entityCounter) > s.config.MaxEntitiesPerRegion {
		slog.Warn("max entities per region exceeded; entity still created",
			"max", s.config.MaxEntitiesPerRegion, "id", id)
	}

	kaiju := engine.NewEntity(s.host.WorkGroup())
	if !s.host.SetEntityId(kaiju, entityID) {
		s.entityCounter--
		s.host.DestroyEntity(kaiju)
		return 0, fmt.Errorf("entity: failed to register engine entity id %q", entityID)
	}
	kaiju.SetName(displayName)
	kaiju.Transform.SetPosition(position)
	// Initial placement in the spatial index (Story EF-003). The Kaiju
	// transform is the authoritative position; the index mirrors it.
	s.index.insert(id, position)

	s.order = append(s.order, id)
	s.idToEntityMap[id] = entityID
	s.state[id] = EntityStatePending
	s.entityTypes[id] = entityType
	s.displayNames[id] = displayName

	s.attachBridge(kaiju, id)

	s.state[id] = EntityStateActive
	s.emitLifecycle(EventEntityCreated, id)
	return id, nil
}

// DestroyEntity implements EntityStore. Order per the lifecycle bridge
// contract: (1) snapshot position before Kaiju destroy unregisters the
// EntityId, (2) Kaiju native destroy, (3) remove the GameEntityID -> EntityId
// mapping, (4) transition to Destroyed, (5) emit entity.destroyed. The
// counter never decrements and the ID is never reused.
func (s *EntityStoreImpl) DestroyEntity(id GameEntityID) error {
	state, ok := s.state[id]
	if !ok {
		return fmt.Errorf("entity: no entity with id %d", id)
	}
	if state == EntityStateDestroyed {
		return fmt.Errorf("entity: id %d already destroyed", id)
	}

	// Snapshot before host.DestroyEntity unregisters the Kaiju EntityId
	// so the entity.destroyed event carries the correct position.
	payload := s.buildPayload(id)

	s.bridgeSuppress[id] = true
	if kaiju := s.host.EntityById(s.idToEntityMap[id]); kaiju != nil {
		s.host.DestroyEntity(kaiju)
	}
	delete(s.bridgeSuppress, id)

	s.retire(id, payload)
	return nil
}

// retire removes the entity's mapping and data, marks it Destroyed, and emits
// entity.destroyed. The payload must be built before host.DestroyEntity
// unregisters the Kaiju EntityId (so the event carries the correct position).
// Subscribers dispatch synchronously, so all subscribed systems receive the
// event before DestroyEntity returns.
func (s *EntityStoreImpl) retire(id GameEntityID, payload EntityEventData) {
	delete(s.idToEntityMap, id)
	// Purge the entity's data from every typed component store. No
	// component.removed events fire during destruction — entity.destroyed is
	// the single destruction notification (Story EF-002: component events
	// cover explicit removal only).
	for _, st := range s.storesByName {
		st.removeEntity(id)
	}
	delete(s.entityTypes, id)
	delete(s.displayNames, id)
	delete(s.lastEvent, id)
	// Drop the entity from the spatial index (Story EF-003). Idempotent:
	// the Kaiju OnDestroy bridge path reaches retire exactly once.
	s.index.remove(id)
	s.state[id] = EntityStateDestroyed
	s.removeFromOrder(id)
	s.events.Emit(Event{
		Type:   EventEntityDestroyed,
		Source: "entity_framework",
		Data:   payload,
	})
}

// UpdateSuspension runs the distance-based suspension check for all live
// entities, in registration order (deterministic): Active entities beyond
// suspension_distance are suspended; Suspended entities within it are
// reactivated. Component data is preserved across both transitions.
//
// The method iterates a snapshot of the registration order so that a
// subscriber destroying an entity during synchronous event dispatch does
// not mutate the ranged slice and skip entities.
func (s *EntityStoreImpl) UpdateSuspension(player matrix.Vec3) {
	orderSnapshot := make([]GameEntityID, len(s.order))
	copy(orderSnapshot, s.order)
	for _, id := range orderSnapshot {
		switch s.state[id] {
		case EntityStateActive:
			if s.distanceTo(id, player) > s.config.SuspensionDistance {
				s.bridgeSuppress[id] = true
				if kaiju := s.host.EntityById(s.idToEntityMap[id]); kaiju != nil {
					kaiju.Deactivate()
				}
				delete(s.bridgeSuppress, id)
				s.state[id] = EntityStateSuspended
				s.emitLifecycle(EventEntitySuspended, id)
			}
		case EntityStateSuspended:
			if s.distanceTo(id, player) <= s.config.SuspensionDistance {
				s.bridgeSuppress[id] = true
				if kaiju := s.host.EntityById(s.idToEntityMap[id]); kaiju != nil {
					kaiju.Activate()
				}
				delete(s.bridgeSuppress, id)
				s.state[id] = EntityStateActive
				s.emitLifecycle(EventEntityReactivated, id)
			}
		}
	}
}

// GetComponentData implements EntityStore. This is the type-erased path used
// by the generic GetComponent helper as a fallback for non-EntityStoreImpl
// stores; on the concrete store the helper takes the typed path (no boxing,
// no reflection). Returns (nil, false) for destroyed or unknown entities,
// for component types never written, and for missing components.
func (s *EntityStoreImpl) GetComponentData(id GameEntityID, componentType string) (any, bool) {
	st, ok := s.storesByName[componentType]
	if !ok {
		return nil, false
	}
	return st.get(id)
}

// SetComponentData implements EntityStore. The erased path cannot create a
// typed store without knowing the component's Go type, so a store must
// already exist for the type (any entity must have been written via the
// generic SetComponent helper first); otherwise an error is returned.
// Returns an error if the value is not the store's component type or the
// entity is destroyed or unknown.
func (s *EntityStoreImpl) SetComponentData(id GameEntityID, componentType string, component any) error {
	if !s.isLive(id) {
		return fmt.Errorf("entity: no live entity with id %d", id)
	}
	st, ok := s.storesByName[componentType]
	if !ok {
		return fmt.Errorf("entity: no component store for type %q (write a component of that type via SetComponent[T] first)", componentType)
	}
	if !st.setErased(id, component) {
		return fmt.Errorf("entity: value of type %T cannot be stored as a %s component", component, componentType)
	}
	return nil
}

// RemoveComponentData implements EntityStore. Removing a component that does
// not exist is a no-op (nil, no event); removing from a destroyed or unknown
// entity returns an error.
func (s *EntityStoreImpl) RemoveComponentData(id GameEntityID, componentType string) error {
	if !s.isLive(id) {
		return fmt.Errorf("entity: no live entity with id %d", id)
	}
	if st, ok := s.storesByName[componentType]; ok {
		st.take(id)
	}
	return nil
}

// QueryEntities implements EntityStore. For Range > 0 the uniform grid
// spatial index narrows the candidate set to the buckets overlapping the
// query circle (Story EF-003); candidates are then filtered by required
// component names and exact distance from Query.Position, read from the
// authoritative Kaiju Transform. When a scanned bucket exceeds the spatial
// overflow threshold, the query degrades to the linear scan with a warning
// (AC-5). Range == 0 disables the range filter and always uses the linear
// scan. Results are returned in registration order. Destroyed entities are
// never returned; suspended entities remain visible to spatial queries.
func (s *EntityStoreImpl) QueryEntities(q Query) []GameEntityID {
	if q.Range > 0 {
		candidates, degraded := s.index.candidates(q.Position, q.Range)
		if !degraded {
			return s.filterSpatialCandidates(candidates, q)
		}
		s.index.recordDegradation()
		slog.Warn("spatial index: bucket overflow, range query degraded to linear scan",
			"bucket_size", s.index.bucketSize,
			"overflow_threshold", s.index.overflowThreshold)
	}

	// Linear scan: the Range == 0 path (no spatial filter) and the AC-5
	// degradation fallback. Iterates the registration-ordered list (never a
	// map) so results are deterministic.
	result := make([]GameEntityID, 0, len(s.order))
	for _, id := range s.order {
		if !s.matchesComponents(id, q.Components) {
			continue
		}
		if q.Range > 0 && s.distanceTo(id, q.Position) > q.Range {
			continue
		}
		result = append(result, id)
	}
	return result
}

// filterSpatialCandidates applies the component and exact-distance filters to
// the spatial index candidates and returns the matches in registration order.
// The store's registration order is strictly ascending by GameEntityID (IDs
// come from a monotonic counter and are never reused), so sorting the
// candidates ascending reproduces it without a lookup map.
func (s *EntityStoreImpl) filterSpatialCandidates(candidates []GameEntityID, q Query) []GameEntityID {
	result := make([]GameEntityID, 0, len(candidates))
	for _, id := range candidates {
		if !s.matchesComponents(id, q.Components) {
			continue
		}
		if s.distanceTo(id, q.Position) > q.Range {
			continue
		}
		result = append(result, id)
	}
	sort.Sort(idSlice(result))
	return result
}

// idSlice sorts GameEntityIDs ascending, which equals the store's registration
// order: the order slice is append-only ascending and never reuses IDs.
type idSlice []GameEntityID

func (s idSlice) Len() int           { return len(s) }
func (s idSlice) Less(i, j int) bool { return s[i] < s[j] }
func (s idSlice) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }

// UpdatePosition implements EntityStore. It updates only the spatial index;
// the Kaiju Transform remains authoritative and is never mutated here —
// systems that move an entity update the transform themselves and pass the
// same position to this method. Unknown and destroyed IDs are rejected.
func (s *EntityStoreImpl) UpdatePosition(id GameEntityID, newPosition matrix.Vec3) error {
	if !s.isLive(id) {
		return fmt.Errorf("entity: no live entity with id %d", id)
	}
	s.index.update(id, newPosition)
	return nil
}

// SpatialIndexDegradationCount returns how many range queries degraded to a
// linear scan because a scanned bucket exceeded the spatial overflow
// threshold (Story EF-003 AC-5 evidence; each degradation is also logged with
// slog).
func (s *EntityStoreImpl) SpatialIndexDegradationCount() uint64 {
	return s.index.degradedQueries
}

// EntityState implements EntityStore.
func (s *EntityStoreImpl) EntityState(id GameEntityID) (EntityState, bool) {
	st, ok := s.state[id]
	return st, ok
}

// subscribeSelf registers the store's internal handler at priority 1 for all
// lifecycle events (ADR-0011: Entity Framework owns the lifecycle and
// processes the transition first).
func (s *EntityStoreImpl) subscribeSelf() {
	handler := func(ev Event) {
		if data, ok := ev.Data.(EntityEventData); ok {
			s.lastEvent[data.EntityID] = ev.Type
		}
	}
	for _, eventType := range []EventType{
		EventEntityCreated, EventEntityDestroyed, EventEntitySuspended, EventEntityReactivated,
	} {
		s.events.Subscribe(eventType, 1, handler)
	}
}

// attachBridge wires the Kaiju entity lifecycle hooks (OnActivate,
// OnDeactivate, OnDestroy) to game-layer events (TR-ef-004). While the store
// drives a transition itself, bridgeSuppress prevents duplicate emission;
// when an engine-side system drives the Kaiju entity directly, the bridge
// syncs game-layer state and emits the corresponding event.
func (s *EntityStoreImpl) attachBridge(kaiju *engine.Entity, id GameEntityID) {
	kaiju.OnActivate.Add(func() { s.handleKaijuActivate(id) })
	kaiju.OnDeactivate.Add(func() { s.handleKaijuDeactivate(id) })
	kaiju.OnDestroy.Add(func() { s.handleKaijuDestroy(id) })
}

func (s *EntityStoreImpl) handleKaijuActivate(id GameEntityID) {
	if s.bridgeSuppress[id] {
		return
	}
	if s.state[id] == EntityStateSuspended {
		s.state[id] = EntityStateActive
		s.emitLifecycle(EventEntityReactivated, id)
	}
}

func (s *EntityStoreImpl) handleKaijuDeactivate(id GameEntityID) {
	if s.bridgeSuppress[id] {
		return
	}
	if s.state[id] == EntityStateActive {
		s.state[id] = EntityStateSuspended
		s.emitLifecycle(EventEntitySuspended, id)
	}
}

func (s *EntityStoreImpl) handleKaijuDestroy(id GameEntityID) {
	if s.bridgeSuppress[id] {
		return
	}
	if st, ok := s.state[id]; ok && st != EntityStateDestroyed {
		payload := s.buildPayload(id)
		s.retire(id, payload)
	}
}

// emitLifecycle builds and emits a lifecycle event for an entity.
func (s *EntityStoreImpl) emitLifecycle(eventType EventType, id GameEntityID) {
	s.events.Emit(Event{
		Type:   eventType,
		Source: "entity_framework",
		Data:   s.buildPayload(id),
	})
}

// buildPayload snapshots the entity's identity and position.
func (s *EntityStoreImpl) buildPayload(id GameEntityID) EntityEventData {
	return EntityEventData{
		EntityID:    id,
		EntityType:  s.entityTypes[id],
		DisplayName: s.displayNames[id],
		Position:    s.kaijuPosition(id),
	}
}

// kaijuPosition reads the entity position from the Kaiju transform (the
// authoritative transform source, ADR-0001). Returns the zero vector if the
// engine entity is no longer registered.
func (s *EntityStoreImpl) kaijuPosition(id GameEntityID) matrix.Vec3 {
	if kaiju := s.host.EntityById(s.idToEntityMap[id]); kaiju != nil {
		return kaiju.Transform.Position()
	}
	return matrix.Vec3{}
}

// distanceTo is the distance from an entity to a point in world units.
func (s *EntityStoreImpl) distanceTo(id GameEntityID, point matrix.Vec3) float64 {
	if kaiju := s.host.EntityById(s.idToEntityMap[id]); kaiju != nil {
		return float64(kaiju.Transform.Position().Distance(point))
	}
	return math.MaxFloat64
}

func (s *EntityStoreImpl) matchesComponents(id GameEntityID, required []string) bool {
	if len(required) == 0 {
		return true
	}
	for _, name := range required {
		st, ok := s.storesByName[name]
		if !ok || !st.contains(id) {
			return false
		}
	}
	return true
}

// removeFromOrder drops a destroyed ID from the registration-ordered list.
func (s *EntityStoreImpl) removeFromOrder(id GameEntityID) {
	for i, o := range s.order {
		if o == id {
			s.order = append(s.order[:i], s.order[i+1:]...)
			return
		}
	}
}
