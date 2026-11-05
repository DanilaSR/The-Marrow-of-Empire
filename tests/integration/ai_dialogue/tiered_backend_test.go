package aidialogue_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"marrow-of-empire/ai_dialogue"
)

// fastStub is a stub LLMBackend that returns quickly (simulates tiny model).
type fastStub struct {
	status   aidialogue.ModelStatus
	response string
	err      error
	callCount int32
}

func (s *fastStub) Generate(ctx context.Context, prompt string) (string, error) {
	s.callCount++
	return s.response, s.err
}
func (s *fastStub) Status() aidialogue.ModelStatus { return s.status }
func (s *fastStub) Shutdown() error                 { return nil }

// fullStub is a stub LLMBackend for the primary model.
type fullStub struct {
	status    aidialogue.ModelStatus
	response  string
	err       error
	callCount int32
}

func (s *fullStub) Generate(ctx context.Context, prompt string) (string, error) {
	s.callCount++
	return s.response, s.err
}
func (s *fullStub) Status() aidialogue.ModelStatus { return s.status }
func (s *fullStub) Shutdown() error                 { return nil }

// ────────────────────────────────────────────────────────────────
// AC-1: Tiny model returns deflection in <100ms
// ────────────────────────────────────────────────────────────────

func TestTinyCGoBackend_ReturnsDeflection_Under100ms(t *testing.T) {
	tb := aidialogue.NewTinyCGoBackend("test-model.gguf")
	if err := tb.Load(); err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if tb.Status() != aidialogue.StatusReady {
		t.Fatalf("status after Load = %v, want Ready", tb.Status())
	}

	start := time.Now()
	resp, err := tb.Generate(context.Background(), "What do you think?")
	elapsed := time.Since(start)

	if err != nil {
		t.Errorf("Generate should succeed: %v", err)
	}
	if resp == "" {
		t.Error("response should not be empty")
	}
	if elapsed > 100*time.Millisecond {
		t.Errorf("Generate took %v, want <100ms", elapsed)
	}
	if tb.Status() != aidialogue.StatusReady {
		t.Errorf("status after Generate = %v, want Ready", tb.Status())
	}
}

// ────────────────────────────────────────────────────────────────
// AC-2: Complex query routed to IPC primary
// ────────────────────────────────────────────────────────────────

func TestTieredBackend_ComplexQuery_RoutesToPrimary(t *testing.T) {
	fast := &fastStub{status: aidialogue.StatusReady, response: "fast"}
	full := &fullStub{status: aidialogue.StatusReady, response: "full complex response"}

	tb := aidialogue.NewTieredBackend(fast, full)

	// Complex prompt: domain-matched, entity references.
	resp, err := tb.Generate(context.Background(),
		"What can you tell me about the old mill near Greyhaven?")

	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}
	if resp != "full complex response" {
		t.Errorf("response = %q, want %q (should route to primary)", resp, "full complex response")
	}
	if full.callCount != 1 {
		t.Errorf("primary callCount = %d, want 1", full.callCount)
	}
	// Fast should NOT have been called for a complex query.
	if fast.callCount != 0 {
		t.Errorf("tiny model callCount = %d, want 0 (complex query)", fast.callCount)
	}
}

// ────────────────────────────────────────────────────────────────
// AC-3: Simple query routed to tiny CGo backend
// ────────────────────────────────────────────────────────────────

func TestTieredBackend_SimpleQuery_RoutesToTiny(t *testing.T) {
	fast := &fastStub{status: aidialogue.StatusReady, response: "I don't know much about that."}
	full := &fullStub{status: aidialogue.StatusReady, response: "full"}

	tb := aidialogue.NewTieredBackend(fast, full)

	// Simple query: empty facts, deflection — use ClassifyResult with IsFallback.
	resp, err := tb.Generate(context.Background(), "hello")

	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}
	// Without topic-matcher metadata, "hello" is 1 token, no entities → simple.
	if resp != fast.response {
		t.Errorf("response = %q, want %q (should route to tiny model)", resp, fast.response)
	}
	if fast.callCount != 1 {
		t.Errorf("tiny model callCount = %d, want 1", fast.callCount)
	}
}

func TestTieredBackend_IsFallbackQuery_RoutesToTiny(t *testing.T) {
	fast := &fastStub{status: aidialogue.StatusReady, response: "fast fallback"}
	full := &fullStub{status: aidialogue.StatusReady, response: "full"}

	tb := aidialogue.NewTieredBackend(fast, full)

	// Simulate topic-matcher returning IsFallback.
	shouldTry := tb.ShouldTryFast("any question about unknown topic",
		aidialogue.ClassifyResult{IsFallback: true},
	)
	if !shouldTry {
		t.Error("ShouldTryFast with IsFallback should return true")
	}
}

func TestTieredBackend_NoDomainNoEntities_RoutesToTiny(t *testing.T) {
	fast := &fastStub{status: aidialogue.StatusReady, response: "greeting"}
	full := &fullStub{status: aidialogue.StatusReady, response: "full"}

	tb := aidialogue.NewTieredBackend(fast, full)

	// DomainCount=0, EntityRefCount=0 → greeting / conversational opener.
	resp, err := tb.Generate(context.Background(), "Good day to you.")

	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}
	if resp != "greeting" {
		t.Errorf("response = %q, want %q (greeting→tiny model)", resp, "greeting")
	}
}

// ────────────────────────────────────────────────────────────────
// AC-4: CGo panic → recover → fallback to primary
// ────────────────────────────────────────────────────────────────

func TestTieredBackend_TinyModelFails_FallsBackToPrimary(t *testing.T) {
	fast := &fastStub{
		status: aidialogue.StatusReady,
		err:    errors.New("simulated tiny model failure"),
	}
	full := &fullStub{status: aidialogue.StatusReady, response: "primary handled it"}

	tb := aidialogue.NewTieredBackend(fast, full)

	// Simple prompt would normally route to fast, but fast fails → primary.
	resp, err := tb.Generate(context.Background(), "hello")

	if err != nil {
		t.Fatalf("Generate should succeed via primary: %v", err)
	}
	if resp != "primary handled it" {
		t.Errorf("response = %q, want %q (fallback to primary)", resp, "primary handled it")
	}
	if full.callCount != 1 {
		t.Errorf("primary callCount = %d, want 1", full.callCount)
	}
}

func TestTinyCGoBackend_PanicRecovery_Isolation(t *testing.T) {
	tb := aidialogue.NewTinyCGoBackend("test.gguf")
	if err := tb.Load(); err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	// The stub Generate wraps the call in recover(). Calling it works as expected.
	resp, err := tb.Generate(context.Background(), "hello")
	if err != nil {
		t.Fatalf("stub Generate should succeed: %v", err)
	}
	if resp == "" {
		t.Error("response should not be empty")
	}
	// Status should return to Ready after generation.
	if tb.Status() != aidialogue.StatusReady {
		t.Errorf("status after stub Generate = %v, want Ready", tb.Status())
	}
}

// ────────────────────────────────────────────────────────────────
// AC-5: Tiny model unavailable → invisible failover
// ────────────────────────────────────────────────────────────────

func TestTieredBackend_TinyModelNil_AllQueriesRouteToPrimary(t *testing.T) {
	full := &fullStub{status: aidialogue.StatusReady, response: "primary only"}

	// fast == nil — tiny model not available.
	tb := aidialogue.NewTieredBackend(nil, full)

	resp, err := tb.Generate(context.Background(), "hello")
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}
	if resp != "primary only" {
		t.Errorf("response = %q, want 'primary only'", resp)
	}
	if full.callCount != 1 {
		t.Errorf("primary callCount = %d, want 1", full.callCount)
	}

	// Status with nil fast + Ready full → Ready.
	if tb.Status() != aidialogue.StatusReady {
		t.Errorf("Status = %v, want Ready", tb.Status())
	}
}

func TestTieredBackend_TinyModelDegraded_InvisibleFailover(t *testing.T) {
	fast := &fastStub{status: aidialogue.StatusDegraded}
	full := &fullStub{status: aidialogue.StatusReady, response: "primary"}

	tb := aidialogue.NewTieredBackend(fast, full)

	// Simple prompt — would route to fast, but fast is degraded, so primary.
	resp, err := tb.Generate(context.Background(), "hello")
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}
	if resp != "primary" {
		t.Errorf("response = %q, want 'primary' (invisible failover)", resp)
	}
}

func TestTieredBackend_BothDegraded_StatusIsDegraded(t *testing.T) {
	fast := &fastStub{status: aidialogue.StatusDegraded}
	full := &fullStub{status: aidialogue.StatusDegraded}

	tb := aidialogue.NewTieredBackend(fast, full)

	if tb.Status() != aidialogue.StatusDegraded {
		t.Errorf("Status = %v, want Degraded", tb.Status())
	}
}

// ────────────────────────────────────────────────────────────────
// TieredBackend Status composition
// ────────────────────────────────────────────────────────────────

func TestTieredBackend_Status_FullReady_FastLoading(t *testing.T) {
	fast := &fastStub{status: aidialogue.StatusLoading}
	full := &fullStub{status: aidialogue.StatusReady}

	tb := aidialogue.NewTieredBackend(fast, full)
	if tb.Status() != aidialogue.StatusReady {
		t.Errorf("Status = %v, want Ready (full is ready)", tb.Status())
	}
}

func TestTieredBackend_Status_FastReady_FullLoading(t *testing.T) {
	fast := &fastStub{status: aidialogue.StatusReady}
	full := &fullStub{status: aidialogue.StatusLoading}

	tb := aidialogue.NewTieredBackend(fast, full)
	if tb.Status() != aidialogue.StatusReady {
		t.Errorf("Status = %v, want Ready (fast is ready)", tb.Status())
	}
}

// ────────────────────────────────────────────────────────────────
// Query classification
// ────────────────────────────────────────────────────────────────

func TestIsSimple_IsFallback_ReturnsTrue(t *testing.T) {
	qc := aidialogue.QueryClassifier{}
	if !qc.IsSimple("any text", aidialogue.ClassifyResult{IsFallback: true}) {
		t.Error("IsFallback should always classify as simple")
	}
}

func TestIsSimple_NoDomainsNoEntities_ReturnsTrue(t *testing.T) {
	qc := aidialogue.QueryClassifier{}
	if !qc.IsSimple("Hello there, friend.", aidialogue.ClassifyResult{
		DomainCount: 0, EntityRefCount: 0,
	}) {
		t.Error("no domains + no entities should classify as simple greeting")
	}
}

func TestIsSimple_ShortNoEntities_ReturnsTrue(t *testing.T) {
	qc := aidialogue.QueryClassifier{}
	if !qc.IsSimple("Yes.", aidialogue.ClassifyResult{
		DomainCount: 1, EntityRefCount: 0,
	}) {
		t.Error("short prompt (<5 tokens) with no entities should be simple")
	}
}

func TestIsSimple_DomainMatched_ReturnsFalse(t *testing.T) {
	qc := aidialogue.QueryClassifier{}
	if qc.IsSimple("What can you tell me about the old mill near Greyhaven?", aidialogue.ClassifyResult{
		DomainCount: 2, EntityRefCount: 3,
	}) {
		t.Error("domain-matched + entity refs should be complex")
	}
}

func TestIsSimple_DefaultConservative(t *testing.T) {
	// When uncertain (domains=1, entities=1), default to complex.
	qc := aidialogue.QueryClassifier{}
	if qc.IsSimple("Some question here ok", aidialogue.ClassifyResult{
		DomainCount: 1, EntityRefCount: 1,
	}) {
		t.Error("domains=1 + entities=1 should NOT be simple (conservative)")
	}
}
