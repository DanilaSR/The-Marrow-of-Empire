// Corruption recovery primitives (Story SAVE-003, TR-save-005, ADR-0002):
// CRC32 checksums and .bak backups at the file layer — the foundation the
// save/load path stands on. Every .sav written here carries the ADR-0002
// header (format version, CRC32 checksum, timestamp); before an overwrite the
// existing .sav is preserved as .bak, so a corrupted or half-written .sav
// always has a last-known-good fallback.
//
// Control manifest rules honored here (Foundation Layer, ADR-0002):
//   - Checksum (CRC32) + .bak backup — corrupted save detected, restored
//   - The checksum covers the data portion: everything after the 16-byte
//     header (crc32.ChecksumIEEE, matching the SaveManager's framing)
package save

import (
	"errors"
	"fmt"
	"hash/crc32"
	"os"
	"strings"
	"time"
)

var (
	// ErrNoValidSaveFile is returned by ReadWithRecovery when neither the .sav
	// nor the .bak file is usable (SAVE-003 AC-2): the save cannot be
	// recovered, and the caller must surface the failure to the player.
	ErrNoValidSaveFile = errors.New("save: no valid save file (.sav and .bak both unusable)")
)

// ComputeChecksum returns the CRC32 (IEEE) checksum of save data — the value
// stored in the .sav header, covering everything after the 16-byte header
// (ADR-0002). IEEE is used so .sav files stay byte-identical to the
// SaveManager's own framing, which already computes crc32.ChecksumIEEE over
// its payload.
func ComputeChecksum(data []byte) uint32 {
	return crc32.ChecksumIEEE(data)
}

// WriteWithBackup writes data as a .sav file at path, framed with the
// ADR-0002 header: [Version uint32][Checksum uint32 — CRC32 of data][Timestamp
// int64], little-endian. The header version is stamped with DefaultSaveVersion
// (the SaveManager owns versioning policy; see its Config.CurrentVersion).
//
// Before overwriting an existing .sav, the previous version is copied to the
// .bak path (GDD: "the previous version is kept as save_[slot_id].bak"). If
// that copy fails — e.g. disk full — the write aborts and the previous .sav
// is left untouched, so the last good save always survives a failed save
// attempt (SAVE-003 AC-3). A first save (no .sav yet) writes without a .bak:
// nothing is being overwritten.
func WriteWithBackup(path string, data []byte) error {
	bak := backupPath(path)
	if old, err := os.ReadFile(path); err == nil {
		if err := os.WriteFile(bak, old, 0o644); err != nil {
			return fmt.Errorf("save: write backup %q: %w", bak, err)
		}
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("save: read existing save %q for backup: %w", path, err)
	}

	framed := encodeHeader(saveHeader{
		Version:   DefaultSaveVersion,
		Checksum:  ComputeChecksum(data),
		Timestamp: time.Now().UnixNano(),
	})
	framed = append(framed, data...)
	if err := os.WriteFile(path, framed, 0o644); err != nil {
		return fmt.Errorf("save: write %q: %w", path, err)
	}
	return nil
}

// ReadWithRecovery reads the .sav file at path and returns its payload —
// everything after the 16-byte header — after verifying the CRC32 checksum
// (ADR-0002). When the .sav is unusable — checksum mismatch, or a file
// smaller than the header (clearly a crash mid-write) — the .bak file is
// tried, and its payload is returned when valid: the restore is automatic
// (SAVE-003 AC-1 and AC-4). The function never rewrites the .sav; promoting
// the recovered .bak back to .sav is the caller's decision (the UI offers it
// per AC-1, and the caller logs the incident).
//
// Errors:
//   - ErrSaveNotFound (wrapped) when no .sav exists — a missing save is not
//     corruption, so the .bak alone is never treated as a save (SAVE-003 AC-4).
//   - ErrNoValidSaveFile when the .sav is unusable and no valid .bak exists
//     (SAVE-003 AC-2: both corrupted, or no backup to fall back on).
func ReadWithRecovery(path string) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("%w: %s", ErrSaveNotFound, path)
		}
		return nil, fmt.Errorf("save: read %q: %w", path, err)
	}
	if payload, ok := verifyFramed(data); ok {
		return payload, nil
	}

	bak := backupPath(path)
	bakData, bakErr := os.ReadFile(bak)
	if bakErr != nil {
		return nil, fmt.Errorf("%w: .sav unusable (%s), .bak unreadable: %v",
			ErrNoValidSaveFile, describe(data), bakErr)
	}
	if payload, ok := verifyFramed(bakData); ok {
		return payload, nil
	}
	return nil, fmt.Errorf("%w: .sav unusable (%s), .bak unusable (%s)",
		ErrNoValidSaveFile, describe(data), describe(bakData))
}

// verifyFramed checks that data carries a full 16-byte header whose checksum
// matches the payload (CRC32 of everything after the header, ADR-0002),
// returning the payload on success. A file smaller than the header cannot
// carry a checksum and is always rejected — the incomplete-write case.
func verifyFramed(data []byte) ([]byte, bool) {
	if len(data) < saveHeaderSize {
		return nil, false
	}
	header := decodeHeader(data[:saveHeaderSize])
	payload := data[saveHeaderSize:]
	if ComputeChecksum(payload) != header.Checksum {
		return nil, false
	}
	return payload, true
}

// describe summarizes why a file failed verification, for error messages.
func describe(data []byte) string {
	if len(data) < saveHeaderSize {
		return fmt.Sprintf("%d bytes, shorter than the %d-byte header", len(data), saveHeaderSize)
	}
	return "checksum mismatch"
}

// backupPath returns the .bak path for a .sav path (GDD: save_[slot_id].bak —
// the .sav extension is replaced, e.g. save_3.sav -> save_3.bak).
func backupPath(path string) string {
	if strings.HasSuffix(path, ".sav") {
		return strings.TrimSuffix(path, ".sav") + ".bak"
	}
	return path + ".bak"
}
