---
name: pod-serialization-measurements
description: Measured Kaiju pod serialization throughput, allocations, determinism, and write-path hazards (2026-08-09, Go 1.26.5, linux/amd64)
metadata:
  type: reference
---

Measured `kaijuengine.com/engine/encoding/pod` (benchmarked 2026-08-09, temp benchmark removed after use; re-create on demand). Fixture: 500-entity world modeled on EF CR-2 components, 1.09MB encoded blob (51KB thumbnail + 63KB world state included).

- Encode: ~73ms / 1.09MB (14.9 MB/s), 8.84MB allocated, 681,805 allocs/op
- Decode: ~42ms, 3.8MB allocated, 649,374 allocs/op
- Encode to unbuffered `os.File` instead of `bytes.Buffer`: 751ms — 10.2x slower (binary.Write issues one syscall per scalar; ~681k syscalls). Buffered writer or encode-to-memory-then-single-write is mandatory.
- `[]byte` image payload (51.2KB) encodes to ~102KB (2.0x bloat: pod writes a type-id byte per uint8 element) and takes 11.7ms with 153k allocs. Store image blobs raw outside pod, or accept 2x size + ~10ms.
- CRC32-IEEE of 1.09MB: 74us. SHA-256: 3.65ms. Checksum cost is a non-issue at save sizes.
- Non-deterministic output: two encodes of identical state differ (Go map iteration order leaks via `reflect.Value.MapKeys()`). Same byte count, different bytes. Byte-equality features (re-encode verify, dedup) are impossible; checksum-over-written-bytes still works.
- Encode does two full reflection walks (extractUsedRegistryKeys + extractUsedFieldKeys, then encodeValue) plus `slices.Index` linear string searches per field/value. `klib.BinaryWrite` = `encoding/binary.Write` (second reflection layer per scalar).
- pod caps: 253 unique registered types per encode; 255 fields per struct; empty slices/maps/arrays are skipped on encode (lean entities encode tiny).
- EntityData/RegisterEntityData path: engine callers pass struct VALUES to `Encode` (pointer branch in encodeFields appears dead code — `k > reflect.UnsafePointer` is never true).
- Screenshot capture: `device.ScreenshotRGBA()` is a full-resolution swapchain readback via `host.RunOnRenderThread` (blocking, pipeline-stalling). No 320px offscreen render path exists yet.
- Steam wrapper (`kaiju/src/platform/steam/steam_api.go`) exposes only Friends/UserStats/Utils — no RemoteStorage/Cloud API. Steam Cloud sync has zero engine support today.

How to apply: cite these numbers in save/load, EF serialization budget reviews, and any pod-based persistence design. Budgets claiming <100ms/2500 entities (EF CR-8) are unverified vs measured ~0.15ms/entity encode at this density — 3.7x over.
