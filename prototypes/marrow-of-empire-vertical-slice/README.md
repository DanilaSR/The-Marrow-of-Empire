# Marrow of Empire — Vertical Slice

## Hypothesis being tested

> A player, starting from nothing, experiences the core fantasy — asking
> questions, following a thread, fighting for access, and connecting the
> dots — within ~5 minutes, without developer guidance; and that loop can be
> built at representative quality on schedule.

**Note on structure**: unlike a throwaway prototype, this slice is
**production-tree integration** (user-approved): the INT work lives in
`src/` and assembles the implemented epics into one playable loop. Only the
planning/record documents live here. See `PLAN.md` for scope and the
per-INT records for details.

## How to run

```bash
# 1. Build the standalone game (from kaiju/src)
cd kaiju/src && CGO_ENABLED=1 go build -tags="debug,ai_driver" -o ../../bin/game-standalone .

# 2. Build the LLM companion (from src)
cd ../src && go build -o ../bin/companion ./ai_dialogue/companion/

# 3. Run from the PROJECT ROOT (content DB is CWD-relative)
LLAMA_SERVER_BIN=/path/to/llama-server ./bin/game-standalone
```

Drive/verify via the AI Driver on `127.0.0.1:7777` (health, screenshot,
input) — see the `kaijuengine-aidriver` skill.

## Status: in progress

| INT | Scope | Status |
|-----|-------|--------|
| 1 | World bootstrap (walkable debug world) | ✅ Complete |
| 2 | Talk loop (3 NPCs, dialogue pipeline, LLM companion) | ✅ Complete |
| 3 | Combat wiring + mine encounter + victory/loot | ✅ Complete |
| 4 | Journal/revelation + panel fix + playtest | Planned |

## Findings (running list)

1. Local LLM decode ~1.2–1.4 tok/s on dev hardware vs 10 tok/s gate — real
   model works but is too slow for real-time dialogue on this machine.
2. Markup absolute-positioned panels render top-left at ~half size in this
   engine version (title screen/console patterns work).
3. NPC pick radius must absorb a ~1.7u click-ray ground offset (1.5 → 3.0).
4. Matcher threshold 0.3 too strict for single-topic questions (slice: 0.1).
5. Latent engine panic: un-Init'd portrait element; markup text needs
   `InnerLabel()`, not `ToLabel()`.
6. A second `ui.Manager` on one host does not render — share the game's.

Full details per INT: `int-001-world-bootstrap.md`, `int-002-talk-loop.md`.
