# Story 001: LLM Backend Interface & IPC Subprocess

> **Epic**: AI Dialogue Engine
> **Status**: Complete
> **Layer**: Feature
> **Type**: Integration
> **Estimate**: M (6–8 hours)
> **Manifest Version**: 2026-08-09
> **Last Updated**: 2026-08-11

## Completion Notes
**Completed**: 2026-08-11
**Criteria**: 3/5 passing (AC-25 warm-up timing, AC-26 RSS budget deferred — require reference hardware for MANUAL QA)
**Deviations**: ADVISORY — ADR-0008 degradation table (60s retry, 3-timeout→restart) diverges from GDD CR-5 (session-stable). Implementation follows GDD as authoritative.
**Test Evidence**: Integration — `tests/integration/ai_dialogue/llm_backend_test.go` (4 tests, all PASS, go vet clean, -race clean)
**Code Review**: Complete — lead-programmer + qa-tester, 3 fixes applied (restart-during-shutdown leak, SIGTERM alignment, nil-err guards)

## Context

**GDD**: `design/gdd/ai-dialogue-engine.md`
**Requirements**:
- `TR-ai-001`: LLM via GGUF (~3.8B, Q5_K_M), llama.cpp inference backend
- `TR-ai-005`: IPC — separate llama.cpp process, stdin/stdout JSON, spawn/health-check/restart lifecycle

> ⚠ **GDD/ADR model mismatch**: The GDD (2026-08-09 review) designates **Fiendish_LLAMA_3B.Q4_K_M** (Llama 3.2 3B RP fine-tune, ~2.1 GB) as the authoritative model. ADR-0008 and TR-ai-001 still reference the older NPC-LLM-3_8B-Q5_K_M (Phi3 3.8B, ~2.7 GB). The GDD is authoritative — implement against the Fiendish model spec. The ADR and TR registry will be updated separately.

**ADR Governing Implementation**: ADR-0008: LLM Inference Integration
**ADR Decision Summary**: Separate llama.cpp process with stdin/stdout JSON IPC. The LLM runs as a child process spawned by the game — crash isolation, independent restart, no CGo entanglement. `LLMBackend` interface (`Generate(ctx, prompt) → (string, error)`) enables future backend swaps.

**Engine**: Kaiju Engine 26.07.29.1-nightly (Go/Vulkan) | **Risk**: HIGH
**Engine Notes**: llama.cpp GGUF inference is external to Kaiju — no Kaiju APIs required for inference. IPC mechanism uses Go `os/exec` + stdin/stdout pipes. Verify llama.cpp binary availability and IPC latency with game running.

**Control Manifest Rules (Feature Layer)**:
- Required: LLM inference via separate llama.cpp process — stdin/stdout JSON IPC, crash isolation
- Required: `LLMBackend` interface: `Generate(ctx, prompt) → (string, error)` — swappable
- Forbidden: Never use in-process CGo LLM bindings — model crash would take down the game
- Forbidden: Never use HTTP for local LLM — adds latency without benefit over stdin/stdout
- Guardrail: Model load ~15s during main menu (non-blocking), LLM memory ~3-4GB separate process

---

## Acceptance Criteria

*From GDD `design/gdd/ai-dialogue-engine.md`, scoped to this story:*

- [ ] **AC-15**: GIVEN the model file is configured to a nonexistent path, WHEN the engine initializes, THEN the state is `TextMode` AND a one-time notification is displayed AND subsequent queries return quote-styled fact text.
- [ ] **AC-16**: GIVEN the capability probe stub returns 7 tok/s (below the 10 tok/s minimum), WHEN the engine initializes, THEN the state is `TextMode`.
- [ ] **AC-20**: GIVEN the model subprocess crashes during generation, WHEN the crash is detected, THEN the current response falls back to text AND the subprocess is restarted once AND if restart succeeds, the next query uses the LLM.
- [ ] **AC-25**: GIVEN the engine on reference hardware (desktop CPU, NVMe SSD), WHEN the model loads from NVMe SSD, THEN the state reaches `Ready` (load + probe + warm-up) within 12s measured from subprocess start. SATA SSD: within 20s. Classified: MANUAL QA — model not in CI by design.
- [ ] **AC-26**: GIVEN the engine `Ready` on reference hardware with the standard question bank, WHEN 100 queries are run, THEN peak RSS stays below 10 GB total (model subprocess ~3.5–5 GB + game process ~2–4 GB). Classified: MANUAL QA.

---

## Implementation Notes

*Derived from ADR-0008 Implementation Guidelines and GDD CR-1, CR-7:*

### LLMBackend Interface

```go
type LLMBackend interface {
    Generate(ctx context.Context, prompt string) (string, error)
    Status() ModelStatus
    Shutdown() error
}

type ModelStatus int
const (
    StatusUnloaded   ModelStatus = iota
    StatusLoading
    StatusReady
    StatusGenerating
    StatusDegraded
    StatusError
)
```

### IPC Message Format (newline-delimited JSON over stdin/stdout)

```go
type PromptRequest struct {
    Prompt    string  `json:"prompt"`
    MaxTokens int     `json:"max_tokens"`
    Temp      float64 `json:"temperature"`
}

type PromptResponse struct {
    Text       string `json:"text"`
    TokensUsed int    `json:"tokens_used"`
    DurationMs int    `json:"duration_ms"`
    Error      string `json:"error,omitempty"`
}
```

### Subprocess Lifecycle
- Spawn: `os/exec.Command` for llama.cpp binary with GGUF model path as argument
- Health check: periodic ping (or detect exit via `cmd.Wait` in goroutine)
- Restart: on crash, spawn new process. Max 1 restart per session (with 5-min cooldown)
- Shutdown: send "quit" command via stdin, then SIGTERM after grace period

### Capability Probe (CR-1, Formulas § Latency Budget)
- Run a ~150-token benchmark prompt at startup
- Measure BOTH prefill speed (tok/s) and decode speed (tok/s)
- Gate: ALL THREE of (a) decode ≥10 tok/s, (b) cold-start total ≤25s × 0.85 margin, (c) cached total ≤12s
- If probe fails → session-stable TextMode

### Model Warm-Up (CR-7)
- Load during main menu (background, non-blocking)
- Steps: Load GGUF → Capability probe → Pre-fill generic greeting prompt (pages weights via mmap)
- Target: <12s NVMe desktop, <20s SATA SSD
- Main Menu shows "The old tongue wakes..." with progress
- If player starts game before warm-up completes → TextMode until ready, then transition with in-character cue

### Model File Discovery
- Check game directory first, then `$XDG_DATA_HOME/marrow-of-empire/models/`
- GGUF filename: `Fiendish_LLAMA_3B.Q4_K_M.gguf` (authoritative per GDD 2026-08-09 review)
- Checksum validation before loading

### Text Mode on Load Failure
- Missing model file or capability probe fail → `StatusDegraded` (session-stable)
- One-time notification via oxblood knot glyph + settings notification
- All NPC responses use quote-styled fact text (text-fallback rules from CR-5)

---

## Out of Scope

*Handled by neighbouring stories — do not implement here:*

- Story 002: Prompt construction from KG data (the `prompt` string passed to `Generate`)
- Story 003: Response post-processing (entity resolution, repetition detection, truncation)
- Story 004: State machine (per-NPC states, timeout handling, concurrency orchestration)
- Story 004: Conversation memory (rolling summary, disclosed facts) — save/load persistence

---

## QA Test Cases

*QA gate skipped — Lean mode. Test case specifications will be generated before /dev-story.*

- **AC-15**: Model file missing → TextMode
- **AC-16**: Capability probe fail → TextMode
- **AC-20**: Subprocess crash → auto-restart → recover
- **AC-25**: Warm-up timing on reference hardware (MANUAL QA)
- **AC-26**: Peak RSS memory budget (MANUAL QA)

---

## Test Evidence

**Story Type**: Integration
**Required evidence**: `tests/integration/ai_dialogue/llm_backend_test.go` — must exist and pass

**Status**: [ ] Not yet created

---

## Dependencies

- Depends on: None (Foundation/Core epics already implemented)
- Unlocks: Story 002 (Prompt Builder needs LLMBackend interface defined)
