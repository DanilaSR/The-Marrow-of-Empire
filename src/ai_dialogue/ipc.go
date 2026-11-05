// IPC message types and the stdin/stdout JSON communication layer
// (ADR-0008 Key Interfaces, TR-ai-005, Story 001 Implementation Notes).
//
// Protocol: newline-delimited JSON. The game writes one PromptRequest line
// to the subprocess's stdin; the subprocess replies with exactly one
// PromptResponse line on stdout. The "quit" command (ShutdownRequest) is
// out-of-band: no response is expected, the subprocess exits cleanly.
package aidialogue

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"sync"
)

// PromptRequest is a generation request sent to the LLM subprocess as a
// single newline-delimited JSON line (ADR-0008).
type PromptRequest struct {
	Prompt    string  `json:"prompt"`
	MaxTokens int     `json:"max_tokens"`
	Temp      float64 `json:"temperature"`
}

// PromptResponse is the subprocess reply: one JSON line per request
// (ADR-0008).
type PromptResponse struct {
	Text       string `json:"text"`
	TokensUsed int    `json:"tokens_used"`
	DurationMs int    `json:"duration_ms"`
	Error      string `json:"error,omitempty"`
}

// ShutdownRequest is the out-of-band "quit" command sent on stdin before
// SIGTERM during graceful shutdown (Story 001 Implementation Notes). The
// subprocess exits with status 0 upon receiving it.
type ShutdownRequest struct {
	Command string `json:"command"`
}

// ipcResult is a response line or a transport error delivered to the
// waiting request.
type ipcResult struct {
	resp PromptResponse
	err  error
}

// conn is the stdin/stdout JSON pipe to the LLM subprocess.
//
// Concurrency model: requests are serialized by the caller (the Backend
// serves one request at a time), writes are guarded by writeMu, and a
// dedicated read-loop goroutine owns stdout. Responses are delivered to a
// single registered waiter. If the caller's context is cancelled while a
// request is in flight, the waiter is unregistered and the late response is
// discarded by the read loop — the pipe never desynchronizes.
type conn struct {
	stdin    io.WriteCloser
	read     *bufio.Reader
	writeMu  sync.Mutex
	waiterMu sync.Mutex
	waiter   chan ipcResult
	dead     chan struct{} // closed when the read loop exits (EOF/pipe error)
}

// newConn starts the read loop over stdout. The caller owns the pipes and
// must eventually call closeStdin (the read loop ends on EOF).
func newConn(stdin io.WriteCloser, stdout io.Reader) *conn {
	c := &conn{
		stdin: stdin,
		read:  bufio.NewReader(stdout),
		dead:  make(chan struct{}),
	}
	go c.readLoop()
	return c
}

// request writes req to the subprocess and waits for its response, honouring
// ctx cancellation. Returns ErrProcessExited if the subprocess died while
// waiting.
func (c *conn) request(ctx context.Context, req *PromptRequest) (PromptResponse, error) {
	payload, err := json.Marshal(req)
	if err != nil {
		return PromptResponse{}, fmt.Errorf("aidialogue: marshal request: %w", err)
	}
	waiter := make(chan ipcResult, 1)
	c.waiterMu.Lock()
	c.waiter = waiter
	c.waiterMu.Unlock()

	c.writeMu.Lock()
	_, writeErr := fmt.Fprintf(c.stdin, "%s\n", payload)
	c.writeMu.Unlock()
	if writeErr != nil {
		c.unregister(waiter)
		return PromptResponse{}, fmt.Errorf("%w: %v", ErrProcessExited, writeErr)
	}

	select {
	case res := <-waiter:
		return res.resp, res.err
	case <-c.dead:
		c.unregister(waiter)
		return PromptResponse{}, ErrProcessExited
	case <-ctx.Done():
		c.unregister(waiter)
		return PromptResponse{}, ctx.Err()
	}
}

// unregister drops the waiter if it is still the registered one. A cancelled
// request abandons its waiter; the read loop discards the late response.
func (c *conn) unregister(w chan ipcResult) {
	c.waiterMu.Lock()
	if c.waiter == w {
		c.waiter = nil
	}
	c.waiterMu.Unlock()
}

// deliver sends res to the registered waiter, if any. Never blocks.
func (c *conn) deliver(res ipcResult) {
	c.waiterMu.Lock()
	w := c.waiter
	c.waiterMu.Unlock()
	if w == nil {
		return
	}
	select {
	case w <- res:
	default:
	}
}

// readLoop owns the stdout pipe: each line is unmarshalled and delivered to
// the current waiter; lines with no waiter (cancelled requests) are
// discarded. On EOF or a read error the loop delivers ErrProcessExited and
// closes c.dead, releasing any pending request.
func (c *conn) readLoop() {
	for {
		line, err := c.read.ReadBytes('\n')
		if len(line) > 0 {
			var resp PromptResponse
			if jerr := json.Unmarshal(line, &resp); jerr == nil {
				c.deliver(ipcResult{resp: resp})
			} else {
				c.deliver(ipcResult{err: fmt.Errorf("aidialogue: malformed response line: %w", jerr)})
			}
		}
		if err != nil {
			c.deliver(ipcResult{err: ErrProcessExited})
			close(c.dead)
			return
		}
	}
}

// quit sends the out-of-band "quit" command. No response is expected; the
// subprocess exits cleanly.
func (c *conn) quit() error {
	payload, err := json.Marshal(ShutdownRequest{Command: "quit"})
	if err != nil {
		return err
	}
	c.writeMu.Lock()
	_, werr := fmt.Fprintf(c.stdin, "%s\n", payload)
	c.writeMu.Unlock()
	return werr
}

// closeStdin closes the subprocess's stdin, signalling end-of-input. Called
// during shutdown after the quit command; safe to call multiple times
// (subsequent calls return the pipe's close error or nil once closed).
func (c *conn) closeStdin() error {
	return c.stdin.Close()
}
