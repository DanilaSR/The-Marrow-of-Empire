# Smoke Test: Critical Paths

**Purpose**: Run these checks in under 15 minutes before any QA hand-off.
**Run via**: `/smoke-check` (which reads this file)
**Update**: Add new entries when new core systems are implemented.

## Core Stability (always run)

1. Game launches to main menu without crash
2. New game can be started from the main menu
3. Main menu responds to all inputs without freezing
4. AI Dialogue Engine model loads successfully (status: Ready)
5. Save game completes without error
6. Load game restores correct state (entity positions, investigation progress)

## Core Mechanic — AI NPC Conversation

7. Player can initiate conversation with an NPC
8. Free-text question input works (type + Enter)
9. NPC generates an in-character response within 2 seconds
10. Topic suggestions appear based on NPC knowledge domains
11. Conversation can be ended and restarted

## Core Mechanic — Investigation

12. NPC facts with `leads_to` hints create investigation threads
13. Investigation Journal displays active threads correctly
14. Contradiction between two NPC facts is detected and flagged

## Core Mechanic — Combat

15. Combat initiates when player attacks a hostile entity
16. Turn order displays correctly with AP per character
17. Attack resolves with damage applied to target
18. Combat ends when all enemies are defeated; XP awarded

## Data Integrity

19. Save/load round-trip: investigation threads survive reload
20. Save/load round-trip: cleared ruin remains cleared after reload
21. No save file corruption on consecutive quick-saves

## Entity Framework (Sprint 1)

25. Entity creation succeeds and returns valid GameEntityID
26. Component SetComponent/GetComponent round-trip returns correct value
27. Spatial index range query returns entities within range, excludes entities outside range
28. Entity save/load round-trip produces identical entity state (all components, positions, lifecycles)

## Performance

22. 60fps target maintained during exploration (no combat, no dialogue)
23. No frame drops below 30fps during LLM inference
24. No memory growth over 10 minutes of play
