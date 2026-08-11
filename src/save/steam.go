// Steam Auto-Cloud integration contract (Story SAVE-002, TR-save-003,
// ADR-0002 CR-7). This file defines the Cloud contract and the no-op stub;
// it never references Steamworks code, so every build compiles and save/load
// works identically without Steam (AC-3).
//
// INTEGRATION POINT — real Steam Auto-Cloud implementation:
// kaijuengine.com/platform/steam (engine 26.07.29.1-nightly) exposes
// init/user/stats APIs but no remote-storage bindings, and the Steamworks SDK
// is not installed in this repo, so no real implementation ships here. To
// implement Steam Auto-Cloud:
//  1. Extend kaijuengine.com/platform/steam with ISteamRemoteStorage bindings
//     (FileWrite, FileRead, FileDelete, FileExists, FileGetTimestamp,
//     FileSize) mirroring the existing steam_wrapper.cpp/h pattern.
//  2. Add a file in this package with //go:build steam defining a steamCloud
//     type implementing Cloud plus a NewCloud override — mirroring the
//     engine's bootstrap_with_steam.go / bootstrap_without_external_game_service.go
//     tag pair.
//  3. Auto-Cloud semantics: RegisterFile must write through
//     ISteamRemoteStorage::FileWrite so the Steam client syncs the file and
//     knows it exists (AC-1). Local writes always happen first; cloud sync is
//     best-effort (ADR-0002: "local save always succeeds").
package save

import (
	"errors"
	"time"
)

// ErrCloudUnavailable is returned by every Cloud operation when Steam Cloud
// is not available: no Steam runtime (offline/GOG), the client is not running,
// or the user is not logged on. Callers treat cloud errors as best-effort —
// local save/load is never blocked by the cloud (AC-3, ADR-0002).
var ErrCloudUnavailable = errors.New("save: steam cloud unavailable")

// Cloud is the Steam Auto-Cloud integration contract (TR-save-003). Save
// files are registered with the cloud on creation (RegisterFile); at launch
// the UI compares each local save's timestamp with FileInfo and, on mismatch,
// shows both versions and lets the player choose (AC-2 — the dialog itself is
// UI).
type Cloud interface {
	// Available reports whether Steam Cloud can be used right now: the
	// Steam client is running, the user is logged on, and remote storage is
	// enabled.
	Available() bool

	// RegisterFile writes data to the cloud and registers the file with
	// Steam Auto-Cloud (ISteamRemoteStorage::FileWrite), so the Steam client
	// syncs it (AC-1). name is the save file name, e.g. "save_3.sav".
	RegisterFile(name string, data []byte) error

	// FileInfo reports the cloud copy of the named save file. Exists is
	// false — with nil error — when the cloud holds no copy; other failures
	// return ErrCloudUnavailable.
	FileInfo(name string) (CloudFile, error)

	// ReadFile downloads the cloud copy of the named save file
	// (ISteamRemoteStorage::FileRead). The conflict dialog uses it to show
	// the cloud version before the player chooses (AC-2).
	ReadFile(name string) ([]byte, error)

	// DeleteFile removes the cloud copy (ISteamRemoteStorage::FileDelete).
	DeleteFile(name string) error
}

// CloudFile describes the cloud copy of one save file (TR-save-003: conflict
// detection compares timestamps with the local copy).
type CloudFile struct {
	// Name is the save file name, e.g. "save_3.sav".
	Name string
	// Exists is false when the cloud holds no copy of the file.
	Exists bool
	// Size is the cloud copy size in bytes.
	Size int64
	// ModifiedAt is the cloud copy's last write time, used to detect a
	// local/cloud mismatch at launch (AC-2).
	ModifiedAt time.Time
}

// NoopCloud is the no-op Cloud stub for builds without Steamworks and for the
// offline/GOG case (AC-3): every operation reports the cloud as unavailable,
// so save/load functions exactly as if Steam Cloud did not exist.
type NoopCloud struct{}

// Available implements Cloud: always false — there is no Steam runtime.
func (NoopCloud) Available() bool { return false }

// RegisterFile implements Cloud. See the interface doc.
func (NoopCloud) RegisterFile(name string, data []byte) error { return ErrCloudUnavailable }

// FileInfo implements Cloud: the cloud holds no copies.
func (NoopCloud) FileInfo(name string) (CloudFile, error) {
	return CloudFile{Name: name}, ErrCloudUnavailable
}

// ReadFile implements Cloud. See the interface doc.
func (NoopCloud) ReadFile(name string) ([]byte, error) { return nil, ErrCloudUnavailable }

// DeleteFile implements Cloud. See the interface doc.
func (NoopCloud) DeleteFile(name string) error { return ErrCloudUnavailable }

// NewCloud returns the Cloud implementation for this build. Until the engine
// ships remote-storage bindings (see the integration-point note at the top of
// this file), every build gets the NoopCloud stub: Steam Cloud is reported
// unavailable and local save/load is unaffected (AC-3).
//
// Example usage:
//
//	cloud := save.NewCloud()
//	if cloud.Available() {
//	    _ = cloud.RegisterFile(filepath.Base(path), data) // best-effort (AC-1)
//	}
func NewCloud() Cloud { return NoopCloud{} }
