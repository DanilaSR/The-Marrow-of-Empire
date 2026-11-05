# Audio Assets

Story AUDIO-001 (Audio Engine Foundation) establishes this layout
(production/epics/audio-system/story-001-audio-engine-foundation.md, impl
note 3). Assets are loaded through the engine asset pipeline from the
database rooted at this directory.

## Subdirectories

| Directory  | Format                 | Story   | Content                         |
|------------|------------------------|---------|---------------------------------|
| `sfx/`     | WAV (uncompressed)     | AUDIO-001/004 | Impacts, UI feedback, footsteps |
| `music/`   | OGG (compressed)       | AUDIO-001/003 | Adaptive music layers           |
| `ambient/` | OGG (compressed)       | AUDIO-002 | Time-of-day soundscapes         |

## Fixtures

- `sfx/test_tone.wav` — 1s 440 Hz sine, 22.05 kHz mono PCM s16le. QA fixture
  for Story AUDIO-001 AC-3.
- `music/test_music.ogg` — 1s 330 Hz sine, 22.05 kHz mono Vorbis. QA fixture
  for Story AUDIO-001 AC-3.

## Conventions

- Keys are relative to this directory (e.g. `sfx/test_tone.wav`).
- SFX are always uncompressed WAV (low decode latency, short sounds); music
  and ambient are always compressed OGG (storage).
- File names: kebab-case (ADR-0004 naming rule).
