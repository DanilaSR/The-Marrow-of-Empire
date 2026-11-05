# Main Menu & Settings — Review Log

## Review — 2026-08-09 — Verdict: MAJOR REVISION NEEDED (revised same session)
Scope signal: M
Specialists: ux-designer, ui-programmer, qa-lead, accessibility-specialist, creative-director (senior)
Blocking items: 7 (pre-revision) → 0 (post-revision) | Recommended: 18 → 6

**Summary**: Four independent adversarial reviews converged on three core failures.
(1) The GDD had six cross-GDD contract breaks with Camera and Save/Load —
first-launch prompt dismissal semantics were irreconcilable, seven camera
accessibility settings were orphaned, the Camera category was stale, camera-on-load
behavior contradicted between GDDs, and thumbnail format was JPEG in CR-2 vs PNG
in the UI contract. (2) The dependency model was fiction where it mattered most —
the Input System binding registry does not exist in Kaiju Engine, yet the GDD
marked it Hard. Steam Cloud, VSync toggle, texture quality, AA, audio device
selection, and low-pass filter were all promised but none exist in the engine.
(3) The AC suite was fundamentally unmeasurable — 16 of 26 ACs failed on
measurement: game-time animations asserted as wall-clock, "within one frame" with
no frame-stepping harness, freeze claims with no observability contract, and
performance ACs with no hardware, no tool, no percentile. Additional findings:
save slot metadata contrast was 1.17:1 (WCAG requires 4.5:1), no keyboard/gamepad
navigation guarantee existed for any screen except the prompt, the Summary section
was empty, and a stale duplicate GDD file coexisted with the live document.

**Revision (2026-08-09, same session)**: All 7 blockers resolved in a single
revision pass. Prompt dismissal reconciled with Camera GDD (Escape = defaults).
Thumbnails converged on PNG. All 7 orphaned accessibility settings added; Text Size,
Subtitle Size, Speaker Name added. Input Binding System split into its own GDD;
Deferred Engine Features appendix added (6 features with owners and engine
prerequisites). AC measurement protocol section added; 5 key ACs rewritten with
game-time clocks, frame-stepping harness, freeze observability, and Camera GDD
benchmark precedent. Save slot metadata constrained to dark stone cards (≥12:1);
destructive buttons use oxblood fill; amber selection gets redundant cues. Global
keyboard/gamepad navigation guarantee added as CR-8 with focus indicator spec.
settings.cfg format specified (JSON with debounced write). Camera Shake labels
renamed to Full/Reduced/Off. "Decide Later" auto-save disable now shows persistent
warning. Stale duplicate GDD deleted. Tactical Combat Engine dependency status
corrected. Summary section filled.

Prior verdict resolved: Yes — MAJOR REVISION NEEDED → revised same session → pending re-review
