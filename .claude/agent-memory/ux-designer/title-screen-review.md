---
name: title-screen-review
description: Title screen UX spec vs implementation review 2026-08-09 — verdict NEEDS REVISION; New Game confirm dialog missing (soft-lock), a11y modal controls inert, no reduced-motion gate
metadata:
  type: project
---

Title screen implementation review delivered 2026-08-09 against `design/ux/title-screen.md` (spec) vs `src/ui/title_screen/title_screen.go`, `title_screen_focus.go`, `assets/ui/title_screen.html`. Verdict: **NEEDS REVISION**.

**Why:** the first-launch and accessibility core purpose is the spec's centerpiece; the implementation's modal is a shell.

**Headline findings:**
1. CRITICAL — New Game confirmation dialog markup does not exist. `StateConfirmNew` shows only the dim overlay with focus cleared: with saves present, clicking New Game soft-locks the game (only unguarded `clickQuit` could escape, and it is covered by the dim overlay).
2. MAJOR — First-launch modal is cosmetic: Subtitles toggle is a static Panel (no handler, no state), "Remember my choice" checkbox has no default-checked state and is never read, dropdown defaults to "Off" (spec: Reduced). Start Game writes only `{"firstLaunchComplete":true}` to `settings/title_screen.json` — the three accessibility values are never persisted (spec requires them in `settings.cfg`).
3. MAJOR — Reduced-motion gate absent: no setting is read, no duration collapse, `SetAtmospheric()` is a no-op flag.
4. MAJOR — Loading state sets root opacity 0 (transparent), spec says hard cut to warm soot `#141210`.
5. MAJOR — No Escape / gamepad B anywhere (spec: dismisses dialog and toast); toast has no click handler and auto-dismisses at 4s not 5s; knot glyph is a literal "!".
6. MAJOR (verify) — `menuPanel = panels[1]` in bindElements is almost certainly the wrong element (dim-overlay for direct children, title-rule for all descendants) — the 300ms entrance slide likely animates the wrong node. Engine `FindElementsByTag` semantics unverified.
7. MAJOR — Tooltip shows unconditionally in NoSaves state (spec: on hover/focus); after FirstLaunch→Default with no saves, the disabled Load button stays in focus order with no tooltip — two inconsistent disabled representations.
8. Minor — 8 of 11 spec motions have no Go tween (modal slide/dim, dialog slide/dim, toast slide, desaturate crossfade, exit fade-in); only scene fade, menu slide, exit fade-out covered; background scene is an empty Image with no animated smoke/river/ship binding.
9. Minor — `clickQuit` has no state guard (other three buttons do); StateError leaves the FirstLaunch modal visible (from settings-write-failure path) and dims the menu 72% behind the toast (spec: menu fully interactive).

**How to apply:** when the ui-programmer/team-ui asks about title screen fixes, lead with findings 1-2-3. The Kaiju Go-side tween pattern itself (polling tween values in Update via `SetElementStyleProperty`) is the right architecture for the no-transitions CSS engine — the gap is coverage, not approach.
