// Unit tests for INT-2 companion wire codec (aidialogue.CompanionParseLine,
// CompanionBuildCompletionBody, CompanionParseCompletionBody).
//
// Evidence path: prototypes/marrow-of-empire-vertical-slice (integration story INT-2)
package aidialogue_test

import (
	"encoding/json"
	"strings"
	"testing"

	aidialogue "marrow-of-empire/ai_dialogue"
)

func TestCompanionParseLine_PromptRequest(t *testing.T) {
	req, quit, err := aidialogue.CompanionParseLine(
		[]byte(`{"prompt":"hello","max_tokens":64,"temperature":0.7}`))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if quit {
		t.Fatal("prompt line must not be quit")
	}
	if req.Prompt != "hello" || req.MaxTokens != 64 || req.Temp != 0.7 {
		t.Fatalf("request fields: %+v", req)
	}
}

func TestCompanionParseLine_QuitCommand(t *testing.T) {
	_, quit, err := aidialogue.CompanionParseLine([]byte(`{"command":"quit"}`))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if !quit {
		t.Fatal("quit command must set quit=true")
	}
}

func TestCompanionParseLine_UnknownCommand(t *testing.T) {
	_, quit, err := aidialogue.CompanionParseLine([]byte(`{"command":"frobnicate"}`))
	if err == nil {
		t.Fatal("unknown command must error")
	}
	if quit {
		t.Fatal("unknown command must not quit")
	}
}

func TestCompanionParseLine_EmptyLine(t *testing.T) {
	req, quit, err := aidialogue.CompanionParseLine([]byte("   \n"))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if quit || req.Prompt != "" {
		t.Fatal("empty line must be a no-op")
	}
}

func TestCompanionParseLine_Malformed(t *testing.T) {
	if _, _, err := aidialogue.CompanionParseLine([]byte(`{not json`)); err == nil {
		t.Fatal("malformed line must error")
	}
}

func TestCompanionParseLine_EmptyPromptRejected(t *testing.T) {
	if _, _, err := aidialogue.CompanionParseLine([]byte(`{"prompt":"","max_tokens":8}`)); err == nil {
		t.Fatal("empty prompt must error")
	}
}

func TestCompanionParseLine_ZeroMaxTokensDefaults(t *testing.T) {
	req, _, err := aidialogue.CompanionParseLine([]byte(`{"prompt":"x","max_tokens":0}`))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if req.MaxTokens <= 0 {
		t.Fatalf("max_tokens must default to a positive cap, got %d", req.MaxTokens)
	}
}

func TestCompanionBuildCompletionBody_Shape(t *testing.T) {
	body, err := aidialogue.CompanionBuildCompletionBody(aidialogue.PromptRequest{
		Prompt: "say hi", MaxTokens: 42, Temp: 0.5,
	})
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(body, &decoded); err != nil {
		t.Fatalf("body is not JSON: %v", err)
	}
	if decoded["prompt"] != "say hi" || decoded["max_tokens"] != float64(42) ||
		decoded["temperature"] != 0.5 || decoded["model"] != "marrow-npc" {
		t.Fatalf("body fields: %v", decoded)
	}
}

func TestCompanionParseCompletionBody_Normal(t *testing.T) {
	resp, err := aidialogue.CompanionParseCompletionBody(
		[]byte(`{"choices":[{"text":"\nThe ore is wrong.","index":0}],"usage":{"completion_tokens":12}}`),
		1234,
	)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if resp.Text != "The ore is wrong." {
		t.Fatalf("leading newline must be stripped, got %q", resp.Text)
	}
	if resp.TokensUsed != 12 || resp.DurationMs != 1234 {
		t.Fatalf("usage/duration: %+v", resp)
	}
}

func TestCompanionParseCompletionBody_ServerError(t *testing.T) {
	resp, err := aidialogue.CompanionParseCompletionBody(
		[]byte(`{"error":{"message":"slot unavailable"}}`), 0)
	if err != nil {
		t.Fatalf("server errors surface in the response body, not as parse errors: %v", err)
	}
	if !strings.Contains(resp.Error, "slot unavailable") {
		t.Fatalf("error must be carried in the response: %+v", resp)
	}
}

func TestCompanionParseCompletionBody_NoChoices(t *testing.T) {
	if _, err := aidialogue.CompanionParseCompletionBody([]byte(`{"choices":[]}`), 0); err == nil {
		t.Fatal("empty choices must error")
	}
}
