# Epic: AI Dialogue Engine

> **Layer**: Feature | **GDD**: ai-dialogue-engine.md | **Status**: Ready | **Stories**: 4 stories
> **Governing ADRs**: ADR-0008 | **Engine Risk**: HIGH

## Overview
Implement the local LLM inference system: llama.cpp subprocess with stdin/stdout JSON IPC, prompt construction from NPC Knowledge Graph data (personality + trust-filtered facts + history + question), response pipeline with graceful degradation (text fallback), model warm-up during main menu, and the LLMBackend interface for future backend swaps. Core innovation — Pillar 1 delivery mechanism.

## GDD Requirements
| TR-ID | Requirement | ADR |
|-------|-------------|-----|
| TR-ai-001 | LLM via GGUF (~3.8B, Q5_K_M) | ADR-0008 ✅ |
| TR-ai-002 | Prompt template | ADR-0008 ✅ |
| TR-ai-003 | Graceful degradation | ADR-0008 ✅ |
| TR-ai-004 | <2s response, non-blocking | ADR-0008 ✅ |
| TR-ai-005 | IPC subprocess | ADR-0008 ✅ |

## Stories

| # | Story | Type | Status | ADR |
|---|-------|------|--------|-----|
| 001 | LLM Backend Interface & IPC Subprocess | Integration | Complete | ADR-0008 |
| 002 | Prompt Builder | Logic | Complete | ADR-0008 |
| 003 | Response Post-Processor | Logic | Ready | ADR-0008 |
| 004 | State Machine, Degradation & Conversation Memory | Integration | Ready | ADR-0008 |

| 005 | Tiered Routing & Tiny Model CGo Backend | Integration | Ready | ADR-0008 (revised) |

## Next Step
Run `/story-readiness production/epics/ai-dialogue-engine/story-001-llm-backend-ipc.md`
