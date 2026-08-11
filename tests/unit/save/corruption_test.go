// Unit tests for Story SAVE-003: Corruption recovery (TR-save-005, ADR-0002).
// WriteWithBackup frames .sav files with the ADR-0002 16-byte header and
// preserves the previous save as .bak before overwriting; ReadWithRecovery
// verifies the CRC32 checksum and falls back to the .bak when the .sav is
// corrupted. Each test writes into its own t.TempDir — no ambient filesystem
// state, no execution-order dependence, deterministic.
package save_test

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"marrow-of-empire/save"
)

// saveHeaderSize mirrors ADR-0002's fixed 16-byte header (4 version + 4
// checksum + 8 timestamp, little-endian). Payload bytes live after it; the
// tests corrupt payload bytes so the header checksum goes stale.
const saveHeaderSize = 16

// writeTwice writes two generations of a save: the second WriteWithBackup
// call preserves the first generation as .bak (GDD: "Before overwriting a
// save, the previous version is kept as save_[slot_id].bak").
func writeTwice(t *testing.T, path string, first, second []byte) {
	t.Helper()
	if err := save.WriteWithBackup(path, first); err != nil {
		t.Fatalf("WriteWithBackup(first): %v", err)
	}
	if err := save.WriteWithBackup(path, second); err != nil {
		t.Fatalf("WriteWithBackup(second): %v", err)
	}
}

// corruptPayload flips one byte inside the data portion of a framed .sav file
// — everything after the 16-byte header — leaving the header checksum stale,
// the corruption a faulty disk or an edited save produces.
func corruptPayload(t *testing.T, path string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%s): %v", path, err)
	}
	if len(data) <= saveHeaderSize {
		t.Fatalf("cannot corrupt %s: only %d bytes (need > %d)", path, len(data), saveHeaderSize)
	}
	data[saveHeaderSize] ^= 0xFF
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("WriteFile(%s): %v", path, err)
	}
}

// bakPath derives the .bak path the way the save system names it (GDD:
// save_[slot_id].bak — the .sav extension is replaced).
func bakPath(path string) string {
	return strings.TrimSuffix(path, ".sav") + ".bak"
}

// GIVEN a valid save, WHEN it is written and read back, THEN the payload is
// byte-identical — and a first save creates no .bak (nothing was overwritten,
// so nothing needed backing up).
func TestWriteReadRoundTrip_ReturnsOriginalData(t *testing.T) {
	path := filepath.Join(t.TempDir(), "save_0.sav")
	payload := []byte("the marrow remembers its roots")

	if err := save.WriteWithBackup(path, payload); err != nil {
		t.Fatalf("WriteWithBackup: %v", err)
	}
	got, err := save.ReadWithRecovery(path)
	if err != nil {
		t.Fatalf("ReadWithRecovery: %v", err)
	}
	if string(got) != string(payload) {
		t.Errorf("round-trip payload = %q, want %q", got, payload)
	}
	if _, err := os.Stat(bakPath(path)); !os.IsNotExist(err) {
		t.Errorf(".bak exists after first save, want none (err = %v)", err)
	}
}

// AC-1: GIVEN a corrupted .sav with an intact .bak, WHEN the save is read,
// THEN the payload comes back from .bak automatically — the last successful
// save — and the corrupted .sav is left untouched (the layer reads, it does
// not repair; promoting .bak back to .sav is the orchestrator's decision).
func TestCorruptedSav_RecoversFromBak(t *testing.T) {
	path := filepath.Join(t.TempDir(), "save_1.sav")
	first := []byte("last successful save")
	second := []byte("this save will be corrupted")
	writeTwice(t, path, first, second)

	corruptPayload(t, path)

	got, err := save.ReadWithRecovery(path)
	if err != nil {
		t.Fatalf("ReadWithRecovery: %v, want recovery from .bak", err)
	}
	if string(got) != string(first) {
		t.Errorf("recovered payload = %q, want %q (the .bak: last successful save)", got, first)
	}
	// Read-only recovery: the .sav file still holds the corrupted bytes.
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(.sav): %v", err)
	}
	if data[saveHeaderSize] == second[0] {
		t.Error(".sav payload byte was repaired by ReadWithRecovery — recovery must be read-only")
	}
}

// AC-2: GIVEN both .sav and .bak corrupted, WHEN the save is read, THEN
// ErrNoValidSaveFile is returned — never silent data loss, never a panic.
func TestBothSavAndBakCorrupted_ReturnsError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "save_2.sav")
	writeTwice(t, path, []byte("first"), []byte("second"))
	corruptPayload(t, path)
	corruptPayload(t, bakPath(path))

	_, err := save.ReadWithRecovery(path)
	if !errors.Is(err, save.ErrNoValidSaveFile) {
		t.Errorf("ReadWithRecovery error = %v, want ErrNoValidSaveFile", err)
	}
}

// AC-4: GIVEN no .sav file at all, WHEN the save is read, THEN
// ErrSaveNotFound is returned — a missing save is not corruption, so the .bak
// alone is never treated as a save, and the call never panics.
func TestMissingSav_ReturnsErrSaveNotFound(t *testing.T) {
	path := filepath.Join(t.TempDir(), "save_3.sav")

	_, err := save.ReadWithRecovery(path)
	if !errors.Is(err, save.ErrSaveNotFound) {
		t.Errorf("ReadWithRecovery error = %v, want ErrSaveNotFound", err)
	}
}

// Implementation note (SAVE-003): GIVEN a .sav truncated by a crash mid-write
// — smaller than the 16-byte header, so clearly incomplete — WHEN the save is
// read, THEN the .bak is restored automatically, with no prompt.
func TestTruncatedSav_AutoRestoresFromBak(t *testing.T) {
	path := filepath.Join(t.TempDir(), "save_4.sav")
	first := []byte("last successful save")
	writeTwice(t, path, first, []byte("half-written second save"))

	if err := os.WriteFile(path, []byte("part"), 0o644); err != nil {
		t.Fatalf("truncate .sav: %v", err)
	}

	got, err := save.ReadWithRecovery(path)
	if err != nil {
		t.Fatalf("ReadWithRecovery: %v, want automatic .bak restore", err)
	}
	if string(got) != string(first) {
		t.Errorf("recovered payload = %q, want %q (the .bak)", got, first)
	}
}

// AC-3: GIVEN the backup cannot be written (a directory occupies the .bak
// path, standing in for a full disk), WHEN the save is written, THEN the
// write fails and the previous .sav is retained intact — the last good save
// always survives a failed save attempt.
func TestWriteFailure_RetainsPreviousSav(t *testing.T) {
	path := filepath.Join(t.TempDir(), "save_5.sav")
	first := []byte("first save")
	if err := save.WriteWithBackup(path, first); err != nil {
		t.Fatalf("WriteWithBackup(first): %v", err)
	}

	if err := os.Mkdir(bakPath(path), 0o755); err != nil {
		t.Fatalf("Mkdir(.bak path): %v", err)
	}

	if err := save.WriteWithBackup(path, []byte("second save")); err == nil {
		t.Fatal("WriteWithBackup succeeded, want error (backup copy failed)")
	}

	got, err := save.ReadWithRecovery(path)
	if err != nil {
		t.Fatalf("ReadWithRecovery after failed write: %v", err)
	}
	if string(got) != string(first) {
		t.Errorf("payload after failed write = %q, want %q (previous save retained)", got, first)
	}
}
