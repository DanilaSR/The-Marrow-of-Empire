// Config and model file discovery (Story 001 Implementation Notes: Model
// File Discovery; ADR-0008 Migration Plan: "Auto-detect model file at
// startup: check game directory, then $XDG_DATA_HOME/marrow-of-empire/models/").
package aidialogue

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/joho/godotenv"
)

// Config holds every tunable for the LLM backend. All gameplay values are
// data-driven (coding standards) — DefaultConfig provides the story-mandated
// values and callers override per deployment (tests inject a stub binary and
// temp model directories).
type Config struct {
	// BinaryPath is the path to the llama.cpp-based companion binary that
	// implements the stdin/stdout JSON protocol in ipc.go. The game ships
	// this wrapper alongside the model (ADR-0008: two binaries to
	// distribute). Integration tests inject a stub.
	BinaryPath string

	// ModelName is the GGUF filename to look for. Authoritative per the GDD
	// 2026-08-09 review: Fiendish_LLAMA_3B.Q4_K_M.gguf (Llama 3.2 3B RP
	// fine-tune, ~2.1 GB).
	ModelName string

	// ModelDirs is the search order for the model file, checked in order.
	// If empty, FindModelFile uses the defaults: the game executable's
	// directory, then $XDG_DATA_HOME/marrow-of-empire/models/ (ADR-0008
	// Migration Plan).
	ModelDirs []string

	// ExpectedSHA256 is the hex-encoded SHA-256 of the model file; empty
	// skips checksum validation (Story 001: checksum validation before
	// loading).
	ExpectedSHA256 string

	// Threads is the -threads argument passed to the subprocess. Reference
	// hardware: 8 threads (GDD CR-1).
	Threads int
	// ContextSize is the -ctx-size argument: 2048-token context window
	// (TR-ai-004).
	ContextSize int

	// MaxTokens is the default max_tokens for Generate requests (GDD target
	// response 50-150 tokens).
	MaxTokens int
	// Temperature is the default sampling temperature for Generate requests.
	Temperature float64

	// ExtraArgs are appended verbatim to the subprocess command line.
	ExtraArgs []string

	// DefaultGenerateTimeout is applied by Generate when the caller's
	// context carries no deadline. The GDD timeout is 25s (CR-5); per-NPC
	// timeout orchestration belongs to Story 004.
	DefaultGenerateTimeout time.Duration

	// Probe settings (GDD CR-1, Formulas § Latency Budget; Story 001
	// Implementation Notes). ProbePrompt is the ~150-token benchmark prompt.
	ProbePrompt string
	// ProbePromptTokens is the estimated token count of ProbePrompt. The
	// protocol does not report prompt token counts, so the probe uses this
	// configured estimate (a tuning knob).
	ProbePromptTokens int
	// ProbeDecodeTokens is the max_tokens of the second probe request; the
	// decode rate is derived from the duration difference between the two
	// probe runs (the prefill component cancels out).
	ProbeDecodeTokens int
	// ProbeResponseTokens is the reference max-length response (150 tokens)
	// used to model the cold-start and cached totals.
	ProbeResponseTokens int
	// MinDecodeTokPerSec is the decode-speed gate: decode ≥ 10 tok/s
	// (GDD CR-1; the threshold carries a 1.7× margin over the 6 tok/s needed
	// to complete a 150-token response within the 25s timeout).
	MinDecodeTokPerSec float64
	// MaxColdStartMs is the cold-start total gate: ≤ 25s × 0.85 = 21.25s
	// (Story 001 Implementation Notes).
	MaxColdStartMs int64
	// MaxCachedMs is the cached total gate: ≤ 12s (Story 001; GDD CR-7
	// target total warm-up).
	MaxCachedMs int64

	// RestartMax is the maximum number of subprocess restarts per session
	// (Story 001: max 1 restart per session; GDD CR-5).
	RestartMax int
	// RestartCooldown is the minimum interval between restarts (Story 001:
	// 5-min cooldown).
	RestartCooldown time.Duration

	// QuitGrace is how long Shutdown waits after the quit command before
	// signalling the process (SIGTERM on unix).
	QuitGrace time.Duration
	// TermGrace is how long Shutdown waits after the signal before SIGKILL.
	TermGrace time.Duration
}

// DefaultModelName is the authoritative GGUF filename (GDD 2026-08-09
// review: Fiendish_LLAMA_3B.Q4_K_M — Llama 3.2 3B RP fine-tune, Q4_K_M
// quantized, ~2.1 GB).
const DefaultModelName = "Fiendish_LLAMA_3B.Q4_K_M.gguf"

// defaultProbePrompt is the ~150-token benchmark prompt run at startup
// (GDD CR-1, CR-7 step 2). Neutral prose in the game's register — the probe
// measures the machine, not the model's dialogue skill.
const defaultProbePrompt = "You are the voice of a quiet land. Speak plainly and briefly. The old roads run from the mountain passes down to the salt coast, past villages that have stood for three hundred years and will stand for three hundred more. The harvest is in, the granaries are full, and the nights grow long. In the market square the traders argue over the price of wool and iron, while the watch keeps its lanterns lit until dawn. Answer with a few short sentences in the voice of a weathered innkeeper who has seen every season and knows the worth of a patient word."

// DefaultConfig returns the story-mandated defaults (Story 001, GDD CR-1,
// CR-7, Formulas § Latency Budget; TR-ai-004).
func DefaultConfig() Config {
	return Config{
		BinaryPath:             "marrow-llm", // companion wrapper shipped with the game
		ModelName:              DefaultModelName,
		Threads:                8,
		ContextSize:            2048,
		MaxTokens:              150,
		Temperature:            0.7,
		DefaultGenerateTimeout: 25 * time.Second,
		ProbePrompt:            defaultProbePrompt,
		ProbePromptTokens:      150,
		ProbeDecodeTokens:      8,
		ProbeResponseTokens:    150,
		MinDecodeTokPerSec:     10,
		MaxColdStartMs:         21250, // 25s × 0.85 margin
		MaxCachedMs:            12000,
		RestartMax:             1,
		RestartCooldown:        5 * time.Minute,
		QuitGrace:              2 * time.Second,
		TermGrace:              2 * time.Second,
	}
}

// FindModelFile locates the model file, checking Config.ModelDirs in order
// (default: the game executable's directory, then
// $XDG_DATA_HOME/marrow-of-empire/models/), and validates the SHA-256 when
// Config.ExpectedSHA256 is set. Returns ErrModelNotFound when the file does
// not exist anywhere (AC-15).
func FindModelFile(cfg Config) (string, error) {
	dirs := cfg.ModelDirs
	if len(dirs) == 0 {
		dirs = defaultModelDirs()
	}
	for _, dir := range dirs {
		path := filepath.Join(dir, cfg.ModelName)
		info, err := os.Stat(path)
		if err != nil || info.IsDir() {
			continue
		}
		if err := verifyChecksum(path, cfg.ExpectedSHA256); err != nil {
			return "", err
		}
		return path, nil
	}
	return "", fmt.Errorf("%w: searched %v", ErrModelNotFound, dirs)
}

// defaultModelDirs returns the standard search order: the game executable's
// directory first, then the platform data directory
// ($XDG_DATA_HOME/models, falling back to ~/.local/share/models).
func defaultModelDirs() []string {
	// .env is optional: it lets a local checkout point XDG_DATA_HOME at a
	// directory holding the multi-GB model files, which are never committed.
	_ = godotenv.Load()
	dirs := make([]string, 0, 2)
	if exe, err := os.Executable(); err == nil {
		dirs = append(dirs, filepath.Dir(exe))
	}
	dataHome := os.Getenv("XDG_DATA_HOME")
	if dataHome == "" {
		if home, err := os.UserHomeDir(); err == nil {
			dataHome = filepath.Join(home, ".local", "share")
		}
	}
	if dataHome != "" {
		dirs = append(dirs, filepath.Join(dataHome, "models"))
	}
	return dirs
}

// verifyChecksum streams the file through SHA-256 and compares it to the
// expected hex digest. An empty expected digest skips validation.
func verifyChecksum(path, expectedHex string) error {
	if expectedHex == "" {
		return nil
	}
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return fmt.Errorf("aidialogue: checksum read: %w", err)
	}
	got := hex.EncodeToString(h.Sum(nil))
	if got != expectedHex {
		return fmt.Errorf("%w: got %s, want %s", ErrChecksumMismatch, got, expectedHex)
	}
	return nil
}
