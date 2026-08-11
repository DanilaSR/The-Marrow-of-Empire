package entity

import (
	"log/slog"
	"reflect"
	"sort"
)

// EventType is a namespaced string identifier for event types (ADR-0011).
type EventType string

// Entity lifecycle event types (ADR-0011). The Entity Framework is the sole
// producer of these events; it subscribes to its own events at priority 1.
const (
	// EventEntityCreated fires on Pending -> Active.
	EventEntityCreated EventType = "entity.created"
	// EventEntityDestroyed fires on Active|Suspended -> Destroyed.
	EventEntityDestroyed EventType = "entity.destroyed"
	// EventEntitySuspended fires on Active -> Suspended.
	EventEntitySuspended EventType = "entity.suspended"
	// EventEntityReactivated fires on Suspended -> Active.
	EventEntityReactivated EventType = "entity.reactivated"
)

// Component lifecycle event types (Story EF-002). The Entity Framework is
// the sole producer; payloads are ComponentEventData. Entity destruction
// does not fire component events — entity.destroyed is the single
// destruction notification.
const (
	// EventComponentAdded fires when a component is first attached to an
	// entity (the first SetComponent of that type).
	EventComponentAdded EventType = "component.added"
	// EventComponentRemoved fires when a component is explicitly detached
	// via RemoveComponent; the payload carries the removed value.
	EventComponentRemoved EventType = "component.removed"
	// EventComponentChanged fires when SetComponent replaces an existing
	// component; the payload carries the old and new values (EF-002 AC-5).
	EventComponentChanged EventType = "component.changed"
)

// Event is a lightweight event payload (ADR-0011). Events are dispatched
// synchronously — all subscribers are notified within the same frame.
type Event struct {
	// Type identifies the event; subscribers filter on it.
	Type EventType
	// Timestamp is the game time in minutes when the event fired. The game
	// clock (ADR-0007) is not implemented yet, so this is always 0 for now.
	Timestamp float64
	// Source is the system that emitted the event.
	Source string
	// Data is the type-specific payload; consumers cast to the expected type.
	Data any
}

// EventHandler receives dispatched events.
type EventHandler func(event Event)

// EventBus is the central synchronous event dispatcher (ADR-0011).
//
// Subscribers register per event type with a priority; priority 1 dispatches
// first. Lower priority values take precedence. Ties are broken by
// registration order (earlier registrations dispatch first). Dispatch is
// synchronous: Emit returns only after every subscriber has been notified.
//
// Example usage:
//
//	bus.Subscribe(EventEntityDestroyed, 2, func(ev entity.Event) {
//	    data := ev.Data.(entity.EntityEventData)
//	    log.Info("entity destroyed", "id", data.EntityID)
//	})
type EventBus interface {
	// Subscribe registers a handler for an event type with a priority.
	// Lower priority = higher precedence. Priority 1 dispatches first.
	// Ties are broken by registration order.
	Subscribe(eventType EventType, priority int, handler EventHandler)

	// Emit dispatches an event to all subscribers synchronously, in priority
	// order, within the same frame.
	Emit(event Event)

	// Unsubscribe removes a handler previously registered with Subscribe for
	// the given event type. Handlers are matched by code-pointer identity
	// (function values are not directly comparable in Go); pass back the
	// exact handler value from Subscribe. Removing a handler that is not
	// registered is a no-op. Subscribers that live for the whole process
	// never need it; systems that tear down must call it so registrations
	// never outlive their owner.
	Unsubscribe(eventType EventType, handler EventHandler)
}

// EventBusImpl is the concrete EventBus. It holds subscriber lists per event
// type, kept sorted by priority at subscription time so dispatch is a single
// ordered slice iteration with no per-emit allocation.
type EventBusImpl struct {
	subscribers map[EventType][]subscriber
	// inDispatch guards against recursive emission of the same event type
	// during its own dispatch (ADR-0011 mitigation: suppressed with warning).
	inDispatch map[EventType]bool
}

// subscriber is a single registered handler with its priority.
type subscriber struct {
	priority int
	handler  EventHandler
}

// NewEventBus returns an empty EventBusImpl ready for subscription.
func NewEventBus() *EventBusImpl {
	return &EventBusImpl{
		subscribers: make(map[EventType][]subscriber),
		inDispatch:  make(map[EventType]bool),
	}
}

// Subscribe registers a handler for an event type with a priority and keeps
// the subscriber list sorted: ascending priority (1 dispatches first), stable
// so equal-priority registrations keep insertion order (GDD: registration
// order breaks ties deterministically).
func (b *EventBusImpl) Subscribe(eventType EventType, priority int, handler EventHandler) {
	list := b.subscribers[eventType]
	list = append(list, subscriber{priority: priority, handler: handler})
	sort.SliceStable(list, func(i, j int) bool {
		return list[i].priority < list[j].priority
	})
	b.subscribers[eventType] = list
}

// Emit dispatches an event synchronously to all subscribers of its type, in
// priority order. Recursive emission of the same event type during its own
// dispatch is suppressed with a warning (ADR-0011).
func (b *EventBusImpl) Emit(event Event) {
	if b.inDispatch[event.Type] {
		slog.Warn("recursive event emission suppressed", "type", event.Type, "source", event.Source)
		return
	}
	b.inDispatch[event.Type] = true
	defer delete(b.inDispatch, event.Type)
	for _, sub := range b.subscribers[event.Type] {
		sub.handler(event)
	}
}

// Unsubscribe removes the first registration of handler for eventType,
// matched by code-pointer identity (function values are not directly
// comparable in Go, so reflection compares the underlying code pointers).
// Pass back the exact handler value from Subscribe. Removing a handler
// that was never registered is a no-op. Method values of the same method
// on different receivers share a code pointer, so when several subscribers
// use the same method the oldest matching registration is removed — with
// one subscriber per handler value this is exact.
func (b *EventBusImpl) Unsubscribe(eventType EventType, handler EventHandler) {
	list := b.subscribers[eventType]
	for i, sub := range list {
		if sameHandler(sub.handler, handler) {
			b.subscribers[eventType] = append(list[:i], list[i+1:]...)
			return
		}
	}
}

// sameHandler reports whether two handlers are the same function value by
// comparing their code pointers via reflection.
func sameHandler(a, b EventHandler) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return reflect.ValueOf(a).Pointer() == reflect.ValueOf(b).Pointer()
}
