// settings_cfg.go implements the real settings.cfg file adapter (TR-menu-002,
// ADR-0002): per-user JSON persistence at an injectable path. The main-menu
// screens (Story 005 SettingsScreen, Story 007 KeybindingsScreen) expose
// reader/writer/has-file seams; this adapter wires those seams to disk.
//
// Contract (Story 008 AC-2):
//   - Dotted keys map onto nested JSON ("camera.edge_scroll_speed" ->
//     {"camera": {"edge_scroll_speed": "20"}}); flat keys (Story 003's
//     "camera_motion", "subtitles") stay top-level.
//   - Values are stored as JSON strings for lossless round-trips; reading
//     accepts strings, numbers, and booleans (hand-edited files keep
//     working).
//   - A missing file yields an empty store (defaults) with no error; a
//     corrupt (unparsable) file yields an empty store plus a logged error —
//     never a dialog (AC-2: "missing/corrupt file → defaults, no error
//     dialog"). Partially invalid values (objects/arrays under a key) read
//     as absent; other categories are unaffected.
//   - Writes are explicit Flush() calls (the screens' debounce clock drives
//     when); the game layer flushes on clean shutdown via host.OnClose.
//
// No goroutines: the adapter is owned by the game layer and only touched
// from the main thread, matching the screen controllers' goroutine-free
// debounce contract.
package game

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// SettingsCfg is the on-disk settings store. The zero value is not usable;
// construct with NewSettingsCfg.
type SettingsCfg struct {
	path string // settings.cfg file path (injectable for tests)
	data map[string]any
	ok   bool // file exists AND parsed cleanly (HasFile semantics)
}

// NewSettingsCfg creates the adapter for the given settings.cfg path. No
// file I/O happens here; call Load (or rely on the lazy read in HasFile).
func NewSettingsCfg(path string) *SettingsCfg {
	return &SettingsCfg{path: path}
}

// DefaultSettingsPath returns the platform-standard per-user settings.cfg
// path (Linux: $XDG_CONFIG_HOME/marrow-of-empire/settings.cfg). Follows the
// ADR-0002 "per-user, not per-save" rule; a user-config dir is used because
// settings are configuration, not save data.
func DefaultSettingsPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "marrow-of-empire", "settings.cfg"), nil
}

// Path returns the configured file path (diagnostics/tests).
func (s *SettingsCfg) Path() string { return s.path }

// Load reads and decodes the file. A missing file returns nil (empty store —
// defaults apply). A corrupt file returns an error for logging while leaving
// the store empty (defaults) — the caller must NOT surface a dialog (AC-2).
func (s *SettingsCfg) Load() error {
	raw, err := os.ReadFile(s.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			s.data = map[string]any{}
			s.ok = false
			return nil
		}
		// Unreadable (permissions, directory): same contract as missing —
		// defaults, log, no dialog.
		slog.Warn("settings.cfg: unreadable, using defaults", "path", s.path, "error", err)
		s.data = map[string]any{}
		s.ok = false
		return nil
	}
	var parsed map[string]any
	if err := json.Unmarshal(raw, &parsed); err != nil {
		slog.Warn("settings.cfg: corrupt, using defaults", "path", s.path, "error", err)
		s.data = map[string]any{}
		s.ok = false
		return nil
	}
	if parsed == nil {
		parsed = map[string]any{}
	}
	s.data = parsed
	s.ok = true
	return nil
}

// HasFile implements the screens' HasSettingsFile probe: true only when a
// valid (parsed) settings.cfg exists. A missing or corrupt file reports
// false so the first-launch defaults path runs (AC-2/AC-5).
func (s *SettingsCfg) HasFile() bool {
	if s.data == nil {
		if err := s.Load(); err != nil {
			slog.Warn("settings.cfg: load failed during HasFile", "error", err)
		}
	}
	return s.ok && len(s.data) > 0
}

// Get implements the screens' reader seam: the stored value for a dotted
// key, or "" when absent or not a scalar. A partial settings.cfg is valid —
// missing or unparsable keys keep their defaults (Story 005 contract).
func (s *SettingsCfg) Get(key string) string {
	if s.data == nil {
		if err := s.Load(); err != nil {
			return ""
		}
	}
	val, ok := lookup(s.data, key)
	if !ok {
		return ""
	}
	return scalarString(val)
}

// Set stores a value under the dotted key (in memory only; call Flush to
// write). Mirrors the screens' formatted-value contract: values are strings.
func (s *SettingsCfg) Set(key, value string) {
	if s.data == nil {
		s.data = map[string]any{}
		s.ok = true // a Set implies intent to persist; HasFile may follow
	}
	assign(s.data, key, value)
}

// SetAndFlush stores the value and writes the file immediately. This is the
// screens' writer-seam binding: the screens call it per dirty key when their
// debounce flush fires (or on clean shutdown). Returns the write error so
// the screens' retry state machine keeps the key dirty on failure.
func (s *SettingsCfg) SetAndFlush(key, value string) error {
	s.Set(key, value)
	return s.Flush()
}

// Flush writes the store to disk, creating parent directories. Returns the
// write error (the screens' write seam retries on failure); a nil store is
// a no-op.
func (s *SettingsCfg) Flush() error {
	if s.data == nil {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return fmt.Errorf("settings.cfg: create dir: %w", err)
	}
	raw, err := json.MarshalIndent(s.data, "", "  ")
	if err != nil {
		return fmt.Errorf("settings.cfg: encode: %w", err)
	}
	if err := os.WriteFile(s.path, raw, 0o644); err != nil {
		return fmt.Errorf("settings.cfg: write: %w", err)
	}
	s.ok = true
	return nil
}

// lookup walks a dotted key through the nested map. Flat keys (no dot) are
// top-level. Returns (value, found).
func lookup(m map[string]any, key string) (any, bool) {
	parts := strings.Split(key, ".")
	var cur any = m
	for i, part := range parts {
		node, ok := cur.(map[string]any)
		if !ok {
			return nil, false
		}
		v, ok := node[part]
		if !ok {
			return nil, false
		}
		if i == len(parts)-1 {
			return v, true
		}
		cur = v
	}
	return nil, false
}

// assign writes a value at the dotted key, creating intermediate maps.
func assign(m map[string]any, key string, value any) {
	parts := strings.Split(key, ".")
	cur := m
	for i, part := range parts {
		if i == len(parts)-1 {
			cur[part] = value
			return
		}
		next, ok := cur[part].(map[string]any)
		if !ok {
			next = map[string]any{}
			cur[part] = next
		}
		cur = next
	}
}

// scalarString renders a JSON scalar as the screens' string contract:
// strings as-is, numbers and booleans via their canonical Go forms.
// Objects/arrays (not a valid setting value) report "" so the key reads as
// absent and the default applies.
func scalarString(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case float64:
		return strconv.FormatFloat(t, 'f', -1, 64)
	case bool:
		return strconv.FormatBool(t)
	case json.Number:
		return t.String()
	}
	return ""
}
