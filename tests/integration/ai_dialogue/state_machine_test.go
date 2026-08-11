package aidialogue_test

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"marrow-of-empire/ai_dialogue"
)

// stubBackend is a test double for LLMBackend.
type stubBackend struct {
	mu       sync.Mutex
	status   aidialogue.ModelStatus
	response string
	err      error
	// blockCh, if set, blocks Generate until closed or ctx cancelled.
	blockCh    chan struct{}
	callCount  int32
	lastPrompt string
	onGenerate func()
}

func newStubBackend(status aidialogue.ModelStatus) *stubBackend {
	return &stubBackend{status: status}
}

func (s *stubBackend) Generate(ctx context.Context, prompt string) (string, error) {
	atomic.AddInt32(&s.callCount, 1)
	s.mu.Lock()
	s.lastPrompt = prompt
	blockCh := s.blockCh
	s.mu.Unlock()

	if s.onGenerate != nil {
		s.onGenerate()
	}

	if blockCh != nil {
		select {
		case <-blockCh:
		case <-ctx.Done():
			return "", ctx.Err()
		}
	}

	if ctx.Err() != nil {
		return "", ctx.Err()
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	return s.response, s.err
}

func (s *stubBackend) Status() aidialogue.ModelStatus {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.status
}

func (s *stubBackend) Shutdown() error { return nil }

func (s *stubBackend) setBlockCh(ch chan struct{}) {
	s.mu.Lock(); s.blockCh = ch; s.mu.Unlock()
}

func (s *stubBackend) setResponse(r string, err error) {
	s.mu.Lock(); s.response = r; s.err = err; s.mu.Unlock()
}

func (s *stubBackend) setStatus(st aidialogue.ModelStatus) {
	s.mu.Lock(); s.status = st; s.mu.Unlock()
}

// ────────────────────────────────────────────────────────────────
// AC-17: Timeout → text fallback → retry
// ────────────────────────────────────────────────────────────────

func TestTimeout_FallsBackToTextMode_IncrementsCounter(t *testing.T) {
	backend := newStubBackend(aidialogue.StatusReady)
	blockCh := make(chan struct{})
	backend.setBlockCh(blockCh)

	// Short timeout so the test runs fast — context deadline fires after 50ms.
	sm := aidialogue.NewStateMachine(backend, 1,
		aidialogue.WithTimeoutThreshold(50*time.Millisecond),
	)

	resp, err := sm.Submit(context.Background(), "test question")
	close(blockCh) // clean up

	// Timeout produces text fallback, not an error.
	if err != nil {
		t.Errorf("timeout should produce text fallback, not error: %v", err)
	}
	if sm.State() != aidialogue.StateTextMode {
		t.Errorf("after timeout, state = %v, want StateTextMode", sm.State())
	}
	if !sm.IsTextMode() {
		t.Error("IsTextMode() should be true after timeout")
	}
	if resp == "" {
		t.Error("text fallback response should not be empty")
	}
}

func TestTimeout_NextQueryRetriesNormally_ReturnsToReady(t *testing.T) {
	backend := newStubBackend(aidialogue.StatusReady)

	// First query: timeout (blocked backend).
	blockCh := make(chan struct{})
	backend.setBlockCh(blockCh)
	sm := aidialogue.NewStateMachine(backend, 1,
		aidialogue.WithTimeoutThreshold(50*time.Millisecond),
	)
	sm.Submit(context.Background(), "q1")
	close(blockCh)
	time.Sleep(10 * time.Millisecond)

	if sm.State() != aidialogue.StateTextMode {
		t.Fatalf("after timeout, state = %v, want StateTextMode", sm.State())
	}

	// Second query: backend ready, should succeed and return to Ready.
	backend.setBlockCh(nil)
	backend.setResponse("Hello, traveler!", nil)
	sm.Submit(context.Background(), "q2")

	if sm.State() != aidialogue.StateReady {
		t.Errorf("after successful retry, state = %v, want StateReady", sm.State())
	}
}

func TestTimeout_PerNPCIsolation(t *testing.T) {
	// NPC A: timeout.
	blockChA := make(chan struct{})
	backendA := newStubBackend(aidialogue.StatusReady)
	backendA.setBlockCh(blockChA)
	smA := aidialogue.NewStateMachine(backendA, 1,
		aidialogue.WithTimeoutThreshold(50*time.Millisecond),
	)

	// NPC B: normal operation.
	backendB := newStubBackend(aidialogue.StatusReady)
	backendB.setResponse("Greetings from NPC B!", nil)
	smB := aidialogue.NewStateMachine(backendB, 2,
		aidialogue.WithTimeoutThreshold(25*time.Second),
	)

	// Trigger timeout on NPC A.
	smA.Submit(context.Background(), "q")
	close(blockChA)
	time.Sleep(10 * time.Millisecond)

	if !smA.IsTextMode() {
		t.Error("NPC A should be in TextMode after timeout")
	}

	// NPC B should still work normally.
	resp, err := smB.Submit(context.Background(), "hello")
	if err != nil {
		t.Errorf("NPC B should not error: %v", err)
	}
	if smB.State() != aidialogue.StateReady {
		t.Errorf("NPC B state = %v, want StateReady", smB.State())
	}
	_ = resp
}

func TestTimeout_ThreeTimeouts_SessionTextMode_NoMoreRetries(t *testing.T) {
	backend := newStubBackend(aidialogue.StatusReady)

	sm := aidialogue.NewStateMachine(backend, 1,
		aidialogue.WithTimeoutThreshold(50*time.Millisecond),
	)

	// Trigger 3 timeouts by blocking the backend each time.
	for i := 0; i < 3; i++ {
		blockCh := make(chan struct{})
		backend.setBlockCh(blockCh)
		sm.Submit(context.Background(), "q")
		close(blockCh)
		time.Sleep(15 * time.Millisecond)
	}

	// After 3 timeouts, session-stable text mode: no inference requests.
	backend.setBlockCh(nil)
	backend.setResponse("Should not be called", nil)
	sm.Submit(context.Background(), "another question")

	if sm.State() != aidialogue.StateTextMode {
		t.Errorf("after 3 timeouts, state = %v, want StateTextMode (session-stable)", sm.State())
	}
}

// ────────────────────────────────────────────────────────────────
// AC-18: Esc cancel
// ────────────────────────────────────────────────────────────────

func TestCancel_GenerationStops_StateReturnsToReady(t *testing.T) {
	backend := newStubBackend(aidialogue.StatusReady)
	blockCh := make(chan struct{})
	backend.setBlockCh(blockCh)

	sm := aidialogue.NewStateMachine(backend, 1)

	go func() { sm.Submit(context.Background(), "question") }()
	time.Sleep(20 * time.Millisecond)

	sm.Cancel()

	if sm.State() != aidialogue.StateReady {
		t.Errorf("after cancel, state = %v, want StateReady", sm.State())
	}

	close(blockCh)
}

func TestCancel_NotGenerating_IsNoop(t *testing.T) {
	backend := newStubBackend(aidialogue.StatusReady)
	sm := aidialogue.NewStateMachine(backend, 1)
	sm.Cancel()
	if sm.State() != aidialogue.StateReady {
		t.Errorf("cancel while Ready should stay Ready, got %v", sm.State())
	}
}

// ────────────────────────────────────────────────────────────────
// AC-19: New question cancels current
// ────────────────────────────────────────────────────────────────

func TestSubmit_DuringGeneration_CancelsCurrent_StartsNew(t *testing.T) {
	backend := newStubBackend(aidialogue.StatusReady)
	firstBlock := make(chan struct{})
	backend.setBlockCh(firstBlock)
	backend.setResponse("first response", nil)

	sm := aidialogue.NewStateMachine(backend, 1)

	go func() { sm.Submit(context.Background(), "q1") }()
	time.Sleep(20 * time.Millisecond)

	backend.setBlockCh(nil)
	backend.setResponse("second response", nil)
	resp, err := sm.Submit(context.Background(), "q2")
	close(firstBlock)

	if err != nil {
		t.Errorf("second submit should succeed: %v", err)
	}
	if resp != "second response" {
		t.Errorf("got response %q, want %q", resp, "second response")
	}
}

func TestSubmit_QueueDepthNeverExceedsOne(t *testing.T) {
	backend := newStubBackend(aidialogue.StatusReady)
	blockCh := make(chan struct{})
	backend.setBlockCh(blockCh)
	backend.setResponse("final", nil)

	sm := aidialogue.NewStateMachine(backend, 1)

	for i := 0; i < 4; i++ {
		go func() { sm.Submit(context.Background(), "intermediate") }()
		time.Sleep(5 * time.Millisecond)
	}

	backend.setBlockCh(nil)
	resp, err := sm.Submit(context.Background(), "final question")

	close(blockCh)
	time.Sleep(10 * time.Millisecond)

	if err != nil {
		t.Errorf("final submit should succeed: %v", err)
	}
	if resp != "final" {
		t.Errorf("got %q, want %q", resp, "final")
	}
}

// ────────────────────────────────────────────────────────────────
// BUG 1 regression: stale-generation ABA race
// ────────────────────────────────────────────────────────────────

func TestRegression_StaleGeneration_DoesNotCorruptState(t *testing.T) {
	backend := newStubBackend(aidialogue.StatusReady)

	chA := make(chan struct{})
	chB := make(chan struct{})
	backend.setBlockCh(chA)
	backend.setResponse("a", nil)

	sm := aidialogue.NewStateMachine(backend, 1,
		aidialogue.WithTimeoutThreshold(5*time.Second),
	)

	// Submit A — blocks on chA.
	var respA string
	var errA error
	doneA := make(chan struct{})
	go func() {
		respA, errA = sm.Submit(context.Background(), "qA")
		close(doneA)
	}()
	time.Sleep(20 * time.Millisecond)

	// Submit B — cancels A, blocks on chB. Run in goroutine so we can unblock.
	backend.setBlockCh(chB)
	backend.setResponse("b", nil)
	doneB := make(chan struct{})
	var respB string
	var errB error
	go func() {
		respB, errB = sm.Submit(context.Background(), "qB")
		close(doneB)
	}()
	time.Sleep(20 * time.Millisecond)

	// Unblock B first — B completes successfully.
	close(chB)
	select {
	case <-doneB:
	case <-time.After(time.Second):
		t.Fatal("Submit B did not complete")
	}

	if errB != nil {
		t.Fatalf("Submit B should succeed: %v", errB)
	}
	if respB != "b" {
		t.Errorf("Submit B response = %q, want %q", respB, "b")
	}

	// Unblock A — its token is stale, completion must be a no-op.
	close(chA)
	select {
	case <-doneA:
	case <-time.After(time.Second):
		t.Fatal("Submit A goroutine did not complete")
	}

	// A must be superseded.
	if errA == nil {
		t.Error("Submit A should be superseded by B (stale generation)")
	}
	if respA != "" {
		t.Errorf("Submit A response = %q, want empty (superseded)", respA)
	}

	// State must remain Ready — A's stale completion must not corrupt it.
	if sm.State() != aidialogue.StateReady {
		t.Errorf("after stale completion, state = %v, want StateReady", sm.State())
	}

	// Cancel must still work (no-op).
	sm.Cancel()
	if sm.State() != aidialogue.StateReady {
		t.Errorf("Cancel after ABA: state = %v, want StateReady", sm.State())
	}
}

// ────────────────────────────────────────────────────────────────
// BUG 2 regression: timeout classification
// ────────────────────────────────────────────────────────────────

func TestRegression_DeadlineExceeded_IncrementsCounter(t *testing.T) {
	backend := newStubBackend(aidialogue.StatusReady)
	// Stub returns context.DeadlineExceeded directly — simulates
	// a real context deadline firing.
	backend.setResponse("", context.DeadlineExceeded)

	sm := aidialogue.NewStateMachine(backend, 1,
		aidialogue.WithTimeoutThreshold(1*time.Millisecond),
	)

	// Let the real deadline fire (1ms timeout).
	time.Sleep(10 * time.Millisecond)
	resp, err := sm.Submit(context.Background(), "test")

	// Must NOT be an error — timeout should produce text fallback.
	if err != nil {
		t.Errorf("DeadlineExceeded should produce fallback, got error: %v", err)
	}
	if resp == "" {
		t.Error("text fallback response should not be empty")
	}
	// State must be TextMode, not Ready.
	if sm.State() != aidialogue.StateTextMode {
		t.Errorf("after DeadlineExceeded, state = %v, want StateTextMode", sm.State())
	}
}

func TestRegression_Canceled_ReturnsToReady_NoCounterChange(t *testing.T) {
	backend := newStubBackend(aidialogue.StatusReady)
	blockCh := make(chan struct{})
	backend.setBlockCh(blockCh)

	sm := aidialogue.NewStateMachine(backend, 1,
		aidialogue.WithTimeoutThreshold(5*time.Second),
	)

	go func() { sm.Submit(context.Background(), "q") }()
	time.Sleep(20 * time.Millisecond)

	// Cancel — this should return state to Ready, NOT text mode.
	sm.Cancel()
	close(blockCh)
	time.Sleep(20 * time.Millisecond)

	if sm.State() != aidialogue.StateReady {
		t.Errorf("after Cancel, state = %v, want StateReady (not TextMode)", sm.State())
	}
	if sm.IsTextMode() {
		t.Error("Cancel should not trigger text mode")
	}
}

// ────────────────────────────────────────────────────────────────
// AC-27: Save/load round-trip
// ────────────────────────────────────────────────────────────────

func TestConversationMemory_Snapshot_Restore_RoundTrip(t *testing.T) {
	mem := &aidialogue.ConversationMemory{}
	mem.RecordExchange("What do you know about the mill?", "The old mill burned down last winter.")
	mem.RecordDisclosed("fact_mill_001")
	mem.RecordDisclosed("fact_mill_002")

	snap := mem.Snapshot()

	restored := &aidialogue.ConversationMemory{}
	restored.Restore(snap)

	if restored.ExchangeCount != 1 {
		t.Errorf("ExchangeCount = %d, want 1", restored.ExchangeCount)
	}
	if len(restored.DisclosedFacts) != 2 {
		t.Errorf("DisclosedFacts len = %d, want 2", len(restored.DisclosedFacts))
	}
	if restored.RollingSummary == "" {
		t.Error("RollingSummary should not be empty after restore")
	}
}

func TestConversationMemory_GapAcknowledgement(t *testing.T) {
	mem := &aidialogue.ConversationMemory{}
	mem.RecordExchange("hello", "Greetings, traveler.")

	if mem.GapAcknowledgement() == "" {
		t.Error("GapAcknowledgement should not be empty when ExchangeCount > 0")
	}

	empty := &aidialogue.ConversationMemory{}
	if empty.GapAcknowledgement() != "" {
		t.Error("GapAcknowledgement should be empty for fresh memory")
	}
}

func TestConversationMemory_RollingSummary_Truncation(t *testing.T) {
	mem := &aidialogue.ConversationMemory{}
	longQ := "This is a very long question that takes up significant space."
	longR := "This is a very long response that also takes up space."

	for i := 0; i < 20; i++ {
		mem.RecordExchange(longQ, longR)
	}

	if len(mem.RollingSummary) > 1000 {
		t.Errorf("RollingSummary length = %d, want ≤ ~1000", len(mem.RollingSummary))
	}
	if mem.RollingSummary == "" {
		t.Error("RollingSummary should not be empty after truncation")
	}
}

func TestConversationMemory_RecordDisclosed_NoDuplicates(t *testing.T) {
	mem := &aidialogue.ConversationMemory{}
	mem.RecordDisclosed("fact_001")
	mem.RecordDisclosed("fact_001")

	if len(mem.DisclosedFacts) != 1 {
		t.Errorf("DisclosedFacts len = %d, want 1", len(mem.DisclosedFacts))
	}
}

func TestConversationMemory_SnapshotIsDeepCopy(t *testing.T) {
	mem := &aidialogue.ConversationMemory{}
	mem.RecordDisclosed("fact_001")
	snap := mem.Snapshot()

	mem.RecordDisclosed("fact_002")
	mem.RecordExchange("q", "r")

	if len(snap.DisclosedFacts) != 1 {
		t.Errorf("snapshot len = %d, want 1 (deep copy)", len(snap.DisclosedFacts))
	}
}

func TestStateMachine_Memory_SaveRestore_RoundTrip(t *testing.T) {
	backend := newStubBackend(aidialogue.StatusReady)
	backend.setResponse("The mill burned down last winter.", nil)
	sm := aidialogue.NewStateMachine(backend, 1)

	sm.Submit(context.Background(), "Tell me about the mill")
	mem := sm.Memory()

	if mem.ExchangeCount != 1 {
		t.Errorf("ExchangeCount = %d, want 1", mem.ExchangeCount)
	}

	sm2 := aidialogue.NewStateMachine(backend, 1)
	sm2.RestoreMemory(mem)

	mem2 := sm2.Memory()
	if mem2.ExchangeCount != 1 {
		t.Errorf("restored ExchangeCount = %d, want 1", mem2.ExchangeCount)
	}
	if mem2.RollingSummary != mem.RollingSummary {
		t.Error("RollingSummary should match after restore")
	}
}

// ────────────────────────────────────────────────────────────────
// AC-31: Concurrency
// ────────────────────────────────────────────────────────────────

func TestConcurrency_RapidQuestionsAndCancels_StateInvariants(t *testing.T) {
	backend := newStubBackend(aidialogue.StatusReady)
	backend.setResponse("ok", nil)
	backend.onGenerate = func() {
		time.Sleep(5 * time.Millisecond)
	}

	sm := aidialogue.NewStateMachine(backend, 1,
		aidialogue.WithTimeoutThreshold(5*time.Second),
	)

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			sm.Submit(context.Background(), "rapid question")
		}()
	}
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			time.Sleep(2 * time.Millisecond)
			sm.Cancel()
		}()
	}

	wg.Wait()
	time.Sleep(20 * time.Millisecond)

	state := sm.State()
	if state != aidialogue.StateReady && state != aidialogue.StateTextMode {
		t.Errorf("final state = %v, want StateReady or StateTextMode", state)
	}
}

func TestConcurrency_NeverGeneratingWithoutActiveRequest(t *testing.T) {
	backend := newStubBackend(aidialogue.StatusReady)
	sm := aidialogue.NewStateMachine(backend, 1)

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); sm.Submit(context.Background(), "q") }()
		wg.Add(1)
		go func() { defer wg.Done(); sm.Cancel() }()
	}
	wg.Wait()
	time.Sleep(20 * time.Millisecond)

	sm.Close()
	if sm.State() == aidialogue.StateGenerating {
		t.Error("state should not be Generating after Close")
	}
}

func TestConcurrency_NoOrphanedGenerationAfterCancel(t *testing.T) {
	backend := newStubBackend(aidialogue.StatusReady)
	blockCh := make(chan struct{})
	backend.setBlockCh(blockCh)
	backend.setResponse("response", nil)

	sm := aidialogue.NewStateMachine(backend, 1,
		aidialogue.WithTimeoutThreshold(5*time.Second),
	)

	go func() { sm.Submit(context.Background(), "question") }()
	time.Sleep(20 * time.Millisecond)

	sm.Cancel()
	close(blockCh)
	time.Sleep(20 * time.Millisecond)

	if sm.State() != aidialogue.StateReady {
		t.Errorf("after cancel + unblock, state = %v, want StateReady", sm.State())
	}
}

// ────────────────────────────────────────────────────────────────
// Basics
// ────────────────────────────────────────────────────────────────

func TestNewStateMachine_InitialStateFromBackend(t *testing.T) {
	tests := []struct {
		status aidialogue.ModelStatus
		want   aidialogue.ConversationState
	}{
		{aidialogue.StatusUnloaded, aidialogue.StateUnloaded},
		{aidialogue.StatusLoading, aidialogue.StateLoading},
		{aidialogue.StatusReady, aidialogue.StateReady},
		{aidialogue.StatusDegraded, aidialogue.StateTextMode},
		{aidialogue.StatusError, aidialogue.StateTextMode},
	}

	for _, tt := range tests {
		backend := newStubBackend(tt.status)
		sm := aidialogue.NewStateMachine(backend, 1)
		if sm.State() != tt.want {
			t.Errorf("backend status %v → state %v, want %v", tt.status, sm.State(), tt.want)
		}
	}
}

func TestClose_ResetsCounters(t *testing.T) {
	backend := newStubBackend(aidialogue.StatusReady)

	sm := aidialogue.NewStateMachine(backend, 1,
		aidialogue.WithTimeoutThreshold(50*time.Millisecond),
	)

	blockCh := make(chan struct{})
	backend.setBlockCh(blockCh)
	sm.Submit(context.Background(), "q")
	close(blockCh)
	time.Sleep(10 * time.Millisecond)

	sm.Close()

	mem := sm.Memory()
	if mem.ExchangeCount != 0 {
		t.Errorf("ExchangeCount after Close = %d, want 0", mem.ExchangeCount)
	}
}

func TestBackendError_FallsBackToTextMode(t *testing.T) {
	backend := newStubBackend(aidialogue.StatusReady)
	backend.setResponse("", aidialogue.ErrModelCrashed)
	sm := aidialogue.NewStateMachine(backend, 1)

	resp, err := sm.Submit(context.Background(), "question")
	if err != nil {
		t.Logf("backend error should produce fallback, got: %v", err)
	}
	if resp == "" {
		t.Error("text fallback response should not be empty")
	}
	if sm.State() != aidialogue.StateTextMode {
		t.Errorf("after backend error, state = %v, want StateTextMode", sm.State())
	}
}
