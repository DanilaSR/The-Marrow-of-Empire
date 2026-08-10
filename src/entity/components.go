// Component definitions for Story EF-002 (Component Model) and EF-005
// (Serialization).
//
// Components are pure data containers (GDD CR-2): they hold state, they
// never hold logic. Systems operate on them exclusively through the typed
// component stores (GetComponent[T], SetComponent[T], RemoveComponent[T]).
// Field-level write ownership is declared in the GDD CR-7 ownership table;
// write arbitration arrives with Story EF-004.
//
// Serialization (EF-005): every component type is registered with pod at
// init time (control manifest: "all component stores implement EntityData,
// registered via pod.Register" — ADR-0001, ADR-0002). Component structs are
// pure data with no pointers, interfaces, or channels, which pod requires
// (pointer and interface fields are silently skipped; interface-typed map
// values cannot be encoded at all). Persistent.StateFlags uses the
// StateFlagValue tagged union for this reason.
package entity

import "kaijuengine.com/matrix"

// Health is the damageable-state component (GDD CR-2 #1). Attaches to
// Characters, NPCs, Wildlife, and Structures.
//
// Field ownership (GDD CR-7): Health.current and Health.temporary belong to
// the Tactical Combat Engine (damage, healing, death); Health.max belongs to
// Character Stats & Progression (changed on level-up or equipment).
type Health struct {
	// Current is the current hit points. Clamped to [0, Max] by the combat
	// system.
	Current int
	// Max is the maximum hit points.
	Max int
	// Temporary is temporary hit points granted by abilities; absorbed
	// before Current.
	Temporary int
}

// Stats is the RPG attribute block (GDD CR-2 #2, ADR-0006). Attaches to
// Characters and NPCs. Attributes range 3-18 with a human baseline of 10
// (TR-stats-001); derived values follow TR-stats-003.
//
// Field ownership (GDD CR-7): all Stats fields belong to Character Stats &
// Progression.
type Stats struct {
	// Strength is the physical power attribute.
	Strength int
	// Dexterity is the agility attribute.
	Dexterity int
	// Constitution is the endurance attribute.
	Constitution int
	// Intelligence is the reasoning attribute.
	Intelligence int
	// Wisdom is the perception attribute.
	Wisdom int
	// Charisma is the social presence attribute.
	Charisma int
	// MaxHealth is derived: 20 + Constitution*2 + Level*CON_mod (TR-stats-003).
	MaxHealth int
	// ActionPoints is derived: 4 + DEX_mod, minimum 2 (TR-stats-003).
	ActionPoints int
	// Initiative is derived: DEX_mod + WIS_mod (TR-stats-003).
	Initiative int
	// Evasion is derived: 10 + DEX_mod (TR-stats-003).
	Evasion int
}

// Inventory is the item-carrying component (GDD CR-2 #3). Attaches to
// Characters and NPCs. Items are referenced by GameEntityID — an item is
// itself an entity (EntityTypeItem).
//
// Field ownership (GDD CR-7): all Inventory fields belong to the
// Inventory/Loot System.
type Inventory struct {
	// Items are the carried item entity IDs.
	Items []GameEntityID
	// EquipmentSlots maps slot names (e.g. "weapon", "armor") to equipped
	// item entity IDs.
	EquipmentSlots map[string]GameEntityID
	// Gold is the carried coinage.
	Gold int
}

// Knowledge is the NPC knowledge-graph data (GDD CR-2 #5, ADR-0005). Attaches
// to Characters and NPCs. Fact content lives in the NPC's authored .kg file
// (assets/npc_knowledge/{region}/{npc_id}.kg, TR-kg-004), not in this
// component.
//
// Field ownership (GDD CR-7): all Knowledge fields belong to the NPC
// Knowledge Graph system.
type Knowledge struct {
	// Domains are the NPC's named expertise categories, 2-4 per NPC
	// (TR-kg-001).
	Domains []string
	// KnownFacts are fact IDs the NPC knows (TR-kg-002).
	KnownFacts []string
	// TrustThreshold is the trust level (0-3) required to reveal gated facts
	// (TR-kg-003).
	TrustThreshold int
}

// Faction is faction membership and standing (GDD CR-2 #7). Attaches to
// Characters and NPCs.
//
// Field ownership (GDD CR-7): Faction.faction_id is assigned at entity
// creation and immutable; Faction.standing belongs to the Faction System.
type Faction struct {
	// FactionID identifies the faction (e.g. "thornwell").
	FactionID string
	// Standing is the reputation value with the faction.
	Standing int
}

// Dialogue is conversation data for the AI Dialogue Engine (GDD CR-2 #8).
// Attaches to Characters and NPCs.
//
// Field ownership (GDD CR-7): all Dialogue fields belong to the AI Dialogue
// Engine.
type Dialogue struct {
	// Greeting is the NPC's opening line.
	Greeting string
	// Topics are the available conversation topics.
	Topics []string
	// PersonalityProfile is a reference to the NPC's authored personality,
	// used to build the LLM prompt (TR-ai-002).
	PersonalityProfile string
}

// StatusEffect is one entry in Combatant.StatusEffects (GDD CR-2 #9 schema).
type StatusEffect struct {
	// EffectID is the unique effect type identifier.
	EffectID string
	// SourceID is the GameEntityID of the entity that applied the effect.
	SourceID GameEntityID
	// Duration is the remaining rounds; 0 means instantaneous.
	Duration int
	// StackCount is the current stack count.
	StackCount int
	// Parameters holds effect-specific values (e.g. damage_per_tick).
	Parameters map[string]float64
}

// Combatant is combat participation data (GDD CR-2 #9). Attaches to
// Characters and NPCs.
//
// Field ownership (GDD CR-7): all Combatant fields belong to the Tactical
// Combat Engine.
type Combatant struct {
	// Initiative is rolled at combat start.
	Initiative int
	// ActionPoints is the current action points (4 + DEX_mod, min 2, max 8;
	// TR-combat-002).
	ActionPoints int
	// MovementPoints is the current movement points.
	MovementPoints int
	// MovementSpeed is the movement speed.
	MovementSpeed float64
	// Actions lists the available combat actions.
	Actions []string
	// StatusEffects lists the active status effects.
	StatusEffects []StatusEffect
	// ReactionAvailable is true while the entity's reaction is unused this
	// round.
	ReactionAvailable bool
}

// InteractionType enumerates the player interaction kinds (GDD CR-2). Each
// value dispatches to its owning system (GDD CR-7).
type InteractionType int

const (
	// InteractionTalk initiates conversation with an NPC (AI Dialogue
	// Engine).
	InteractionTalk InteractionType = iota
	// InteractionLoot opens a container or corpse inventory (Inventory/Loot).
	InteractionLoot
	// InteractionRead reads a document, inscription, or book (Investigation
	// Journal).
	InteractionRead
	// InteractionExamine looks closely at an object (Investigation Journal).
	InteractionExamine
	// InteractionOpen opens a door, chest, or gate (Living Environment).
	InteractionOpen
	// InteractionClose closes a door or gate (Living Environment).
	InteractionClose
	// InteractionUse pulls a lever, lights a torch, rings a bell —
	// system-dispatched by context.
	InteractionUse
	// InteractionPickUp takes an item from the world (Inventory).
	InteractionPickUp
	// InteractionSearch searches a container, body, or area for hidden items
	// or clues (Investigation).
	InteractionSearch
)

// Interactable marks an entity as a player interaction target (GDD CR-2 #10).
// Attaches to Characters, NPCs, Items, Props, and Triggers.
//
// Field ownership (GDD CR-7): per InteractionType — talk -> Dialogue,
// loot/pick_up -> Inventory, read/examine/search -> Investigation Journal,
// open/close -> Living Environment, use -> context-dependent.
type Interactable struct {
	// InteractionType is the kind of interaction offered.
	InteractionType InteractionType
	// PromptText is the interaction prompt shown to the player.
	PromptText string
	// Range is the interaction range in world units.
	Range float64
	// IsOnceOnly is true when the interaction is consumed after first use.
	IsOnceOnly bool
}

// MemoryLogEntry is one structured memory record in Persistent.MemoryLog
// (GDD CR-2 #13, TR-pws-002). The log is append-only and capped at 50
// entries per entity with protected entries never evicted; cap enforcement
// arrives with the Persistent World State system.
type MemoryLogEntry struct {
	// Timestamp is the game time in minutes when the event occurred.
	Timestamp float64
	// EventType identifies the structured event (e.g. "kill", "trust_change").
	EventType string
	// Details carries event-specific fields.
	Details map[string]string
}

// StateFlagKind enumerates the value kinds a StateFlagValue can hold
// (TR-pws-001: bool, int64, float64, or string).
type StateFlagKind string

const (
	// StateFlagBool marks a StateFlagValue whose Bool field carries the
	// value.
	StateFlagBool StateFlagKind = "bool"
	// StateFlagInt marks a StateFlagValue whose Int field carries the value.
	StateFlagInt StateFlagKind = "int"
	// StateFlagFloat marks a StateFlagValue whose Float field carries the
	// value.
	StateFlagFloat StateFlagKind = "float"
	// StateFlagString marks a StateFlagValue whose Str field carries the
	// value.
	StateFlagString StateFlagKind = "string"
)

// StateFlagValue is one typed value in Persistent.StateFlags (TR-pws-001):
// values are bool, int64, float64, or string. It is a tagged union — the
// active field is selected by Kind; the others are zero. It exists because
// Kaiju's pod encoder cannot encode interface-typed map values (EF-005);
// the four TR-pws-001 kinds are preserved exactly.
//
// Example usage:
//
//	persistent.StateFlags["world.door_open"] = entity.StateFlagValue{
//	    Kind: entity.StateFlagBool, Bool: true,
//	}
type StateFlagValue struct {
	// Kind selects which field carries the value.
	Kind StateFlagKind
	// Bool carries a boolean value (Kind == StateFlagBool).
	Bool bool
	// Int carries an integer value (Kind == StateFlagInt).
	Int int64
	// Float carries a floating-point value (Kind == StateFlagFloat).
	Float float64
	// Str carries a string value (Kind == StateFlagString).
	Str string
}

// Persistent is world-state tracking and entity memory (GDD CR-2 #13).
// Attaches to Characters, NPCs, Wildlife, Structures, and Props. State flag
// namespaces never conflict (GDD CR-7): trust.* (Trust & Relationship),
// world.* (Persistent World State), investigation.* and thread.* (Thread
// Investigation).
//
// Field ownership (GDD CR-7): namespaced state flags by namespace;
// Persistent.memory_log is append-only by the Entity Framework.
type Persistent struct {
	// StateFlags is a typed key-value flag store (TR-pws-001): each value is
	// a StateFlagValue tagged union holding a bool, int64, float64, or
	// string. StateFlagValue replaces the former map[string]any because pod
	// cannot encode interface-typed map values (EF-005); the TR-pws-001
	// value kinds are unchanged.
	StateFlags map[string]StateFlagValue
	// MemoryLog is the append-only structured event log (TR-pws-002).
	MemoryLog []MemoryLogEntry
	// ModifiedAt is the game time in minutes of the last modification.
	ModifiedAt float64
}

// Visual is rendering attachment data (GDD CR-2 #14). It maps to Kaiju
// rendering.Drawing fields (GDD Visual/Audio Requirements); the framework
// maintains the Drawing.Transform link, systems provide the data.
type Visual struct {
	// ModelRef is the mesh/material reference, resolved via host caches.
	ModelRef string
	// AnimationState is the animation pose name.
	AnimationState string
	// Tint is the RGB tint (damage flash, status effects, faction glow).
	Tint matrix.Vec3
	// VisibilityFlag controls rendering: 1 = stealth-hide, 2 = destroyed-hide,
	// 3 = visible (composite ownership, GDD CR-7).
	VisibilityFlag int
	// RenderLayer is the layer mask for render views.
	RenderLayer int
}

// Audio is sound attachment data (GDD CR-2 #15). The framework does not play
// audio — it provides the data the Audio System reads.
type Audio struct {
	// AmbientSound is the ambient sound loop reference.
	AmbientSound string
	// FootstepType is the footstep material type.
	FootstepType string
	// OcclusionFlag marks the sound as occluded by geometry.
	OcclusionFlag bool
}

// ScheduleEntry is one daily schedule span (GDD CR-2 #12, TR-env-004).
type ScheduleEntry struct {
	// StartTime is the activity start in game minutes.
	StartTime float64
	// EndTime is the activity end in game minutes.
	EndTime float64
	// Activity is the activity name (e.g. "working", "sleeping").
	Activity string
	// TargetPosition is the world position of the activity.
	TargetPosition matrix.Vec3
}

// Schedule is the NPC daily schedule (GDD CR-2 #12, TR-env-004). Attaches to
// NPCs. The Living Environment system advances CurrentActivity and suspends
// NPCs outside active hours.
//
// Field ownership (GDD CR-7): all Schedule fields belong to the Living
// Environment system.
type Schedule struct {
	// Entries are the daily schedule spans.
	Entries []ScheduleEntry
	// CurrentActivity is the activity of the currently active entry.
	CurrentActivity string
}

// init registers every component type with pod for serialization (control
// manifest: "all component stores implement EntityData, registered via
// pod.Register" — ADR-0001, ADR-0002) and registers its typed store factory
// (storeFactories), which Deserialize uses to rebuild stores from a save
// blob before any component of the type has been written.
//
// Nested named types (StatusEffect, MemoryLogEntry, ScheduleEntry,
// InteractionType, StateFlagValue) are part of the encoded layouts: pod
// requires every named type reachable from a serialized value to be
// registered, so they are registered here too.
func init() {
	registerComponent[Health]()
	registerComponent[Stats]()
	registerComponent[Inventory]()
	registerComponent[Knowledge]()
	registerComponent[Faction]()
	registerComponent[Dialogue]()
	registerComponent[StatusEffect]()
	registerComponent[Combatant]()
	registerComponent[Interactable]()
	registerComponent[MemoryLogEntry]()
	registerComponent[Persistent]()
	registerComponent[Visual]()
	registerComponent[Audio]()
	registerComponent[ScheduleEntry]()
	registerComponent[Schedule]()

	registerPodType(InteractionType(0))
		registerPodType(StateFlagKind(""))
	registerPodType(StateFlagValue{})
}
