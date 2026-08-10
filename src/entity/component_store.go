// Typed component stores for Story EF-002 (Component Model).
//
// Architecture: ADR-0001 (Entity Architecture), TR-ef-002. Each component
// type T gets its own store — map[GameEntityID]T — allocated lazily on the
// first write of that type. Typed access (GetComponent[T], SetComponent[T],
// RemoveComponent[T]) operates directly on the store: one map lookup keyed
// by the compile-time type descriptor (*T)(nil), one type assertion, no
// reflection, no interface boxing. The string-keyed interface methods
// (GetComponentData, SetComponentData, RemoveComponentData) and
// QueryEntities go through the erased componentStore view instead.
package entity

import (
	"fmt"
	"reflect"
)

// typedStore[T] is the typed per-component store: map[GameEntityID]T
// (ADR-0001: "map[GameEntityID]ComponentType per component").
//
// Stores are created lazily by storeFor — the map is allocated only when a
// component of type T is first written (GDD CR-2: "nil maps cost nothing";
// ADR-0001 risk mitigation: no per-type overhead for unused components).
// The type name is resolved once at creation; subsequent typed accesses
// never touch reflection.
type typedStore[T any] struct {
	owner  *EntityStoreImpl
	name   string
	values map[GameEntityID]T
}

// componentStore is the type-erased view of a typed store. Only the
// string-keyed interface methods and QueryEntities use it; the typed hot
// path never boxes through it.
type componentStore interface {
	contains(id GameEntityID) bool
	get(id GameEntityID) (any, bool)
	setErased(id GameEntityID, value any) bool
	take(id GameEntityID) bool
	removeEntity(id GameEntityID)
}

// newTypedStore creates a typed store and resolves its type name once.
func newTypedStore[T any](owner *EntityStoreImpl) *typedStore[T] {
	return &typedStore[T]{
		owner:  owner,
		name:   typeName[T](),
		values: make(map[GameEntityID]T),
	}
}

// contains reports whether the entity has a component of this type.
func (st *typedStore[T]) contains(id GameEntityID) bool {
	_, ok := st.values[id]
	return ok
}

// get returns the component boxed as any (erased path only; allocates for
// multi-word structs). Returns (nil, false) when the entity has no component
// of this type.
func (st *typedStore[T]) get(id GameEntityID) (any, bool) {
	v, ok := st.values[id]
	if !ok {
		return nil, false
	}
	return v, true
}

// setErased stores a value through the erased path. Returns false when the
// value is not a T (the caller reports the type error).
func (st *typedStore[T]) setErased(id GameEntityID, value any) bool {
	v, ok := value.(T)
	if !ok {
		return false
	}
	st.put(id, v)
	return true
}

// put stores a component, emitting component.added on first attachment and
// component.changed with old and new values on replacement (Story EF-002
// AC-5).
func (st *typedStore[T]) put(id GameEntityID, v T) {
	old, existed := st.values[id]
	st.values[id] = v
	if existed {
		st.owner.emitComponentEvent(EventComponentChanged, id, st.name, old, v)
	} else {
		st.owner.emitComponentEvent(EventComponentAdded, id, st.name, nil, v)
	}
}

// take removes a component, emitting component.removed with the removed
// value. Returns false when the entity has no such component — removal is
// idempotent and no event fires.
func (st *typedStore[T]) take(id GameEntityID) bool {
	v, ok := st.values[id]
	if !ok {
		return false
	}
	delete(st.values, id)
	st.owner.emitComponentEvent(EventComponentRemoved, id, st.name, v, nil)
	return true
}

// removeEntity drops an entity's data without emitting events. Used during
// entity destruction: entity.destroyed is the single destruction
// notification, and per-component events would storm subscribers (Story
// EF-002: component events cover explicit removal only).
func (st *typedStore[T]) removeEntity(id GameEntityID) {
	delete(st.values, id)
}

// storeFor returns the typed store for T, creating it lazily on first use.
// The registry is keyed by the type descriptor pointer (*T)(nil) — a
// compile-time value, unique per type — so lookups never use reflection.
// storesByName mirrors the same store for the string-keyed paths. Go does
// not allow generic methods, so the typed store accessors are package-level
// functions taking the store.
func storeFor[T any](s *EntityStoreImpl) *typedStore[T] {
	key := (*T)(nil)
	if raw, ok := s.stores[key]; ok {
		return raw.(*typedStore[T])
	}
	st := newTypedStore[T](s)
	s.stores[key] = st
	s.storesByName[st.name] = st
	return st
}

// getComponentTyped is the typed read path: descriptor lookup, one type
// assertion, direct map read. Returns (zero, false) for destroyed or unknown
// entities, for component types never written, and for missing components.
// It never allocates a store — reads of never-written types are free.
func getComponentTyped[T any](s *EntityStoreImpl, id GameEntityID) (T, bool) {
	var zero T
	raw, ok := s.stores[(*T)(nil)]
	if !ok {
		return zero, false
	}
	v, ok := raw.(*typedStore[T]).values[id]
	if !ok {
		return zero, false
	}
	return v, true
}

// setComponentTyped is the typed write path. Components may be set on Active
// and Suspended entities and are preserved across suspension. Unknown and
// destroyed entities are rejected with an error and no state change.
func setComponentTyped[T any](s *EntityStoreImpl, id GameEntityID, component T) error {
	if !s.isLive(id) {
		return fmt.Errorf("entity: no live entity with id %d", id)
	}
	storeFor[T](s).put(id, component)
	return nil
}

// removeComponentTyped is the typed removal path. Removing a component the
// entity does not have is a no-op (nil error, no event); unknown and
// destroyed entities are rejected with an error.
func removeComponentTyped[T any](s *EntityStoreImpl, id GameEntityID) error {
	if !s.isLive(id) {
		return fmt.Errorf("entity: no live entity with id %d", id)
	}
	if raw, ok := s.stores[(*T)(nil)]; ok {
		raw.(*typedStore[T]).take(id)
	}
	return nil
}

// isLive reports whether the entity exists and is not destroyed. Suspended
// entities are live: component data is preserved and mutable across
// suspension (GDD CR-4).
func (s *EntityStoreImpl) isLive(id GameEntityID) bool {
	st, ok := s.state[id]
	return ok && st != EntityStateDestroyed
}

// emitComponentEvent builds and emits a component lifecycle event with a
// ComponentEventData payload (Story EF-002).
func (s *EntityStoreImpl) emitComponentEvent(eventType EventType, id GameEntityID, componentType string, oldValue, newValue any) {
	s.events.Emit(Event{
		Type:   eventType,
		Source: "entity_framework",
		Data: ComponentEventData{
			EntityID:      id,
			ComponentType: componentType,
			OldValue:      oldValue,
			NewValue:      newValue,
		},
	})
}

// GetComponent is the type-safe component accessor (ADR-0001, TR-ef-002).
// The component type must be a named type. Returns (zero, false) for
// destroyed or unknown entities and for missing components.
//
// On *EntityStoreImpl the read is fully typed: no reflection, no interface
// boxing. Other EntityStore implementations fall back to the erased
// GetComponentData path.
//
// Example usage:
//
//	hp, ok := entity.GetComponent[entity.Health](store, id)
//	if !ok { /* entity has no Health component */ }
func GetComponent[T any](store EntityStore, id GameEntityID) (T, bool) {
	if s, ok := store.(*EntityStoreImpl); ok {
		return getComponentTyped[T](s, id)
	}
	var zero T
	comp, ok := store.GetComponentData(id, typeName[T]())
	if !ok {
		return zero, false
	}
	typed, ok := comp.(T)
	return typed, ok
}

// SetComponent is the type-safe component mutator (ADR-0001, TR-ef-002).
// Stores the component, preserving it across suspension/reactivation. The
// first SetComponent of a type on an entity fires component.added; a
// replacement fires component.changed with the old and new values. Returns
// an error if the entity is destroyed or unknown.
//
// Example usage:
//
//	err := entity.SetComponent(store, id, entity.Health{Current: 100, Max: 100})
func SetComponent[T any](store EntityStore, id GameEntityID, component T) error {
	if s, ok := store.(*EntityStoreImpl); ok {
		return setComponentTyped[T](s, id, component)
	}
	return store.SetComponentData(id, typeName[T](), component)
}

// RemoveComponent is the type-safe component remover (Story EF-002). Fires
// component.removed with the removed value. Removing a component the entity
// does not have is a no-op (nil error, no event). Returns an error if the
// entity is destroyed or unknown. Entity destruction purges all components
// without firing per-component events.
//
// Example usage:
//
//	err := entity.RemoveComponent[entity.Health](store, id)
func RemoveComponent[T any](store EntityStore, id GameEntityID) error {
	if s, ok := store.(*EntityStoreImpl); ok {
		return removeComponentTyped[T](s, id)
	}
	return store.RemoveComponentData(id, typeName[T]())
}

// typeName resolves the string key for a component type. Components must be
// named types (anonymous types resolve to "", colliding with each other).
// The typed stores resolve it once per type at creation; only the erased
// fallback paths of the generic helpers resolve it per call.
func typeName[T any]() string {
	return reflect.TypeOf((*T)(nil)).Elem().Name()
}
