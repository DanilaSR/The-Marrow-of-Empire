// Companion binary for the Marrow of Empire LLM backend (ADR-0008).
//
// The game spawns this binary with --model/--ctx-size/--threads arguments
// and speaks newline-delimited JSON on stdin/stdout (aidialogue/ipc.go).
// The companion spawns llama-server with the same arguments, waits for it
// to become healthy, then bridges each prompt request to the
// OpenAI-compatible completion API. On the out-of-band "quit" command (or
// stdin EOF) it terminates llama-server and exits cleanly.
//
// Environment:
//
//	LLAMA_SERVER_BIN  path to the llama-server binary (default: found on PATH)
//	LLAMA_SERVER_PORT port for llama-server (default: 18080)
//
// INT-2 slice note: one localhost HTTP hop inside the companion process —
// see companion_codec.go for the ADR-0008 tension note.
package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	aidialogue "marrow-of-empire/ai_dialogue"

	"github.com/joho/godotenv"
)

func main() {
	// .env is optional; when present it supplies paths such as
	// LLAMA_SERVER_BIN and XDG_DATA_HOME for local model discovery.
	_ = godotenv.Load()
	model := flag.String("model", "", "GGUF model path (required)")
	ctxSize := flag.Int("ctx-size", 2048, "context window size")
	threads := flag.Int("threads", 8, "CPU threads")
	flag.Parse()

	if *model == "" {
		fmt.Fprintln(os.Stderr, "companion: --model is required")
		os.Exit(2)
	}
	if err := run(*model, *ctxSize, *threads); err != nil {
		fmt.Fprintf(os.Stderr, "companion: %v\n", err)
		os.Exit(1)
	}
}

func run(model string, ctxSize, threads int) error {
	port := 18080
	if p := os.Getenv("LLAMA_SERVER_PORT"); p != "" {
		if n, err := strconv.Atoi(p); err == nil && n > 0 {
			port = n
		}
	}
	serverBin := os.Getenv("LLAMA_SERVER_BIN")
	if serverBin == "" {
		serverBin = "llama-server"
	}
	if _, err := exec.LookPath(serverBin); err != nil {
		return fmt.Errorf("llama-server not found at %q (set LLAMA_SERVER_BIN): %w", serverBin, err)
	}

	baseURL := fmt.Sprintf("http://127.0.0.1:%d", port)

	// Spawn llama-server with the game's own arguments. stdout/stderr are
	// captured and only surfaced on failure — the game's stderr capture
	// (aidialogue/process.go) would otherwise fill with server logs.
	var serverLog bytes.Buffer
	server := exec.Command(serverBin,
		"--model", model,
		"--ctx-size", strconv.Itoa(ctxSize),
		"--threads", strconv.Itoa(threads),
		"--host", "127.0.0.1",
		"--port", strconv.Itoa(port),
		"--n-gpu-layers", "0",
	)
	server.Stdout = &serverLog
	server.Stderr = &serverLog
	if err := server.Start(); err != nil {
		return fmt.Errorf("spawn llama-server: %w", err)
	}
	stopped := make(chan struct{})
	go func() { _ = server.Wait(); close(stopped) }()

	// Terminate llama-server on exit paths: quit command, stdin EOF, or
	// SIGTERM/SIGINT from the game.
	shutdown := func() {
		select {
		case <-stopped:
		default:
			_ = server.Process.Signal(syscall.SIGTERM)
			select {
			case <-stopped:
			case <-time.After(10 * time.Second):
				_ = server.Process.Kill()
			}
		}
	}
	defer shutdown()
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)
	go func() {
		<-sigCh
		shutdown()
		os.Exit(0)
	}()

	// Wait for llama-server health (model load takes tens of seconds).
	if err := waitHealthy(baseURL, stopped, 180*time.Second); err != nil {
		return err
	}
	slog.Info("companion: llama-server ready", "model", model, "port", port)

	return serve(os.Stdin, os.Stdout, baseURL)
}

// waitHealthy polls llama-server's /health endpoint until it responds or the
// server process exits.
func waitHealthy(baseURL string, stopped <-chan struct{}, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	client := &http.Client{Timeout: 2 * time.Second}
	for time.Now().Before(deadline) {
		select {
		case <-stopped:
			return fmt.Errorf("llama-server exited during startup")
		default:
		}
		resp, err := client.Get(baseURL + "/health")
		if err == nil {
			_ = resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return nil
			}
		}
		time.Sleep(500 * time.Millisecond)
	}
	return fmt.Errorf("llama-server health check timed out after %s", timeout)
}

// serve runs the stdin/stdout bridge loop until quit or EOF.
func serve(stdin *os.File, stdout *os.File, baseURL string) error {
	reader := bufio.NewReader(stdin)
	writer := bufio.NewWriter(stdout)
	client := &http.Client{} // no client timeout: the game cancels its side and discards late responses

	for {
		line, err := reader.ReadBytes('\n')
		if len(line) > 0 {
			req, quit, perr := aidialogue.CompanionParseLine(line)
			if quit {
				return nil
			}
			if perr != nil {
				writeResponse(writer, aidialogue.PromptResponse{Error: perr.Error()})
				continue
			}
			if req.Prompt == "" {
				continue // empty line
			}
			writeResponse(writer, complete(client, baseURL, req))
		}
		if err != nil {
			return nil // EOF: game closed stdin
		}
	}
}

// complete performs one completion request against llama-server.
func complete(client *http.Client, baseURL string, req aidialogue.PromptRequest) aidialogue.PromptResponse {
	body, err := aidialogue.CompanionBuildCompletionBody(req)
	if err != nil {
		return aidialogue.PromptResponse{Error: err.Error()}
	}
	start := time.Now()
	httpResp, err := client.Post(baseURL+"/v1/completions", "application/json", bytes.NewReader(body))
	if err != nil {
		return aidialogue.PromptResponse{Error: fmt.Sprintf("completion request failed: %v", err)}
	}
	defer httpResp.Body.Close()
	var buf bytes.Buffer
	if _, err := buf.ReadFrom(httpResp.Body); err != nil {
		return aidialogue.PromptResponse{Error: fmt.Sprintf("read completion response: %v", err)}
	}
	if httpResp.StatusCode != http.StatusOK {
		return aidialogue.PromptResponse{Error: fmt.Sprintf("llama-server returned %d: %s",
			httpResp.StatusCode, strings.TrimSpace(buf.String()))}
	}
	resp, err := aidialogue.CompanionParseCompletionBody(buf.Bytes(), int(time.Since(start).Milliseconds()))
	if err != nil {
		return aidialogue.PromptResponse{Error: err.Error()}
	}
	return resp
}

func writeResponse(w *bufio.Writer, resp aidialogue.PromptResponse) {
	payload, _ := json.Marshal(resp)
	_, _ = w.Write(append(payload, '\n'))
	_ = w.Flush()
}
