# Story 003: Corruption Recovery

> **Epic**: Save/Load System
> **Status**: Ready
> **Layer**: Foundation
> **Type**: Logic
> **Estimate**: S (2-3 hours)
> **Manifest Version**: 2026-08-09
> **Depends on**: Story 001

## Context

**GDD**: `design/gdd/save-load-system.md`
**Requirement**: `TR-save-005`

**ADR**: ADR-0002 | **Engine**: Kaiju 26.07.29.1-nightly | **Risk**: HIGH

## Acceptance Criteria

- [ ] GIVEN a .sav file with mismatched checksum, WHEN the player attempts to load, THEN corruption is detected and the player is offered to restore from .bak
- [ ] GIVEN the .bak file is intact, WHEN restore is selected, THEN the .bak loads successfully (state matches last successful save)
- [ ] GIVEN disk is full during save, WHEN the write fails, THEN the previous .bak is retained and player sees "Save failed — insufficient disk space"
- [ ] GIVEN a crash mid-save, WHEN the game restarts, THEN the incomplete .sav is detected (checksum mismatch) and the .bak is restored automatically

## Implementation Notes

From ADR-0002: CRC32 checksum over the data portion (everything after the 16-byte header). Before overwriting a save, the existing .sav is copied to .bak. On load: verify checksum → if mismatch, check .bak → if .bak valid, offer restore. Auto-restore without prompt if .sav was clearly incomplete (file size < header size).
