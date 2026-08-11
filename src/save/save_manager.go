// Package save implements the Save/Load orchestration (Story SAVE-001,
// TR-save-001, TR-save-002, ADR-0002): SaveManagerImpl wraps the Entity
// Framework's full-state serialization (entity.EntityStore) with the versioned
// .sav file format — a fixed 16-byte header (format version, CRC32 checksum,
// timestamp), a pod-encoded metadata block, and the raw EntityStore snapshot
// blob.
//
// Control manifest rules honored here (Foundation Layer, ADR-0002):
//   - Full-state pod serialization — never JSON/text, never delta saves
//   - Save versioning with registered migration functions per transition
//   - CRC32 checksum in the header (detection; .bak restore is SAVE-003)
//
// Save file format (ADR-0002), all fields little-endian (matching klib's
// binary convention used by pod):
//
//	[Header: 16 bytes]
//	  Version    uint32 (4 bytes) — save format version
//	  Checksum   uint32 (4 bytes) — CRC32 of everything after the header
//	  Timestamp  int64  (8 bytes) — unix nanoseconds at save time
//	[Metadata: variable — pod-encoded SaveMetadata]
//	[Entity Data: variable — raw blob from EntityStore.Serialize()]
//
// The metadata block is self-delimiting (pod writes length-prefixed type and
// field lookup tables before the value), so after decoding the metadata the
// remaining bytes of the payload are exactly the entity data blob — no
// metadata-length field needed.
package save

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"hash/crc32"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"kaijuengine.com/engine/encoding/pod"

	"marrow-of-empire/entity"
)

const (
	// ManualSlotCount is the number of manual save slots: 0 through
	// ManualSlotCount-1 (AC-4, ADR-0002: 20 manual slots).
	ManualSlotCount = 20
	// AutoSlotID is the reserved slot for the auto-save (AC-4, ADR-0002).
	AutoSlotID = -1
	// QuicksaveSlotID is the reserved slot for the F5 quicksave (AC-4,
	// ADR-0002).
	QuicksaveSlotID = -2
	// DefaultSaveVersion is the save format version when
	// Config.CurrentVersion is zero (first implementation, ADR-0002
	// migration plan).
	DefaultSaveVersion uint32 = 1
	// saveHeaderSize is the fixed header size in bytes (ADR-0002: 4 + 4 + 8).
	saveHeaderSize = 16
)

var (
	// ErrSlotOutOfRange is returned when a slot ID is outside the managed
	// set: manual slots 0..ManualSlotCount-1, AutoSlotID, or QuicksaveSlotID
	// (AC-4: the slot set is never exceeded).
	ErrSlotOutOfRange = errors.New("save: slot id out of range")
	// ErrSaveNotFound is returned by Load/Quickload when no save file exists
	// for the slot. F9 with no quicksave is a no-op error, never a crash
	// (AC-6 edge case).
	ErrSaveNotFound = errors.New("save: no save file for slot")
	// ErrChecksumMismatch is returned when the CRC32 checksum does not match
	// the payload — the save file is corrupted or was edited (ADR-0002
	// checksum detection; .bak restore is SAVE-003, out of scope).
	ErrChecksumMismatch = errors.New("save: checksum mismatch, save file is corrupted")
	// ErrUnsupportedVersion is returned when the save file version is newer
	// than the running version (an older build cannot load a newer save) or
	// when no registered migration chain leads from the save version to the
	// current version (TR-save-002).
	ErrUnsupportedVersion = errors.New("save: unsupported save file version")
)

// SaveMetadata is the player-facing information attached to a save (ADR-0002).
// It is pod-encoded via an internal encodable layout (saveMetadataLayout):
// pod cannot encode time.Time (unexported fields) or time.Duration
// (unregistered type), so the layout mirrors this struct with registered
// primitive fields and Save/Load convert between the two.
//
// Example usage:
//
//	meta := save.SaveMetadata{
//	    CharacterName: "Kael",
//	    PlayTime:      4 * time.Hour,
//	    RegionID:      "thornwell",
//	    Thumbnail:     screenshotJPEG, // 320px
//	}
//	if err := manager.Save(0, meta); err != nil { /* handle */ }
type SaveMetadata struct {
	// CharacterName is the player character's name, shown in the save list.
	CharacterName string
	// PlayTime is the total playtime at save time.
	PlayTime time.Duration
	// RegionID identifies the current region, e.g. "thornwell".
	RegionID string
	// Thumbnail is a 320px JPEG screenshot of the save moment.
	Thumbnail []byte
	// Timestamp is the wall-clock time of the save. Save fills it with
	// time.Now() when the caller passes the zero value.
	Timestamp time.Time
	// Version is the save format version. Save stamps it with the manager's
	// current version; Load reports the effective (post-migration) version.
	Version uint32
}

// SaveSlot describes one save slot for the save list UI (ADR-0002 CR-2).
type SaveSlot struct {
	// SlotID is the slot the save occupies: a manual slot 0..ManualSlotCount-1,
	// AutoSlotID, or QuicksaveSlotID.
	SlotID int
	// Metadata is the decoded save metadata. Zero-valued when the file cannot
	// be decoded: the slot still appears in the list so the player can delete
	// it (corrupt saves are visible, not hidden).
	Metadata SaveMetadata
	// FilePath is the path of the .sav file.
	FilePath string
}

// MigrationFunc transforms the payload of a save file — everything after the
// 16-byte header — from one format version to the next (TR-save-002). It must
// return the transformed payload; the manager re-stamps the header version
// after each migration step.
//
// Example usage:
//
//	manager.RegisterMigration(1, 2, func(payload []byte) ([]byte, error) {
//	    // v2 renamed the "thornwell" region id to "region_thornwell".
//	    return bytes.Replace(payload, []byte("thornwell"), []byte("region_thornwell"), -1), nil
//	})
type MigrationFunc func(payload []byte) ([]byte, error)

// SaveManager orchestrates save/load for the game (ADR-0002). It wraps an
// entity.EntityStore — the full-state serialization boundary — with the
// versioned .sav file format: header, metadata, and the raw store snapshot.
// See SaveManagerImpl for the concrete implementation.
type SaveManager interface {
	// Save serializes the wrapped store and writes it to the slot with the
	// given metadata. Valid slots: 0..ManualSlotCount-1 plus AutoSlotID and
	// QuicksaveSlotID; any other ID returns ErrSlotOutOfRange. Subscribers of
	// SubscribeSaveCompleted are notified after a successful write.
	Save(slotID int, metadata SaveMetadata) error

	// Load reads the slot, verifies the CRC32 checksum and version
	// compatibility, runs registered migrations when the save predates the
	// current version, and restores the store via EntityStore.Deserialize.
	// Returns the save's metadata, or ErrSaveNotFound when the slot is empty.
	Load(slotID int) (*SaveMetadata, error)

	// ListSaves returns every existing save slot in slot ID order. Files that
	// cannot be decoded are listed with zero-valued metadata.
	ListSaves() []SaveSlot

	// DeleteSave removes the slot's save file. Deleting an empty slot is a
	// no-op (nil error), so the UI can delete unconditionally.
	DeleteSave(slotID int) error

	// Quicksave writes to QuicksaveSlotID (F5). It returns promptly — the
	// caller (input system) owns key handling; the save-completed subscriber
	// receives the "Game Saved" signal (AC-5).
	Quicksave(metadata SaveMetadata) error

	// Quickload restores the quicksave slot (F9). Returns ErrSaveNotFound —
	// never a crash — when no quicksave exists (AC-6).
	Quickload() (*SaveMetadata, error)
}

// Config carries the SaveManager tuning knobs. Zero values fall back to the
// defaults (DefaultSaveDir, DefaultSaveVersion).
type Config struct {
	// SaveDir is the base directory for .sav files; it is created on the
	// first save. Default: the platform-standard directory resolved by
	// DefaultSaveDir (TR-save-004 — Linux: $XDG_DATA_HOME, Windows:
	// %APPDATA%, macOS: ~/Library/Application Support).
	SaveDir string
	// CurrentVersion is the save format version written by Save and migrated
	// to on Load. Default: 1 (DefaultSaveVersion).
	CurrentVersion uint32
}

// SaveCompletedFunc is notified after every successful save — manual, auto,
// or quicksave — with the slot ID and the metadata as written. The UI layer
// subscribes for the "Game Saved" indicator (AC-5); the 1.5-second display
// timer belongs to the UI, not to this package.
type SaveCompletedFunc func(slotID int, metadata SaveMetadata)

// SaveManagerImpl is the concrete SaveManager (ADR-0002).
//
// The game drives saves from a single game thread, so concurrent Save/Load
// calls on the same manager are not supported (the wrapped EntityStore is not
// safe for concurrent serialization). RegisterMigration and
// SubscribeSaveCompleted are safe to call concurrently with Loads.
type SaveManagerImpl struct {
	store   entity.EntityStore
	dir     string
	version uint32

	migrationsMu sync.RWMutex
	// migrations maps fromVersion -> toVersion -> migration function
	// (TR-save-002: one function per version transition).
	migrations map[uint32]map[uint32]MigrationFunc

	subscribersMu sync.Mutex
	subscribers   []SaveCompletedFunc
}

// Compile-time assertion: SaveManagerImpl satisfies SaveManager.
var _ SaveManager = (*SaveManagerImpl)(nil)

// NewSaveManager creates the SaveManagerImpl wrapping the store (dependency
// injection — no singletons). Config zero values fall back to the defaults.
//
// Example usage:
//
//	manager := save.NewSaveManager(store, save.Config{})
//	if err := manager.Save(0, save.SaveMetadata{CharacterName: "Kael"}); err != nil {
//	    /* handle */
//	}
func NewSaveManager(store entity.EntityStore, config Config) *SaveManagerImpl {
	if config.SaveDir == "" {
		config.SaveDir = DefaultSaveDir()
	}
	if config.CurrentVersion == 0 {
		config.CurrentVersion = DefaultSaveVersion
	}
	return &SaveManagerImpl{
		store:      store,
		dir:        config.SaveDir,
		version:    config.CurrentVersion,
		migrations: make(map[uint32]map[uint32]MigrationFunc),
	}
}

// saveHeader is the fixed 16-byte header of a .sav file (ADR-0002). All
// fields are little-endian, matching klib's binary convention used by pod.
type saveHeader struct {
	// Version is the save format version. The header carries it so version
	// checks and migrations run before any decoding.
	Version uint32
	// Checksum is the CRC32 (IEEE) of everything after the header.
	Checksum uint32
	// Timestamp is the unix nanosecond time of the save.
	Timestamp int64
}

// saveMetadataLayout is the pod-encodable form of SaveMetadata. Pod cannot
// encode time.Time (unexported fields) or time.Duration (unregistered type),
// so this layout mirrors the public struct with registered primitive fields —
// the same pattern as entityRecord in the entity framework's serialization.
type saveMetadataLayout struct {
	CharacterName string
	PlayTimeNanos int64
	RegionID      string
	Thumbnail     []byte
	TimestampUnix int64
	Version       uint32
}

func init() {
	// The metadata layout must be registered with pod or encoding fails
	// ("never registered with pod"). A registration collision is warned and
	// skipped: the type is registered either way.
	if err := pod.Register(saveMetadataLayout{}); err != nil {
		slog.Warn("pod registration skipped",
			"type", pod.QualifiedNameForLayout(saveMetadataLayout{}), "err", err)
	}
}

// Save implements SaveManager. See the interface doc for the contract.
func (m *SaveManagerImpl) Save(slotID int, metadata SaveMetadata) error {
	if err := m.validateSlot(slotID); err != nil {
		return err
	}

	// Full-state serialization: one pod snapshot of the entity table and all
	// component stores (TR-save-001, ADR-0002 — never JSON, never delta).
	blob, err := m.store.Serialize()
	if err != nil {
		return fmt.Errorf("save: serialize entity state: %w", err)
	}

	now := time.Now()
	if metadata.Timestamp.IsZero() {
		metadata.Timestamp = now
	}
	metadata.Version = m.version

	metaBlob, err := encodeMetadata(metadata)
	if err != nil {
		return err
	}
	payload := make([]byte, 0, len(metaBlob)+len(blob))
	payload = append(payload, metaBlob...)
	payload = append(payload, blob...)

	data := encodeHeader(saveHeader{
		Version:   m.version,
		Checksum:  crc32.ChecksumIEEE(payload),
		Timestamp: now.UnixNano(),
	})
	data = append(data, payload...)

	if err := os.MkdirAll(m.dir, 0o755); err != nil {
		return fmt.Errorf("save: create save directory %q: %w", m.dir, err)
	}
	path := m.savePath(slotID)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("save: write %q: %w", path, err)
	}
	m.notifySaveCompleted(slotID, metadata)
	return nil
}

// Load implements SaveManager. See the interface doc for the contract.
func (m *SaveManagerImpl) Load(slotID int) (*SaveMetadata, error) {
	if err := m.validateSlot(slotID); err != nil {
		return nil, err
	}
	path := m.savePath(slotID)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("%w: %s", ErrSaveNotFound, path)
		}
		return nil, fmt.Errorf("save: read %q: %w", path, err)
	}
	header, payload, err := m.validateFileHeader(data, path)
	if err != nil {
		return nil, err
	}

	effectiveVersion := header.Version
	if header.Version < m.version {
		payload, effectiveVersion, err = m.migrate(payload, header.Version)
		if err != nil {
			return nil, err
		}
	}

	meta, entityData, err := decodePayload(payload)
	if err != nil {
		return nil, fmt.Errorf("save: decode %q: %w", path, err)
	}
	// The header version is authoritative; report the effective version so
	// callers never observe a stale (pre-migration) version.
	meta.Version = effectiveVersion

	if err := m.store.Deserialize(entityData); err != nil {
		return nil, fmt.Errorf("save: restore entity state from %q: %w", path, err)
	}
	return &meta, nil
}

// ListSaves implements SaveManager. See the interface doc for the contract.
// The save directory is scanned for files named save_{slotID}.sav; foreign
// files are ignored. Decoding failures leave the slot's metadata zero-valued
// so corrupt saves stay visible (and deletable) in the UI.
func (m *SaveManagerImpl) ListSaves() []SaveSlot {
	entries, err := os.ReadDir(m.dir)
	if err != nil {
		// A missing directory simply means no saves yet.
		return nil
	}
	slots := make([]SaveSlot, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		slotID, ok := parseSaveFileName(entry.Name())
		if !ok {
			continue
		}
		path := filepath.Join(m.dir, entry.Name())
		slot := SaveSlot{SlotID: slotID, FilePath: path}
		if data, err := os.ReadFile(path); err == nil && len(data) >= saveHeaderSize {
			if meta, _, err := decodePayload(data[saveHeaderSize:]); err == nil {
				slot.Metadata = meta
			}
		}
		slots = append(slots, slot)
	}
	sort.Slice(slots, func(i, j int) bool { return slots[i].SlotID < slots[j].SlotID })
	return slots
}

// DeleteSave implements SaveManager. See the interface doc for the contract.
// Deleting an empty slot is a no-op so the UI can delete unconditionally.
func (m *SaveManagerImpl) DeleteSave(slotID int) error {
	if err := m.validateSlot(slotID); err != nil {
		return err
	}
	path := m.savePath(slotID)
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("save: delete %q: %w", path, err)
	}
	return nil
}

// Quicksave implements SaveManager. See the interface doc for the contract.
func (m *SaveManagerImpl) Quicksave(metadata SaveMetadata) error {
	return m.Save(QuicksaveSlotID, metadata)
}

// Quickload implements SaveManager. See the interface doc for the contract.
func (m *SaveManagerImpl) Quickload() (*SaveMetadata, error) {
	return m.Load(QuicksaveSlotID)
}

// RegisterMigration registers a migration function for a version transition
// (TR-save-002, AC-3). Load runs migrations in a chain from the save's
// version to the current version, so every step on the path must be
// registered. Registration is read-write locked against concurrent Loads;
// the game registers all migrations at startup, before any load.
//
// Example usage:
//
//	manager.RegisterMigration(1, 2, func(payload []byte) ([]byte, error) {
//		return transformPayload(payload), nil
//	})
func (m *SaveManagerImpl) RegisterMigration(fromVersion, toVersion uint32, fn MigrationFunc) {
	if fn == nil || fromVersion >= toVersion {
		return
	}
	m.migrationsMu.Lock()
	defer m.migrationsMu.Unlock()
	steps := m.migrations[fromVersion]
	if steps == nil {
		steps = make(map[uint32]MigrationFunc)
		m.migrations[fromVersion] = steps
	}
	steps[toVersion] = fn
}

// SubscribeSaveCompleted registers a callback invoked synchronously after
// every successful save — manual, auto, or quicksave — with the slot ID and
// the metadata as written. The returned function unsubscribes. The UI layer
// uses this for the "Game Saved" indicator (AC-5); the 1.5-second display
// timer belongs to the UI.
//
// Example usage:
//
//	unsubscribe := manager.SubscribeSaveCompleted(func(slotID int, meta save.SaveMetadata) {
//		hud.ShowGameSavedIndicator() // hides itself after 1.5s
//	})
//	defer unsubscribe()
func (m *SaveManagerImpl) SubscribeSaveCompleted(fn SaveCompletedFunc) func() {
	m.subscribersMu.Lock()
	defer m.subscribersMu.Unlock()
	m.subscribers = append(m.subscribers, fn)
	index := len(m.subscribers) - 1
	return func() {
		m.subscribersMu.Lock()
		defer m.subscribersMu.Unlock()
		m.subscribers[index] = nil
	}
}

// migrate walks the registered migration chain from the save's version to the
// current version, applying each transition to the payload. Each step prefers
// the +1 transition; if none is registered, the smallest registered
// destination above the current step is used (deterministic). A missing chain
// returns ErrUnsupportedVersion and the load fails — a save is never loaded
// at the wrong version (TR-save-002).
func (m *SaveManagerImpl) migrate(payload []byte, fromVersion uint32) ([]byte, uint32, error) {
	version := fromVersion
	for version < m.version {
		next, fn, ok := m.nextMigration(version)
		if !ok {
			return nil, 0, fmt.Errorf("%w: no migration path from version %d to %d", ErrUnsupportedVersion, fromVersion, m.version)
		}
		transformed, err := fn(payload)
		if err != nil {
			return nil, 0, fmt.Errorf("save: migration %d -> %d failed: %w", version, next, err)
		}
		payload = transformed
		version = next
	}
	return payload, version, nil
}

// nextMigration picks the next migration step from the given version: the +1
// transition when registered, otherwise the smallest registered destination
// above it. Returns ok=false when no step leads upward.
func (m *SaveManagerImpl) nextMigration(version uint32) (uint32, MigrationFunc, bool) {
	m.migrationsMu.RLock()
	defer m.migrationsMu.RUnlock()
	steps := m.migrations[version]
	if len(steps) == 0 {
		return 0, nil, false
	}
	if fn, ok := steps[version+1]; ok {
		return version + 1, fn, true
	}
	var (
		best   uint32
		bestFn MigrationFunc
		found  bool
	)
	for next, fn := range steps {
		if next > version && (!found || next < best) {
			best, bestFn, found = next, fn, true
		}
	}
	return best, bestFn, found
}

// validateSlot rejects slot IDs outside the managed set: manual slots
// 0..ManualSlotCount-1 plus the two reserved slots (AC-4: the 20 + 1 + 1 set
// is never exceeded).
func (m *SaveManagerImpl) validateSlot(slotID int) error {
	switch {
	case slotID >= 0 && slotID < ManualSlotCount:
		return nil
	case slotID == AutoSlotID || slotID == QuicksaveSlotID:
		return nil
	default:
		return fmt.Errorf("%w: %d", ErrSlotOutOfRange, slotID)
	}
}

// savePath returns the .sav file path for a slot (ADR-0002: save_{slotID}.sav).
func (m *SaveManagerImpl) savePath(slotID int) string {
	return filepath.Join(m.dir, fmt.Sprintf("save_%d.sav", slotID))
}

// notifySaveCompleted synchronously dispatches the save-completed signal to
// all subscribers (ADR-0011: synchronous dispatch preserves ordering).
func (m *SaveManagerImpl) notifySaveCompleted(slotID int, metadata SaveMetadata) {
	m.subscribersMu.Lock()
	subs := make([]SaveCompletedFunc, len(m.subscribers))
	copy(subs, m.subscribers)
	m.subscribersMu.Unlock()
	for _, fn := range subs {
		if fn != nil {
			fn(slotID, metadata)
		}
	}
}

// encodeMetadata pod-encodes SaveMetadata via its encodable layout.
func encodeMetadata(metadata SaveMetadata) ([]byte, error) {
	layout := saveMetadataLayout{
		CharacterName: metadata.CharacterName,
		PlayTimeNanos: int64(metadata.PlayTime),
		RegionID:      metadata.RegionID,
		Thumbnail:     metadata.Thumbnail,
		TimestampUnix: metadata.Timestamp.UnixNano(),
		Version:       metadata.Version,
	}
	var buf bytes.Buffer
	if err := pod.NewEncoder(&buf).Encode(layout); err != nil {
		return nil, fmt.Errorf("save: encode metadata: %w", err)
	}
	return buf.Bytes(), nil
}

// decodePayload splits a save payload (everything after the header) into the
// pod-encoded metadata block and the entity data blob. The pod blob is
// self-delimiting: decoding consumes exactly the metadata bytes, so the
// remaining reader bytes are the raw EntityStore snapshot (ADR-0002 framing).
func decodePayload(payload []byte) (SaveMetadata, []byte, error) {
	reader := bytes.NewReader(payload)
	var layout saveMetadataLayout
	if err := pod.NewDecoder(reader).Decode(&layout); err != nil {
		return SaveMetadata{}, nil, err
	}
	entityData := make([]byte, reader.Len())
	if _, err := io.ReadFull(reader, entityData); err != nil {
		return SaveMetadata{}, nil, err
	}
	return layoutToMetadata(layout), entityData, nil
}

// layoutToMetadata converts the encodable layout back into the public
// SaveMetadata. A zero timestamp (never stamped) converts to the zero
// time.Time.
func layoutToMetadata(l saveMetadataLayout) SaveMetadata {
	var ts time.Time
	if l.TimestampUnix != 0 {
		ts = time.Unix(0, l.TimestampUnix)
	}
	return SaveMetadata{
		CharacterName: l.CharacterName,
		PlayTime:      time.Duration(l.PlayTimeNanos),
		RegionID:      l.RegionID,
		Thumbnail:     l.Thumbnail,
		Timestamp:     ts,
		Version:       l.Version,
	}
}

// parseSaveFileName extracts the slot ID from a file name of the form
// "save_{slotID}.sav" (negative IDs parse fine: "save_-1.sav" -> -1). Returns
// ok=false for anything else, so foreign files in the save directory are
// ignored by ListSaves.
func parseSaveFileName(name string) (int, bool) {
	const prefix = "save_"
	const suffix = ".sav"
	if !strings.HasPrefix(name, prefix) || !strings.HasSuffix(name, suffix) {
		return 0, false
	}
	id, err := strconv.Atoi(strings.TrimSuffix(strings.TrimPrefix(name, prefix), suffix))
	if err != nil {
		return 0, false
	}
	return id, true
}

// encodeHeader serializes the 16-byte header, little-endian (matching klib's
// binary convention used by pod).
func encodeHeader(h saveHeader) []byte {
	b := make([]byte, saveHeaderSize)
	binary.LittleEndian.PutUint32(b[0:4], h.Version)
	binary.LittleEndian.PutUint32(b[4:8], h.Checksum)
	binary.LittleEndian.PutUint64(b[8:16], uint64(h.Timestamp))
	return b
}

// decodeHeader parses the 16-byte header, little-endian.
func decodeHeader(b []byte) saveHeader {
	return saveHeader{
		Version:   binary.LittleEndian.Uint32(b[0:4]),
		Checksum:  binary.LittleEndian.Uint32(b[4:8]),
		Timestamp: int64(binary.LittleEndian.Uint64(b[8:16])),
	}
}

// validateFileHeader checks the minimum file size, decodes the 16-byte header,
// rejects saves from a newer build, and verifies the CRC32 checksum (before
// any migration code touches the payload — ADR-0002). It returns the decoded
// header and the payload (everything after the header) on success.
func (m *SaveManagerImpl) validateFileHeader(data []byte, path string) (saveHeader, []byte, error) {
	if len(data) < saveHeaderSize {
		return saveHeader{}, nil, fmt.Errorf("save: %q too short for a header (%d bytes)", path, len(data))
	}
	header := decodeHeader(data[:saveHeaderSize])
	payload := data[saveHeaderSize:]

	if header.Version > m.version {
		// Forward compatibility means newer builds load older saves — the
		// reverse direction is rejected.
		return saveHeader{}, nil, fmt.Errorf("%w: file version %d is newer than the running version %d", ErrUnsupportedVersion, header.Version, m.version)
	}
	// Checksum verification precedes migrations: a corrupted file is rejected
	// before any migration code touches its payload (.bak restore is SAVE-003).
	if crc32.ChecksumIEEE(payload) != header.Checksum {
		return saveHeader{}, nil, fmt.Errorf("%w: %s", ErrChecksumMismatch, path)
	}
	return header, payload, nil
}
