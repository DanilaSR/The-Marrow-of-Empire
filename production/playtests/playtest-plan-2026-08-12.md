# Playtest Plan — Production Gate Validation

**Created**: 2026-08-12
**Purpose**: Generate the playtest evidence required by the Production → Polish gate
**Gate gaps addressed**: Artifacts 9-11 (playtest sessions, coverage, fun validation), Quality checks 3/5/6 (core loop, playtest findings, confusion loops)

## Playtest Goals

1. **Validate the core fantasy**: Does the player feel like an investigator — observing,
   deducing, and questioning — not just a combatant?
2. **Measure the fun**: Is the central mechanic (asking NPCs questions, following threads)
   enjoyable? Do players independently describe experiences matching the Player Fantasy
   sections of core GDDs?
3. **Find confusion loops**: Any point where >50% of testers get stuck without knowing why.
4. **Map the difficulty curve**: Does challenge ramp smoothly across the first 2 hours?
5. **Establish the visual baseline**: Screenshots of every screen in-engine for Polish comparison.

## Fun Hypothesis (from Game Concept)

> "Solving a mystery through observation, deduction, and dialogue — where the player's
> mind is their greatest weapon and the reward is understanding — is more satisfying
> than defeating enemies in combat."

**Validation criteria**: At least one playtester independently describes an experience
matching this statement without being prompted. Track exact quotes.

## Required Sessions (minimum 3 for gate)

### Session 1: New Player Experience (0–30 minutes)

**Focus**: First impressions, onboarding clarity, core loop introduction.

**Observer checklist**:
- [ ] Does the game communicate what to do within 2 minutes?
- [ ] Does the player understand the controls without external help?
- [ ] Does the accessibility prompt appear and make sense? (Main Menu Story 003)
- [ ] Does the title screen convey the game's tone? (Main Menu Story 002)
- [ ] What is the first question the player asks out loud? (Record verbatim)
- [ ] Does the player notice the investigation journal? When?
- [ ] Does the player try to talk to an NPC? What happens?
- [ ] Time to first "aha" moment (if any): ___
- [ ] Confusion moments (timestamp + description):
- [ ] Fun moments (timestamp + description):

**Systems under test**: Title screen, accessibility prompt, first dialogue encounter,
investigation journal shell, local map display.

**GDD acceptance criteria to validate**:
- Main Menu: AC-1 (navigation), AC-3 (transition feel), AC-5 (focus management)
- Dialogue UI: AC-1 (panel display), AC-2 (text reveal), AC-4 (input handling)
- Investigation Journal: AC-1 (journal opens), AC-2 (three-panel layout visible)
- Map System: AC-1 (map displays), AC-2 (player position shown)

### Session 2: Mid-Game Systems (30–90 minutes)

**Focus**: Thread investigation loop, dialogue depth, combat integration, inventory.

**Observer checklist**:
- [ ] Does the player understand how to ask questions? (Question System)
- [ ] Does the player discover and follow an investigation thread?
- [ ] Does the player check the fact log / entity chips? (Journal Story 003)
- [ ] Does the player notice contradictions? (Journal Story 004)
- [ ] Does combat feel integrated into the investigation flow, or like a separate game?
- [ ] Do companion interjections fire? Are they helpful or annoying? (Party System CR-6)
- [ ] Does the player use the character sheet / inventory? (C&I UI)
- [ ] Does trust visibly change with any NPC? (Trust & Relationship)
- [ ] Confusion moments:
- [ ] Fun moments:
- [ ] Quote any unprompted description of the experience:

**Systems under test**: Question System, Thread Investigation, Fact Log & Entity Chips,
Contradiction Display, Combat HUD, Character Sheet, Inventory Grid, Trust System.

**GDD acceptance criteria to validate**:
- Thread Investigation: AC-1 (thread creation on clue), AC-3 (chain visualization),
  AC-5 (thread detail displays)
- Question System: AC-1 (topic matching), AC-3 (question input), AC-5 (suggestions)
- Combat Engine: AC-1 (initiative), AC-2 (grid movement), AC-3 (AP economy),
  AC-5 (attack resolution)
- C&I UI: AC-1 (stat display), AC-5 (inventory grid), AC-8 (weight bar)

### Session 3: Difficulty & Depth (90–120 minutes)

**Focus**: Difficulty curve, system mastery, replay impulse, emotional arc.

**Observer checklist**:
- [ ] Does the player feel challenged but not frustrated?
- [ ] Does the player develop strategies or system mastery?
- [ ] Does the player express curiosity about unresolved threads?
- [ ] Does the player want to keep playing after the session ends?
- [ ] Does the emotional tone match the intended mood (melancholy, mystery, warmth)?
- [ ] Are there moments that feel unfair or arbitrary?
- [ ] Does the save/load flow work as expected? (Save/Load System + Main Menu Story 004)
- [ ] Confusion loops (stuck >2 minutes without progress): ___
- [ ] Would the player recommend this to a friend? (Y/N)
- [ ] One-sentence description of the game in the player's own words:

**Systems under test**: Save/Load screens, full investigation loop (question → clue →
thread → contradiction → revelation), combat difficulty curve, companion utility.

---

## Session Protocol

### Before Each Session
1. Build the game with `go build` (verify clean build first)
2. Launch via Kaiju Engine with `ai_driver` tag for screenshot capture
3. Clear all save data (fresh profile for Session 1; prepared save for Sessions 2-3)
4. Verify screenshot capture works (test one shot)
5. Prepare observer: notepad, timestamp log, this checklist printed

### During Each Session
1. **No guidance unless stuck >5 minutes** — observe, don't teach
2. **Record timestamps** for all checklist events
3. **Capture screenshots** at key moments: first dialogue, first combat, journal open,
   map open, contradiction discovered, thread resolution, save/load screens
4. **Record verbatim quotes** — player's unprompted descriptions are the most valuable data
5. **Note all bugs** — file in `production/qa/bugs/` after session

### After Each Session
1. Fill in the post-session questionnaire below
2. File screenshots in `production/qa/evidence/playtest-[session]-[date]/`
3. Write session report using template at `production/playtemplates/session-report-template.md`
4. Update the balance observations log

---

## Post-Session Questionnaire

Ask the player these questions in conversation (not as a form — let them elaborate):

1. "What was the most fun thing you did?"
2. "What was the most frustrating thing?"
3. "If you could change one thing, what would it be?"
4. "What did you want to do that you couldn't figure out how to do?"
5. "Describe the game to someone who hasn't played it — in one sentence."
6. "Did anything feel unfair?"
7. "What moment do you remember most clearly?"
8. "Would you play another session? Why or why not?"

---

## Screenshot Baseline Requirements (for Polish Gate)

Capture these at native resolution (1920×1080 reference, 1280×720 minimum):

| Screen | Story | What to capture |
|--------|-------|-----------------|
| Title screen | MM-002 | All states: initial, after accessibility prompt, after returning from game |
| Accessibility prompt | MM-003 | First-launch dialog, each option toggled, post-selection state |
| Save/Load screens | MM-004 | Save list (empty, populated), load list, confirmation dialog, delete warning |
| Settings screen | MM-005 | Each settings tab, focus ring visible, reduced-motion toggle |
| Main menu navigation | MM-001 | Screen transition mid-animation, focus ring on each screen type |
| Dialogue panel | DUI-001 | Single speaker, multi-speaker, typewriter mid-reveal, response options |
| Investigation journal | INVJ-001 | Three-panel layout, empty state, populated state |
| Chain list & thread detail | INVJ-002 | Chain selected, thread expanded, thread collapsed |
| Fact log & entity chips | INVJ-003 | Fact log with entries, entity quick-ref bar, entity selected |
| Contradiction display | INVJ-004 | Contradiction alert, resolved contradiction, unresolved |
| Local map | MAP-001 | Map with player marker, pan state, zoom states (min/mid/max) |
| Map markers | MAP-002 | Each marker type (objective, NPC, location, clue, resolved), legend |
| Character sheet | CI-001 | Stat display, all tabs |
| Inventory grid | CI-005 | Grid with items, weight bar at each threshold, filter/sort states |
| Equipment slots | CI-003 | Empty slots, populated slots, drag highlight, comparison tooltip |
| Combat HUD | COMBAT-UI-001 | Turn order bar, action bar, health bars (ally/enemy), AP display, targeting |

---

## Systems Not Yet Testable

These systems have no runnable implementation and are deferred to later playtest rounds:

| System | Reason | Target Round |
|--------|--------|-------------|
| Party System (companions) | No implementation | Round 2 |
| Faction System | No implementation | Round 3 |
| Loot & Economy | No implementation | Round 3 |
| Persistent World State | No implementation | Round 2 |
| World Map & Region Travel | MAP-003 in progress | Round 2 |
| AI Dialogue Engine (real LLM) | CGo stubs — running text-fallback only | Round 2 |
| Audio (ambient/music/SFX) | Engine SoLoud 3D API gap | Round 3 |
| Living Environment (time/weather) | Story 001 only | Round 2 |

---

## Success Criteria for Gate Re-Check

After all 3 sessions:
- [ ] At least 1 playtester independently describes the investigator fantasy (verbatim quote recorded)
- [ ] Core loop (question NPC → gain clue → thread created → investigate → revelation) is
      observed end-to-end in at least 1 session
- [ ] Zero "confusion loops" identified (>50% of testers stuck without knowing why)
- [ ] Fun hypothesis explicitly addressed: validated, revised, or disproven with evidence
- [ ] Screenshot baseline captured for all 18 screen states listed above
- [ ] All discovered bugs filed in `production/qa/bugs/`
- [ ] 3 session reports written in `production/playtests/`
