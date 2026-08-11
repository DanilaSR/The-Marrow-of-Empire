// Platform-standard save directory resolution (Story SAVE-002, TR-save-004,
// ADR-0002 CR-6): the game's .sav files live in the OS-standard per-user data
// directory instead of the working directory.
//
//	Linux:   $XDG_DATA_HOME/marrow-of-empire/saves   (default ~/.local/share/...)
//	Windows: %APPDATA%/marrow-of-empire/saves
//	macOS:   ~/Library/Application Support/marrow-of-empire/saves
//
// Resolution is a pure function of the OS name, an environment lookup, and the
// home directory (ResolveSaveDir), so every platform can be unit-tested on any
// host. DefaultSaveDir wires it to the running process.
package save

import (
	"os"
	"path/filepath"
	"runtime"
)

const (
	// AppDirName is the application directory under each platform's data
	// root (ADR-0002: marrow-of-empire).
	AppDirName = "marrow-of-empire"
	// SavesDirName is the saves subdirectory under AppDirName (ADR-0002:
	// save files live in .../marrow-of-empire/saves/).
	SavesDirName = "saves"
	// fallbackSaveDir is the last-resort save directory when no platform
	// directory can be resolved (unknown OS, no home dir). It matches the
	// pre-SAVE-002 behavior: "saves" relative to the working directory.
	fallbackSaveDir = "saves"
)

// DefaultSaveDir returns the platform-standard save directory for the running
// process (TR-save-004, ADR-0002 CR-6). SaveManager falls back to it when
// Config.SaveDir is empty.
//
// Example usage:
//
//	dir := save.DefaultSaveDir()
//	slog.Info("save directory", "dir", dir)
func DefaultSaveDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		home = ""
	}
	return ResolveSaveDir(runtime.GOOS, os.LookupEnv, home)
}

// ResolveSaveDir resolves the platform-standard save directory for the given
// GOOS value from an environment lookup and a home directory — the seam
// DefaultSaveDir uses. Exported so every platform's resolution is
// unit-testable on any host OS (the tests module lives outside this package).
//
// The XDG Base Directory spec is honored: a relative XDG_DATA_HOME is
// ignored. When no platform directory can be resolved (unknown GOOS, empty
// home dir), fallbackSaveDir is returned.
//
// Example usage:
//
//	dir := save.ResolveSaveDir("linux", os.LookupEnv, homeDir)
func ResolveSaveDir(goos string, lookupEnv func(string) (string, bool), homeDir string) string {
	switch goos {
	case "linux":
		if dir, ok := lookupEnv("XDG_DATA_HOME"); ok && dir != "" && filepath.IsAbs(dir) {
			return filepath.Join(dir, AppDirName, SavesDirName)
		}
		if homeDir != "" {
			return filepath.Join(homeDir, ".local", "share", AppDirName, SavesDirName)
		}
	case "windows":
		if dir, ok := lookupEnv("APPDATA"); ok && dir != "" {
			return filepath.Join(dir, AppDirName, SavesDirName)
		}
		if homeDir != "" {
			return filepath.Join(homeDir, "AppData", "Roaming", AppDirName, SavesDirName)
		}
	case "darwin":
		if homeDir != "" {
			return filepath.Join(homeDir, "Library", "Application Support", AppDirName, SavesDirName)
		}
	}
	return fallbackSaveDir
}
