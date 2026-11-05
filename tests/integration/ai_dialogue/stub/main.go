// Command llm-stub is a test double for the LLM companion binary: it speaks
// the stdin/stdout JSON protocol from marrow-of-empire/ai_dialogue/ipc.go
// (newline-delimited JSON) and simulates llama.cpp behavior — controllable
// prefill/decode latency, deliberate crashes, clean quit. It is built and
// spawned by the integration tests (llm_backend_test.go); it is never
// shipped.
//
// Behavior:
//   - Reads one JSON line per request on stdin.
//   - `{"command":"quit"}` → exit 0.
//   - A prompt containing "CRASH" → exit 1 immediately (no response),
//     simulating a subprocess crash during generation (AC-20).
//   - Any other request → sleeps STUB_PREFILL_MS, then
//     (max_tokens-1) × STUB_TOKEN_MS, then replies with a canned response.
//
// Speed knobs are environment variables (inherited by the spawned process):
//
//	STUB_PREFILL_MS  prefill sleep per request (default 50)
//	STUB_TOKEN_MS    per-token decode sleep (default 20 → 50 tok/s)
//	STUB_RESPONSE    canned response text (default: in-character line)
package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type request struct {
	Prompt    string  `json:"prompt"`
	MaxTokens int     `json:"max_tokens"`
	Temp      float64 `json:"temperature"`
}

type response struct {
	Text       string `json:"text"`
	TokensUsed int    `json:"tokens_used"`
	DurationMs int    `json:"duration_ms"`
	Error      string `json:"error,omitempty"`
}

type shutdownRequest struct {
	Command string `json:"command"`
}

func main() {
	prefillMs := envInt("STUB_PREFILL_MS", 50)
	tokenMs := envInt("STUB_TOKEN_MS", 20)
	responseText := os.Getenv("STUB_RESPONSE")
	if responseText == "" {
		responseText = "The old tongue answers, briefly and in character."
	}

	scanner := bufio.NewScanner(os.Stdin)
	scanner.Buffer(make([]byte, 1<<20), 1<<20)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var quit shutdownRequest
		if json.Unmarshal([]byte(line), &quit) == nil && quit.Command == "quit" {
			os.Exit(0)
		}
		var req request
		if err := json.Unmarshal([]byte(line), &req); err != nil {
			writeResponse(response{Error: fmt.Sprintf("stub: bad request: %v", err)})
			continue
		}
		if strings.Contains(req.Prompt, "CRASH") {
			os.Exit(1) // crash during generation (AC-20)
		}
		start := time.Now()
		time.Sleep(time.Duration(prefillMs) * time.Millisecond)
		time.Sleep(time.Duration(max(0, req.MaxTokens-1)) * time.Duration(tokenMs) * time.Millisecond)
		writeResponse(response{
			Text:       responseText,
			TokensUsed: req.MaxTokens,
			DurationMs: int(time.Since(start).Milliseconds()),
		})
	}
	if err := scanner.Err(); err != nil {
		fmt.Fprintln(os.Stderr, "stub: stdin error:", err)
		os.Exit(1)
	}
}

func writeResponse(r response) {
	payload, err := json.Marshal(r)
	if err != nil {
		return
	}
	fmt.Println(string(payload))
	_ = os.Stdout.Sync()
}

func envInt(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}
