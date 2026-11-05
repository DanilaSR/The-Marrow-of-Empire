// Follow and framing for Story CAM-003: constant-velocity move-toward follow,
// conversation and combat framing, and the camera → HUD off-screen marker data
// contract (GDD CR-3, CR-7, Off-Screen Markers).
//
// Position data source contract: the Follower reads Transform.Position()
// directly for tracked entities (party 1-6, combatants ~10-30, one speaker)
// through PositionSource — never a spatial-index range query (GDD CR-3
// "Position data source"). Membership changes arrive through the Track/Untrack
// APIs; the game layer wires them to EF component lifecycle events
// (on_component_added/on_component_removed for PartyMember and Combatant).
//
// Follow semantics (GDD CR-3): constant-velocity move-toward with distance
// clamping — step = min(follow_speed × dt, distance); the camera reaches the
// target exactly. This is NOT an exponential ease.
package camera

import (
	"fmt"
	"math"

	"kaijuengine.com/engine"
	"kaijuengine.com/matrix"

	"marrow-of-empire/entity"
)

// ── Isometric projection constants ───────────────────────────────────────────
//
// Screen axes from the fixed 30° pitch / 45° yaw camera (GDD "Combat Bounding
// Box Fit Test"): a world offset v = (dx, dy, dz) from the focus projects to
//   screen_x = (dx − dz) × cos(45°)          (world units)
//   screen_y = (dx + dz) × cos(45°)/2 − dy × cos(30°)
// A world AABB of side S therefore spans S×1.414 horizontally and S×0.707
// vertically on screen, matching the GDD fit test.
const (
	isoCos45    = 0.7071067811865476 // cos(45°) — screen-x world scale
	isoCos45Half = 0.3535533905932738 // cos(45°)/2 — screen-y world scale from XZ
	isoSin60    = 0.8660254037844387 // cos(30°) — screen-y world scale from elevation
)

// minCombatFrameSide is the minimum padded framing box side in the XZ plane
// for combat framing (GDD CR-7 rule 1 / AC-11: single combatant → 8×8u box).
const minCombatFrameSide float32 = 8

// followBoxMargin is the inner-90% viewport margin of the framing fit test:
// a box fits at zoom O when S × 1.414 ≤ 2 × O × AR × 0.9 (GDD CR-7).
const followBoxMargin float32 = 0.9

// ── Position source ─────────────────────────────────────────────────────────

// PositionSource reads the authoritative world position of a tracked entity
// from its Kaiju transform (ADR-0001: Transform.Position() is authoritative).
// The Follower reads a small fixed set — party, combatants, speaker — once per
// frame, never through the spatial index (GDD CR-3). Implementations must be
// cheap and allocation-free after warm-up.
type PositionSource interface {
	// Position returns the entity's current world position. Returns
	// (zero, false) when the entity no longer exists.
	Position(id entity.GameEntityID) (matrix.Vec3, bool)

	// DisplayName returns the entity's human-readable name for marker labels.
	// Returns "" for unknown entities.
	DisplayName(id entity.GameEntityID) string
}

// HostPositionSource adapts an entity.KaijuHost to PositionSource. It resolves
// the Kaiju EntityId ("game_{GameEntityID}", the entity framework's mapping
// scheme) and reads Transform.Position() directly — the authoritative position
// source per ADR-0001. The formatted EntityId is cached per GameEntityID on
// first lookup so per-frame reads are allocation-free after warm-up.
//
// Example usage:
//
//	pos := camera.NewHostPositionSource(host)
//	flw := camera.NewFollower(cam, pos, camera.DefaultFollowConfig())
type HostPositionSource struct {
	Host entity.KaijuHost

	// ids caches the formatted Kaiju EntityId per GameEntityID. Lookups are
	// cheap map reads; the cache is bounded by the entities ever tracked in a
	// session (IDs are never reused, ADR-0001).
	ids map[entity.GameEntityID]engine.EntityId
}

// NewHostPositionSource creates a HostPositionSource for the given host.
func NewHostPositionSource(host entity.KaijuHost) *HostPositionSource {
	return &HostPositionSource{
		Host: host,
		ids:  make(map[entity.GameEntityID]engine.EntityId),
	}
}

// Position implements PositionSource: reads Transform.Position() from the
// mapped Kaiju entity. Returns (zero, false) for unknown entities.
func (s *HostPositionSource) Position(id entity.GameEntityID) (matrix.Vec3, bool) {
	kaiju := s.Host.EntityById(s.kaijuID(id))
	if kaiju == nil {
		return matrix.Vec3{}, false
	}
	return kaiju.Transform.Position(), true
}

// DisplayName implements PositionSource: reads the Kaiju entity name.
func (s *HostPositionSource) DisplayName(id entity.GameEntityID) string {
	kaiju := s.Host.EntityById(s.kaijuID(id))
	if kaiju == nil {
		return ""
	}
	return kaiju.Name()
}

// kaijuID resolves the Kaiju EntityId for a GameEntityID, caching the
// formatted string after the first lookup.
func (s *HostPositionSource) kaijuID(id entity.GameEntityID) engine.EntityId {
	if kaijuID, ok := s.ids[id]; ok {
		return kaijuID
	}
	kaijuID := engine.EntityId(fmt.Sprintf("game_%d", id))
	s.ids[id] = kaijuID
	return kaijuID
}

// ── Configuration ────────────────────────────────────────────────────────────

// FollowConfig carries the follow/framing tuning knobs (GDD tuning knob
// table). Zero-valued fields are honored as-is by callers that construct the
// config explicitly; DefaultFollowConfig returns the GDD values.
type FollowConfig struct {
	// FollowSpeedExploration is the exploration follow speed in world units
	// per second (GDD follow_speed_exploration, default 5). Must exceed the
	// party walk speed (≥1.2× — movement GDD pending).
	FollowSpeedExploration float32
	// FollowSpeedCombat is the gentle combat re-center speed used when a
	// re-frame fires over the player's manual adjustment (joiner re-frame
	// overrides the manual flag, GDD CR-7 rule 7). Combat-start framing uses
	// CombatFramingSpeed per GDD CR-7; this knob covers the rare re-center
	// cases (default 8).
	FollowSpeedCombat float32
	// CombatFramingSpeed is the combat-start framing speed in world units per
	// second (GDD combat_framing_speed, default 40).
	CombatFramingSpeed float32
	// ConversationSnapDurationMs is the conversation snap duration in
	// milliseconds (GDD conversation_snap_duration, default 200). The pan to
	// the PC+NPC midpoint takes snap × 2 (400ms), per AC-7.
	ConversationSnapDurationMs float32
	// ConversationOverrideWindow is the window in seconds after a manual
	// adjustment during which a speaker change does not re-frame the camera
	// (GDD conversation_override_window, default 5).
	ConversationOverrideWindow float32
	// MarkerEdgeInsetPx is the minimum distance of off-screen marker anchors
	// from the viewport edge in logical pixels (GDD: ≥ 32px).
	MarkerEdgeInsetPx float32
}

// DefaultFollowConfig returns the GDD-prescribed follow/framing tuning values.
func DefaultFollowConfig() FollowConfig {
	return FollowConfig{
		FollowSpeedExploration:      5,
		FollowSpeedCombat:           8,
		CombatFramingSpeed:          40,
		ConversationSnapDurationMs:  200,
		ConversationOverrideWindow:  5,
		MarkerEdgeInsetPx:           32,
	}
}

// ── Marker data contract (camera → HUD) ─────────────────────────────────────

// TrackedKind is the entity category in the off-screen marker data contract
// (GDD Off-Screen Markers). The camera classifies tracked entities; the HUD
// maps kinds to visual styles (solid amber chevron for party, dashed red for
// hostiles, etc.).
type TrackedKind int

const (
	// TrackedPartyMember is a party member (always tracked, always shown).
	TrackedPartyMember TrackedKind = iota
	// TrackedFriendlyCombatant is a friendly non-party combatant (reserved;
	// no friendly non-party combatants exist yet — classification arrives with
	// the combat engine).
	TrackedFriendlyCombatant
	// TrackedHostileCombatant is an enemy combatant (a combatant that is not
	// a party member).
	TrackedHostileCombatant
	// TrackedNeutral is a neutral entity (reserved).
	TrackedNeutral
)

// ScreenEdge identifies the viewport edge nearest an off-screen entity.
type ScreenEdge int

const (
	// EdgeLeft is the left viewport edge.
	EdgeLeft ScreenEdge = iota
	// EdgeTop is the top viewport edge.
	EdgeTop
	// EdgeRight is the right viewport edge.
	EdgeRight
	// EdgeBottom is the bottom viewport edge.
	EdgeBottom
)

// EdgeMarker is one off-screen marker entry (GDD Off-Screen Markers data
// contract, AC-23): the entity, the nearest screen edge, and the anchor point
// where the HUD places the chevron. Markers are produced for every tracked
// entity outside the viewport, regardless of camera state.
type EdgeMarker struct {
	// EntityID is the stable reference for selection/focus.
	EntityID entity.GameEntityID
	// DisplayName is the character name for the label.
	DisplayName string
	// Kind is the entity category (party member, hostile combatant, ...).
	Kind TrackedKind
	// Edge is the viewport edge nearest the entity's screen position.
	Edge ScreenEdge
	// AnchorX and AnchorY are the marker anchor in logical pixels, clamped to
	// the viewport rect with the edge inset applied.
	AnchorX float32
	// AnchorY is the marker anchor Y in logical pixels.
	AnchorY float32
	// Distance is the distance in world units from the camera focus to the
	// entity, for the distance label.
	Distance float32
}

// ── Follower ────────────────────────────────────────────────────────────────

// Follower computes the camera's follow target per base state and drives the
// camera focus toward it with constant-velocity move-toward (GDD CR-3). It
// owns tracked-entity membership (party, combatants), the player character,
// the selected character, and the speaking NPC; conversation and combat
// framing (target + zoom fit); and the off-screen marker data contract.
//
// The Follower is state-aware only through the base state passed to Update and
// the lifecycle calls from the Controller; it holds no reference to the State
// machine. It never allocates in the per-frame path: membership is stored in
// maps, ordering uses in-place sorts on preallocated scratch slices, and all
// motion is pure float arithmetic.
type Follower struct {
	cam *Isometric
	pos PositionSource
	cfg FollowConfig

	party      map[entity.GameEntityID]struct{}
	combatants map[entity.GameEntityID]struct{}

	// playerCharacter is the PC used for the conversation midpoint. Zero when
	// unset.
	playerCharacter entity.GameEntityID
	// selected is the currently selected character. Zero when nothing is
	// selected.
	selected entity.GameEntityID
	// speaker is the currently speaking NPC. Zero when no dialogue is active.
	speaker entity.GameEntityID

	// inConversation and inCombat track active base-state lifecycles so the
	// re-frame rules (speaker change, turn boundary, joiner) can gate on them.
	inConversation bool
	inCombat       bool

	// overrideTimer counts down the conversation_override_window after the
	// last manual adjustment; while > 0, speaker changes do not re-frame.
	overrideTimer float32

	// Conversation framing state (in-flight timed pan to PC+NPC midpoint).
	convFraming bool
	convTarget  matrix.Vec3
	convSpeed   float32

	// Combat framing state (in-flight move to the combatant box center).
	combatFraming bool
	combatTarget  matrix.Vec3
	combatSpeed   float32
	// framedCount is the number of combatants covered by the current frame;
	// a higher count on join triggers an immediate re-frame (GDD CR-7 rule 7).
	framedCount int
	// manualBlocked suppresses turn-boundary re-frames after the player
	// manually panned/zoomed during the encounter (AC-12). Joiner re-frames
	// still fire regardless (AC-13).
	manualBlocked bool
	// turnReframePending is set when a joiner arrives that does not exceed
	// framedCount; it re-frames at the next turn boundary regardless of the
	// manual flag (AC-13).
	turnReframePending bool

	// markerScratch is the reusable output buffer of OffScreenMarkers
	// (zero-alloc contract: the returned slice aliases this buffer and is
	// invalidated by the next call).
	markerScratch []EdgeMarker
	// enemyScratch holds (id, distance) pairs for the nearest-75%-enemies
	// framing trim; sized on combat start only (event-driven, not per-frame).
	enemyScratch []enemyDist
	// boxScratch holds the combined party + selected-enemy id list for the
	// trimmed framing box computation.
	boxScratch []entity.GameEntityID
}

// enemyDist is one enemy candidate for the nearest-75% framing trim.
type enemyDist struct {
	id   entity.GameEntityID
	dist float32
}

// NewFollower creates a Follower for the given camera, position source, and
// config. The Follower begins with empty tracked sets; the game layer feeds it
// membership via Track/Untrack (wired to component lifecycle events) and the
// Controller drives its lifecycle.
func NewFollower(cam *Isometric, pos PositionSource, cfg FollowConfig) *Follower {
	return &Follower{
		cam:        cam,
		pos:        pos,
		cfg:        cfg,
		party:      make(map[entity.GameEntityID]struct{}),
		combatants: make(map[entity.GameEntityID]struct{}),
	}
}

// ── Membership (wire to EF component lifecycle events) ───────────────────────

// TrackPartyMember adds an entity to the tracked party set (the game layer
// calls this on component.added for the PartyMember component). Follow and
// marker computation read Transform.Position() directly for every member.
func (f *Follower) TrackPartyMember(id entity.GameEntityID) {
	f.party[id] = struct{}{}
}

// UntrackPartyMember removes an entity from the tracked party set (the game
// layer calls this on component.removed). Removing an untracked ID is a no-op.
func (f *Follower) UntrackPartyMember(id entity.GameEntityID) {
	delete(f.party, id)
}

// TrackCombatant adds an entity to the tracked combatant set (the game layer
// calls this on component.added for the Combatant component). Party members
// participating in combat are combatants too — they appear in both sets and
// are framed once.
func (f *Follower) TrackCombatant(id entity.GameEntityID) {
	f.combatants[id] = struct{}{}
}

// UntrackCombatant removes an entity from the tracked combatant set (the game
// layer calls this on component.removed). Removing an untracked ID is a no-op.
func (f *Follower) UntrackCombatant(id entity.GameEntityID) {
	delete(f.combatants, id)
}

// ── Context (set by the game layer / Controller) ─────────────────────────────

// SetPlayerCharacter sets the player character used for the conversation
// midpoint. Zero clears the reference.
func (f *Follower) SetPlayerCharacter(id entity.GameEntityID) {
	f.playerCharacter = id
}

// SetSelected sets the currently selected character. Exploration follow uses
// the selection when the party spread exceeds one viewport half-extent
// (AC-3); End-centering and marker selection flags read it. Zero clears the
// selection. The Controller clears the Manual overlay on selection so base
// framing resumes (GDD: selecting a party member re-engages follow).
func (f *Follower) SetSelected(id entity.GameEntityID) {
	f.selected = id
}

// SetSpeaker sets the currently speaking NPC. During an active conversation a
// speaker change re-frames the PC+NPC midpoint only when the new speaker is
// outside the inner 70% of the viewport AND the player has not manually
// adjusted within conversation_override_window (GDD CR-3 / AC-5, AC-6). The
// speaker reference itself is always updated.
func (f *Follower) SetSpeaker(id entity.GameEntityID) {
	f.speaker = id
	if !f.inConversation {
		return
	}
	if f.overrideTimer > 0 {
		return // player adjusted within the window: no re-frame (AC-6)
	}
	if f.speakerOnScreen(id) {
		return // new speaker still within the inner 70%: no re-frame (AC-5)
	}
	f.beginConversationFraming()
}

// NoteManualAdjust records a manual camera adjustment (drag pan, edge scroll,
// keyboard pan, or wheel zoom). It starts the conversation override window,
// suppresses turn-boundary re-frames for the combat encounter (AC-12), and
// cancels any in-flight framing move (the player may interrupt framing by
// panning, GDD CR-7 rule 6).
func (f *Follower) NoteManualAdjust() {
	f.overrideTimer = f.cfg.ConversationOverrideWindow
	if f.inCombat {
		f.manualBlocked = true
	}
	f.CancelFraming()
}

// ── Lifecycle (driven by the Controller on state transitions) ───────────────

// StartConversation begins conversation framing: the camera zooms to the last
// Conversation zoom if the PC+NPC pair fits within the inner 90% of the
// viewport, or to the minimum zoom register that fits otherwise (AC-4), and
// pans to the pair midpoint over conversation_snap_duration × 2 (AC-7).
func (f *Follower) StartConversation() {
	f.inConversation = true
	_, side, ok := f.conversationPair()
	if !ok {
		// No PC or speaker: hold position; the controller's SetContext still
		// restores the Conversation zoom.
		return
	}
	candidate := f.cam.contextZoom(ConversationContext)
	if !f.boxFits(side, candidate.HalfHeight) {
		if fit, ok := f.zoomThatFits(side); ok {
			candidate = fit
		}
	}
	if f.cam.ZoomState() != candidate {
		f.cam.ZoomTo(candidate)
	}
	f.beginConversationFraming()
}

// EndConversation leaves conversation framing: any in-flight pan is cancelled
// and the camera holds position. The Controller restores the Exploration zoom
// context (AC-8) and resumes exploration follow.
func (f *Follower) EndConversation() {
	f.inConversation = false
	f.convFraming = false
}

// StartCombat begins combat framing (GDD CR-7 rule 1): the camera frames the
// bounding rectangle of all active combatants in the XZ plane — framing zoom
// is the player's Combat zoom at minimum Default; Far when the box does not
// fit at Default; when it does not fit at Far either, the frame covers the
// party plus the nearest 75% of enemies and the remainder receive edge
// markers (AC-10). The box is padded to a minimum 8×8u (AC-11) and the camera
// moves to the box center at combat_framing_speed (AC-9). Zoom is preserved
// on subsequent re-frames (AC-13).
func (f *Follower) StartCombat() {
	f.inCombat = true
	f.manualBlocked = false
	f.turnReframePending = false
	f.combatFraming = false

	center, side, ok := f.combatBoxAll()
	if !ok {
		f.framedCount = 0
		return
	}
	f.framedCount = len(f.combatants)

	framingZoom := f.cam.contextZoom(CombatContext)
	if framingZoom.HalfHeight < ZoomDefault.HalfHeight {
		framingZoom = ZoomDefault // minimum Default (GDD CR-7 rule 1)
	}
	if !f.boxFits(side, framingZoom.HalfHeight) {
		framingZoom = ZoomFar
		if !f.boxFits(side, ZoomFar.HalfHeight) {
			// Oversized encounter: frame party + nearest 75% of enemies; the
			// remainder get edge markers (AC-10).
			center, side, ok = f.combatBoxTrimmed()
			if !ok {
				return
			}
		}
	}
	if f.cam.ZoomState() != framingZoom {
		f.cam.ZoomTo(framingZoom)
	}

	f.combatTarget = center
	f.combatSpeed = f.cfg.CombatFramingSpeed
	f.combatFraming = true
}

// EndCombat leaves combat framing: in-flight framing is cancelled, the
// per-encounter manual block and turn-reframe pendings are cleared. The
// Controller restores the previous base state's zoom context and framing.
func (f *Follower) EndCombat() {
	f.inCombat = false
	f.combatFraming = false
	f.manualBlocked = false
	f.turnReframePending = false
	f.framedCount = 0
}

// NoteTurnBoundary advances the turn-boundary re-frame rules (GDD CR-7 rule 3,
// AC-12): the camera re-frames the combatant box only when the player has not
// manually adjusted during the encounter. A joiner pending from
// NoteCombatantJoined fires regardless of the manual flag (AC-13).
func (f *Follower) NoteTurnBoundary() {
	if !f.inCombat {
		return
	}
	if f.turnReframePending {
		f.turnReframePending = false
		f.reframeCombat()
		return
	}
	if !f.manualBlocked {
		f.reframeCombat()
	}
}

// NoteCombatantJoined handles new combatants joining mid-encounter (GDD CR-7
// rule 7, AC-13): re-frame immediately when the new count exceeds the framed
// count; otherwise re-frame at the next turn boundary. Both fire regardless of
// the manual flag; the player's zoom is preserved.
func (f *Follower) NoteCombatantJoined() {
	if !f.inCombat {
		return
	}
	if len(f.combatants) > f.framedCount {
		f.reframeCombat()
		return
	}
	f.turnReframePending = true
}

// reframeCombat re-targets the camera toward the current combatant box center
// at the framing speed — or, when the re-frame fires over the player's manual
// adjustment, at the gentler FollowSpeedCombat (the rare re-center case).
func (f *Follower) reframeCombat() {
	center, _, ok := f.combatBoxAll()
	if !ok {
		return
	}
	f.combatTarget = center
	if f.manualBlocked {
		f.combatSpeed = f.cfg.FollowSpeedCombat
	} else {
		f.combatSpeed = f.cfg.CombatFramingSpeed
	}
	f.combatFraming = true
	f.framedCount = len(f.combatants)
}

// ── Instant centering (Home / End keys, GDD CR-3) ────────────────────────────

// Home instantly centers the camera on the party centroid — no intermediate
// positions, zoom unchanged (AC-16). It cancels any in-flight framing. The
// Controller clears the Manual overlay so follow resumes.
func (f *Follower) Home() {
	centroid, ok := f.partyCentroid()
	if !ok {
		return
	}
	f.cam.SetFocus(centroid)
	f.CancelFraming()
}

// CenterOnSelected instantly centers the camera on the selected character
// within one frame (AC-17). With no selection, it is a no-op. Zoom unchanged.
func (f *Follower) CenterOnSelected() {
	if f.selected == 0 {
		return // no selection: no-op (AC-17)
	}
	pos, ok := f.pos.Position(f.selected)
	if !ok {
		return
	}
	f.cam.SetFocus(pos)
	f.CancelFraming()
}

// CancelFraming cancels any in-flight conversation or combat framing move
// without moving the camera. Manual input and instant centering call this so
// a stale frame target never drags the camera after the player takes control.
func (f *Follower) CancelFraming() {
	f.convFraming = false
	f.combatFraming = false
}

// ── Per-frame update ─────────────────────────────────────────────────────────

// Update advances camera motion for the given base state by dt seconds. The
// Controller calls this only when follow is active (not Manual, Journal, or
// Transition overlay): Exploration follows the party centroid (or the selected
// character when the party is spread wider than one viewport half-extent),
// Conversation pans to the framed midpoint while a conversation framing move
// is in flight, and Combat moves to the framed box center while a combat
// framing move is in flight. After framing completes the camera holds — there
// is no continuous chase in Conversation or Combat (GDD CR-3, CR-7 rule 2).
// A non-positive or NaN dt is ignored. This path is allocation-free.
func (f *Follower) Update(dt float32, base BaseState) {
	if dt <= 0 || math.IsNaN(float64(dt)) {
		return
	}
	if f.overrideTimer > 0 {
		f.overrideTimer -= dt
	}
	switch base {
	case ExplorationBase:
		f.moveTowardTarget(f.explorationTarget(), f.cfg.FollowSpeedExploration, dt)
	case ConversationBase:
		if f.convFraming {
			if f.moveTowardTarget(f.convTarget, f.convSpeed, dt) {
				f.convFraming = false
			}
		}
	case CombatBase:
		if f.combatFraming {
			if f.moveTowardTarget(f.combatTarget, f.combatSpeed, dt) {
				f.combatFraming = false
			}
		}
	}
}

// moveTowardTarget moves the camera focus toward target by at most speed×dt
// (constant-velocity, distance-clamped). Returns true when the focus arrived
// exactly at the target this frame.
func (f *Follower) moveTowardTarget(target matrix.Vec3, speed, dt float32) bool {
	step := speed * dt
	if math.IsNaN(float64(step)) || step <= 0 {
		return false
	}
	next := moveToward(f.cam.Focus(), target, step)
	if next == f.cam.Focus() {
		return next == target
	}
	f.cam.SetFocus(next)
	return next == target
}

// explorationTarget returns the Exploration follow target: the party centroid,
// or the selected character's position when the party spread exceeds one
// viewport half-extent (GDD CR-3, AC-3). With no party the camera holds.
func (f *Follower) explorationTarget() matrix.Vec3 {
	centroid, ok := f.partyCentroid()
	if !ok {
		return f.cam.Focus() // no party: hold position
	}
	if f.selected != 0 && f.partySpread(centroid) > f.viewportHalfWidth() {
		if pos, ok := f.pos.Position(f.selected); ok {
			return pos
		}
	}
	return centroid
}

// ── Conversation framing helpers ─────────────────────────────────────────────

// conversationPair returns the XZ side length and midpoint of the PC+NPC pair.
// ok is false when either position is unavailable.
func (f *Follower) conversationPair() (mid matrix.Vec3, side float32, ok bool) {
	pc, ok1 := f.pos.Position(f.playerCharacter)
	sp, ok2 := f.pos.Position(f.speaker)
	if !ok1 || !ok2 {
		return matrix.Vec3{}, 0, false
	}
	dx := pc.X() - sp.X()
	dz := pc.Z() - sp.Z()
	if dx < 0 {
		dx = -dx
	}
	if dz < 0 {
		dz = -dz
	}
	side = dx
	if dz > side {
		side = dz
	}
	return pc.Add(sp).Scale(0.5), side, true
}

// beginConversationFraming starts the timed pan to the PC+NPC midpoint at a
// speed that arrives in conversation_snap_duration × 2 (AC-7). Zero distance
// (already framed) does nothing.
func (f *Follower) beginConversationFraming() {
	pc, ok1 := f.pos.Position(f.playerCharacter)
	sp, ok2 := f.pos.Position(f.speaker)
	if !ok1 || !ok2 {
		return
	}
	midpoint := pc.Add(sp).Scale(0.5)
	distance := f.cam.Focus().Distance(midpoint)
	if distance <= 0 {
		return
	}
	duration := f.cfg.ConversationSnapDurationMs * 2 / 1000 // snap × 2 = 400ms
	if duration <= 0 {
		duration = 0.4
	}
	f.convTarget = midpoint
	f.convSpeed = distance / duration
	f.convFraming = true
}

// speakerOnScreen reports whether the given entity's screen position is within
// the inner 70% of the viewport (GDD CR-3, AC-5: no re-frame when inside).
func (f *Follower) speakerOnScreen(id entity.GameEntityID) bool {
	pos, ok := f.pos.Position(id)
	if !ok {
		return false
	}
	focus := f.cam.Focus()
	dx := pos.X() - focus.X()
	dy := pos.Y() - focus.Y()
	dz := pos.Z() - focus.Z()
	sx := (dx - dz) * isoCos45
	sy := (dx+dz)*isoCos45Half - dy*isoSin60
	hw := f.viewportHalfWidth()
	halfH := f.cam.CurrentHalfHeight()
	return abs32(sx) <= 0.7*hw && abs32(sy) <= 0.7*halfH
}

// ── Combat framing helpers ───────────────────────────────────────────────────

// combatBoxAll computes the XZ framing box over all tracked combatants: the
// center at the mid-elevation of the AABB and the side as the larger XZ
// extent, padded to the minimum 8×8u (AC-11). ok is false with no combatants.
func (f *Follower) combatBoxAll() (center matrix.Vec3, side float32, ok bool) {
	if len(f.combatants) == 0 {
		return matrix.Vec3{}, 0, false
	}
	var minX, minY, minZ, maxX, maxY, maxZ float32
	first := true
	for id := range f.combatants {
		pos, present := f.pos.Position(id)
		if !present {
			continue
		}
		if first {
			minX, maxX = pos.X(), pos.X()
			minY, maxY = pos.Y(), pos.Y()
			minZ, maxZ = pos.Z(), pos.Z()
			first = false
			continue
		}
		if pos.X() < minX {
			minX = pos.X()
		}
		if pos.X() > maxX {
			maxX = pos.X()
		}
		if pos.Y() < minY {
			minY = pos.Y()
		}
		if pos.Y() > maxY {
			maxY = pos.Y()
		}
		if pos.Z() < minZ {
			minZ = pos.Z()
		}
		if pos.Z() > maxZ {
			maxZ = pos.Z()
		}
	}
	if first {
		return matrix.Vec3{}, 0, false // no readable combatant positions
	}
	side = max32(maxX-minX, maxZ-minZ)
	if side < minCombatFrameSide {
		side = minCombatFrameSide
	}
	return matrix.NewVec3((minX+maxX)/2, (minY+maxY)/2, (minZ+maxZ)/2), side, true
}

// combatBoxTrimmed computes the framing box for an encounter that does not fit
// at Far zoom: the party plus the nearest 75% of enemies by distance to the
// party centroid (GDD CR-7 rule 1, AC-10). With no party members, all
// combatants are kept. Returns ok=false only when no positions are readable.
func (f *Follower) combatBoxTrimmed() (center matrix.Vec3, side float32, ok bool) {
	pc, hasParty := f.partyCentroid()
	if !hasParty {
		return f.combatBoxAll()
	}
	// Collect enemies (combatants that are not party members) with their
	// distance to the party centroid into the reusable scratch.
	enemies := f.enemyScratch[:0]
	for id := range f.combatants {
		if _, isParty := f.party[id]; isParty {
			continue
		}
		pos, present := f.pos.Position(id)
		if !present {
			continue
		}
		enemies = append(enemies, enemyDist{id: id, dist: pos.Distance(pc)})
	}
	f.enemyScratch = enemies
	if len(enemies) == 0 {
		return f.combatBoxAll()
	}
	keep := int(math.Ceil(0.75 * float64(len(enemies))))
	if keep >= len(enemies) {
		return f.combatBoxAll()
	}
	// Selection sort by ascending distance (nearest first). The list is small
	// (≤ ~30); an in-place sort keeps the combat-start path allocation-free.
	for i := 0; i < len(enemies); i++ {
		best := i
		for j := i + 1; j < len(enemies); j++ {
			if enemies[j].dist < enemies[best].dist {
				best = j
			}
		}
		if best != i {
			enemies[i], enemies[best] = enemies[best], enemies[i]
		}
	}
	// Frame box over party ∪ nearest keep enemies.
	ids := f.boxScratch[:0]
	for id := range f.party {
		ids = append(ids, id)
	}
	for i := 0; i < keep; i++ {
		ids = append(ids, enemies[i].id)
	}
	f.boxScratch = ids
	return f.combatBoxOf(ids)
}

// combatBoxOf computes the XZ framing box (center, padded side) over the given
// entity IDs.
func (f *Follower) combatBoxOf(ids []entity.GameEntityID) (center matrix.Vec3, side float32, ok bool) {
	var minX, minY, minZ, maxX, maxY, maxZ float32
	first := true
	for _, id := range ids {
		pos, present := f.pos.Position(id)
		if !present {
			continue
		}
		if first {
			minX, maxX = pos.X(), pos.X()
			minY, maxY = pos.Y(), pos.Y()
			minZ, maxZ = pos.Z(), pos.Z()
			first = false
			continue
		}
		if pos.X() < minX {
			minX = pos.X()
		}
		if pos.X() > maxX {
			maxX = pos.X()
		}
		if pos.Y() < minY {
			minY = pos.Y()
		}
		if pos.Y() > maxY {
			maxY = pos.Y()
		}
		if pos.Z() < minZ {
			minZ = pos.Z()
		}
		if pos.Z() > maxZ {
			maxZ = pos.Z()
		}
	}
	if first {
		return matrix.Vec3{}, 0, false
	}
	side = max32(maxX-minX, maxZ-minZ)
	if side < minCombatFrameSide {
		side = minCombatFrameSide
	}
	return matrix.NewVec3((minX+maxX)/2, (minY+maxY)/2, (minZ+maxZ)/2), side, true
}

// ── Shared helpers ───────────────────────────────────────────────────────────

// partyCentroid returns the average position of all readable party members.
// ok is false when no party member positions are readable.
func (f *Follower) partyCentroid() (matrix.Vec3, bool) {
	var sum matrix.Vec3
	count := 0
	for id := range f.party {
		if pos, ok := f.pos.Position(id); ok {
			sum = sum.Add(pos)
			count++
		}
	}
	if count == 0 {
		return matrix.Vec3{}, false
	}
	return sum.Scale(1 / float32(count)), true
}

// partySpread returns the maximum distance of any party member from the given
// centroid — the party spread used for the selected-character follow rule.
func (f *Follower) partySpread(centroid matrix.Vec3) float32 {
	var maxD float32
	for id := range f.party {
		if pos, ok := f.pos.Position(id); ok {
			if d := pos.Distance(centroid); d > maxD {
				maxD = d
			}
		}
	}
	return maxD
}

// viewportHalfWidth returns the world-space horizontal half-extent of the
// viewport at the current zoom: O × AR with the aspect ratio clamped to the
// camera's minimum (GDD CR-3: "one viewport half-extent" for the party-spread
// rule).
func (f *Follower) viewportHalfWidth() float32 {
	return f.cam.CurrentHalfHeight() * f.aspect()
}

// aspect returns the current viewport aspect ratio, clamped to the camera's
// minimum aspect (16:10 per ADR-0003). A zero-sized viewport falls back to
// 16:9 so framing math never divides by zero.
func (f *Follower) aspect() float32 {
	w, h := f.cam.ViewportSize()
	if w <= 0 || h <= 0 {
		return 16.0 / 9.0
	}
	a := w / h
	if a < f.cam.Config().MinAspect {
		a = f.cam.Config().MinAspect
	}
	return a
}

// boxFits reports whether an XZ box of the given side fits within the inner
// 90% of the viewport at the given ortho half-height (GDD fit test:
// S × 1.414 ≤ 2 × O × AR × 0.9 — the horizontal constraint binds).
func (f *Follower) boxFits(side, halfHeight float32) bool {
	if side <= 0 || halfHeight <= 0 {
		return false
	}
	return side*isoSqrt2 <= 2*halfHeight*f.aspect()*followBoxMargin
}

// zoomThatFits returns the smallest zoom register (Close, then Default, then
// Far) at which the given box side fits within the inner 90% of the viewport
// (AC-4 "minimum zoom that fits"). Extra Close is never chosen — it is the
// accessibility level, not a framing register. Returns Far when even Far does
// not fit.
func (f *Follower) zoomThatFits(side float32) (ZoomLevel, bool) {
	for _, lvl := range []ZoomLevel{ZoomClose, ZoomDefault, ZoomFar} {
		if f.boxFits(side, lvl.HalfHeight) {
			return lvl, true
		}
	}
	return ZoomFar, true
}

// ── Off-screen markers (camera → HUD data contract) ─────────────────────────

// OffScreenMarkers returns the edge markers for all tracked entities currently
// outside the viewport (GDD Off-Screen Markers, AC-23): one marker per
// off-screen party member and combatant, each with the nearest screen edge and
// a clamped anchor. Markers are produced regardless of camera state, sorted by
// entity ID for determinism.
//
// Zero-allocation contract: the returned slice aliases an internal scratch
// buffer and is invalidated by the next call. The HUD must copy anything it
// retains.
func (f *Follower) OffScreenMarkers() []EdgeMarker {
	viewportW, viewportH := f.cam.ViewportSize()
	if viewportW <= 0 || viewportH <= 0 {
		return f.markerScratch[:0] // minimized window: no markers
	}
	o := f.cam.CurrentHalfHeight()
	hw := f.viewportHalfWidth()
	focus := f.cam.Focus()
	inset := f.cfg.MarkerEdgeInsetPx

	out := f.markerScratch[:0]
	for id := range f.party {
		if m, ok := f.markerFor(id, TrackedPartyMember, focus, o, hw, viewportW, viewportH, inset); ok {
			out = append(out, m)
		}
	}
	for id := range f.combatants {
		if _, isParty := f.party[id]; isParty {
			continue // party members already added above
		}
		if m, ok := f.markerFor(id, TrackedHostileCombatant, focus, o, hw, viewportW, viewportH, inset); ok {
			out = append(out, m)
		}
	}
	// Insertion sort by entity ID: deterministic output, no allocation.
	for i := 1; i < len(out); i++ {
		for j := i; j > 0 && out[j].EntityID < out[j-1].EntityID; j-- {
			out[j], out[j-1] = out[j-1], out[j]
		}
	}
	f.markerScratch = out
	return out
}

// markerFor builds the EdgeMarker for one entity. Returns ok=false when the
// entity is on screen (no marker) or its position is unreadable.
func (f *Follower) markerFor(id entity.GameEntityID, kind TrackedKind, focus matrix.Vec3, o, hw, viewportW, viewportH, inset float32) (EdgeMarker, bool) {
	pos, ok := f.pos.Position(id)
	if !ok {
		return EdgeMarker{}, false
	}
	sx, sy := projectToScreen(pos, focus, o, hw, viewportW, viewportH)
	if sx >= 0 && sx <= viewportW && sy >= 0 && sy <= viewportH {
		return EdgeMarker{}, false // on screen: no marker
	}
	edge, _ := nearestEdge(sx, sy, viewportW, viewportH)
	return EdgeMarker{
		EntityID:    id,
		DisplayName: f.pos.DisplayName(id),
		Kind:        kind,
		Edge:        edge,
		AnchorX:     clamp32(sx, inset, viewportW-inset),
		AnchorY:     clamp32(sy, inset, viewportH-inset),
		Distance:    pos.Distance(focus),
	}, true
}

// projectToScreen projects a world position onto the viewport in logical
// pixels relative to the camera focus: screen_x = (dx−dz)×cos45, screen_y =
// (dx+dz)×cos45/2 − dy×cos30, scaled by the world half-extents (horizontal
// O×AR, vertical O).
func projectToScreen(pos, focus matrix.Vec3, o, halfWidth, viewportW, viewportH float32) (float32, float32) {
	dx := pos.X() - focus.X()
	dy := pos.Y() - focus.Y()
	dz := pos.Z() - focus.Z()
	sxWorld := (dx - dz) * isoCos45
	syWorld := (dx+dz)*isoCos45Half - dy*isoSin60
	sx := (sxWorld/halfWidth)*(viewportW/2) + viewportW/2
	sy := (syWorld/o)*(viewportH/2) + viewportH/2
	return sx, sy
}

// nearestEdge returns the viewport edge nearest the given screen position and
// its pixel distance. screen-y is treated as down-from-top (window space); the
// edge distances are symmetric so the orientation does not change the result.
func nearestEdge(sx, sy, viewportW, viewportH float32) (ScreenEdge, float32) {
	dLeft := sx
	dRight := viewportW - sx
	dTop := sy
	dBottom := viewportH - sy
	best, bestD := EdgeLeft, dLeft
	if dRight < bestD {
		best, bestD = EdgeRight, dRight
	}
	if dTop < bestD {
		best, bestD = EdgeTop, dTop
	}
	if dBottom < bestD {
		best, bestD = EdgeBottom, dBottom
	}
	return best, bestD
}

// ── Move-toward ──────────────────────────────────────────────────────────────

// moveToward moves from toward target by at most maxStep, exactly per the GDD
// follow formula: offset = target − from; step = min(maxStep, |offset|);
// result = from + normalize(offset) × step. The camera reaches the target
// exactly — this is a constant-velocity move-toward with distance clamping,
// never an exponential ease. maxStep must be non-negative and finite.
func moveToward(from, target matrix.Vec3, maxStep float32) matrix.Vec3 {
	offset := target.Subtract(from)
	distance := offset.Length()
	if distance == 0 || distance <= maxStep {
		return target
	}
	return from.Add(offset.Scale(maxStep / distance))
}

// ── Small numeric helpers ────────────────────────────────────────────────────

// isoSqrt2 is √2 — the screen-width factor of the GDD combat fit test
// (S × √2 projects to the screen width of an XZ box of side S).
const isoSqrt2 = 1.4142135623730951

func abs32(v float32) float32 {
	if v < 0 {
		return -v
	}
	return v
}

func max32(a, b float32) float32 {
	if a > b {
		return a
	}
	return b
}

func clamp32(v, lo, hi float32) float32 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
