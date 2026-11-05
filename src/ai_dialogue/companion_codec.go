// Companion wire codec (INT-2 vertical slice): the game speaks
// newline-delimited JSON on stdin/stdout (ipc.go, ADR-0008) and the
// companion binary translates each request to the llama-server
// OpenAI-compatible completion API. The codec lives here — next to the
// protocol types it uses — so it is unit-testable without a running model;
// the process wiring is in ai_dialogue/companion/main.go.
//
// NOTE: the companion reaches llama-server over HTTP on localhost.
// ADR-0008 forbids HTTP for the game's own LLM calls; the companion keeps
// that contract (crash isolation, stdin/stdout for the game) at the cost of
// one localhost hop inside the companion process. Flagged for ADR-0008
// amendment review.
package aidialogue

import (
	"encoding/json"
	"fmt"
	"strings"
)

// CompanionQuitCommand is the out-of-band shutdown command's expected value.
const CompanionQuitCommand = "quit"

// CompanionParseLine decodes one stdin line from the game. It returns the
// request, whether the line was the out-of-band quit command, or a parse
// error. Empty lines are a no-op.
func CompanionParseLine(line []byte) (req PromptRequest, quit bool, err error) {
	line = []byte(strings.TrimSpace(string(line)))
	if len(line) == 0 {
		return req, false, nil
	}
	var probe struct {
		Command string `json:"command"`
	}
	if json.Unmarshal(line, &probe) == nil && probe.Command != "" {
		if probe.Command != CompanionQuitCommand {
			return req, false, fmt.Errorf("companion: unknown command %q", probe.Command)
		}
		return req, true, nil
	}
	if err := json.Unmarshal(line, &req); err != nil {
		return req, false, fmt.Errorf("companion: malformed request line: %w", err)
	}
	if req.Prompt == "" {
		return req, false, fmt.Errorf("companion: request with empty prompt")
	}
	if req.MaxTokens <= 0 {
		req.MaxTokens = 128
	}
	return req, false, nil
}

// companionCompletionRequest is the body posted to llama-server's
// OpenAI-compatible /v1/completions endpoint. The model field is
// informational — llama-server serves its single loaded model regardless of
// the value.
type companionCompletionRequest struct {
	Model       string  `json:"model"`
	Prompt      string  `json:"prompt"`
	MaxTokens   int     `json:"max_tokens"`
	Temperature float64 `json:"temperature"`
}

// companionCompletionResponse is the OpenAI-compatible reply shape.
type companionCompletionResponse struct {
	Choices []struct {
		Text string `json:"text"`
	} `json:"choices"`
	Usage struct {
		CompletionTokens int `json:"completion_tokens"`
	} `json:"usage"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

// CompanionBuildCompletionBody marshals the HTTP request body for a prompt
// request.
func CompanionBuildCompletionBody(req PromptRequest) ([]byte, error) {
	return json.Marshal(companionCompletionRequest{
		Model:       "marrow-npc",
		Prompt:      req.Prompt,
		MaxTokens:   req.MaxTokens,
		Temperature: req.Temp,
	})
}

// CompanionParseCompletionBody decodes the llama-server response into the
// game-facing PromptResponse. OpenAI-compatible servers often lead the
// completion text with a newline; it is stripped here so the response
// processor receives clean text.
func CompanionParseCompletionBody(body []byte, durationMs int) (PromptResponse, error) {
	var cr companionCompletionResponse
	if err := json.Unmarshal(body, &cr); err != nil {
		return PromptResponse{}, fmt.Errorf("companion: malformed completion response: %w", err)
	}
	if cr.Error != nil && cr.Error.Message != "" {
		return PromptResponse{Error: cr.Error.Message}, nil
	}
	if len(cr.Choices) == 0 {
		return PromptResponse{}, fmt.Errorf("companion: completion response with no choices")
	}
	return PromptResponse{
		Text:       strings.TrimPrefix(cr.Choices[0].Text, "\n"),
		TokensUsed: cr.Usage.CompletionTokens,
		DurationMs: durationMs,
	}, nil
}
