// Unit tests for Story SAVE-002: Platform save paths (TR-save-004, ADR-0002
// CR-6). ResolveSaveDir is a pure function of (GOOS, env lookup, home dir), so
// every platform's resolution is verified on any host OS — no filesystem I/O,
// no environment mutation. Test naming: Test[Feature]_[Scenario]_[Expected].
package save_test

import (
	"path/filepath"
	"runtime"
	"testing"

	"marrow-of-empire/save"
)

// env builds a fixed environment lookup for the tests: the exact value under
// test is explicit, and no ambient environment can leak in.
func env(values map[string]string) func(string) (string, bool) {
	return func(key string) (string, bool) {
		v, ok := values[key]
		return v, ok
	}
}

// AC-4: GIVEN Linux with XDG_DATA_HOME set, WHEN the save dir is resolved,
// THEN it is $XDG_DATA_HOME/marrow-of-empire/saves.
func TestResolveSaveDir_LinuxWithXDGDataHome_ReturnsXDGPath(t *testing.T) {
	got := save.ResolveSaveDir("linux", env(map[string]string{
		"XDG_DATA_HOME": "/home/kael/.local/share",
	}), "/home/kael")

	want := filepath.Join("/home/kael/.local/share", save.AppDirName, save.SavesDirName)
	if got != want {
		t.Errorf("ResolveSaveDir(linux, XDG set) = %q, want %q", got, want)
	}
}

// AC-4: GIVEN Linux without XDG_DATA_HOME, WHEN the save dir is resolved,
// THEN it is ~/.local/share/marrow-of-empire/saves (the XDG default).
func TestResolveSaveDir_LinuxWithoutXDGDataHome_FallsBackToLocalShare(t *testing.T) {
	got := save.ResolveSaveDir("linux", env(nil), "/home/kael")

	want := filepath.Join("/home/kael", ".local", "share", save.AppDirName, save.SavesDirName)
	if got != want {
		t.Errorf("ResolveSaveDir(linux, no XDG) = %q, want %q", got, want)
	}
}

// XDG Base Directory spec edge case: a relative XDG_DATA_HOME is ignored, so
// the default data home is used instead.
func TestResolveSaveDir_LinuxRelativeXDGDataHome_IsIgnored(t *testing.T) {
	got := save.ResolveSaveDir("linux", env(map[string]string{
		"XDG_DATA_HOME": "relative/path",
	}), "/home/kael")

	want := filepath.Join("/home/kael", ".local", "share", save.AppDirName, save.SavesDirName)
	if got != want {
		t.Errorf("ResolveSaveDir(linux, relative XDG) = %q, want %q (relative XDG ignored)", got, want)
	}
}

// AC-5: GIVEN Windows with APPDATA set, WHEN the save dir is resolved, THEN
// it is %APPDATA%/marrow-of-empire/saves.
func TestResolveSaveDir_WindowsWithAppData_ReturnsAppDataPath(t *testing.T) {
	got := save.ResolveSaveDir("windows", env(map[string]string{
		"APPDATA": `C:\Users\Kael\AppData\Roaming`,
	}), `C:\Users\Kael`)

	want := filepath.Join(`C:\Users\Kael\AppData\Roaming`, save.AppDirName, save.SavesDirName)
	if got != want {
		t.Errorf("ResolveSaveDir(windows, APPDATA set) = %q, want %q", got, want)
	}
}

// GIVEN Windows without APPDATA, WHEN the save dir is resolved, THEN the
// roaming profile fallback is used.
func TestResolveSaveDir_WindowsWithoutAppData_FallsBackToRoamingProfile(t *testing.T) {
	got := save.ResolveSaveDir("windows", env(nil), `C:\Users\Kael`)

	want := filepath.Join(`C:\Users\Kael`, "AppData", "Roaming", save.AppDirName, save.SavesDirName)
	if got != want {
		t.Errorf("ResolveSaveDir(windows, no APPDATA) = %q, want %q", got, want)
	}
}

// GIVEN macOS, WHEN the save dir is resolved, THEN it is
// ~/Library/Application Support/marrow-of-empire/saves.
func TestResolveSaveDir_Darwin_ReturnsApplicationSupportPath(t *testing.T) {
	got := save.ResolveSaveDir("darwin", env(nil), "/Users/kael")

	want := filepath.Join("/Users/kael", "Library", "Application Support", save.AppDirName, save.SavesDirName)
	if got != want {
		t.Errorf("ResolveSaveDir(darwin) = %q, want %q", got, want)
	}
}

// GIVEN an unknown platform or a missing home dir, WHEN the save dir is
// resolved, THEN the legacy "saves" fallback is returned — the pre-SAVE-002
// behavior — and resolution never panics.
func TestResolveSaveDir_UnknownPlatformOrNoHome_FallsBackToSaves(t *testing.T) {
	if got := save.ResolveSaveDir("plan9", env(nil), "/home/kael"); got != "saves" {
		t.Errorf("ResolveSaveDir(plan9) = %q, want %q", got, "saves")
	}
	if got := save.ResolveSaveDir("linux", env(nil), ""); got != "saves" {
		t.Errorf("ResolveSaveDir(linux, no home) = %q, want %q", got, "saves")
	}
	if got := save.ResolveSaveDir("windows", env(nil), ""); got != "saves" {
		t.Errorf("ResolveSaveDir(windows, no home) = %q, want %q", got, "saves")
	}
}

// Host sanity: DefaultSaveDir on the running OS is an absolute path ending in
// marrow-of-empire/saves (AC-4/AC-5 on the host platform). Skipped on GOOS
// values without a platform-standard directory.
func TestDefaultSaveDir_OnHost_IsAbsoluteAppSavesPath(t *testing.T) {
	switch runtime.GOOS {
	case "linux", "windows", "darwin":
	default:
		t.Skipf("no platform-standard save path for GOOS %q", runtime.GOOS)
	}
	dir := save.DefaultSaveDir()
	if !filepath.IsAbs(dir) {
		t.Errorf("DefaultSaveDir() = %q, want an absolute path", dir)
	}
	if filepath.Base(dir) != save.SavesDirName || filepath.Base(filepath.Dir(dir)) != save.AppDirName {
		t.Errorf("DefaultSaveDir() = %q, want .../%s/%s", dir, save.AppDirName, save.SavesDirName)
	}
}
