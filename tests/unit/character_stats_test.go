// Example unit test — Character Stats attribute modifier formula.
// Pattern for all unit tests in this project.
package unit

import (
	"math"
	"testing"
)

// clamp is a helper matching the expected ADR-0006 modifier clamp.
func clamp(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// modifier returns the BG1-compatible attribute modifier.
func modifier(attr int) int {
	return clamp(int(math.Floor(float64(attr-10)/2.0)), -4, 4)
}

func TestModifier_Attribute10_ReturnsZero(t *testing.T) {
	if m := modifier(10); m != 0 {
		t.Errorf("attribute 10 should give modifier 0, got %d", m)
	}
}

func TestModifier_Attribute14_ReturnsPlus2(t *testing.T) {
	if m := modifier(14); m != 2 {
		t.Errorf("attribute 14 should give modifier +2, got %d", m)
	}
}

func TestModifier_Attribute18_ReturnsPlus4(t *testing.T) {
	if m := modifier(18); m != 4 {
		t.Errorf("attribute 18 should give modifier +4, got %d", m)
	}
}

func TestModifier_Attribute3_ReturnsMinus4(t *testing.T) {
	if m := modifier(3); m != -4 {
		t.Errorf("attribute 3 should give modifier -4, got %d", m)
	}
}

func TestModifier_Attribute8_ReturnsMinus1(t *testing.T) {
	if m := modifier(8); m != -1 {
		t.Errorf("attribute 8 should give modifier -1, got %d", m)
	}
}

func TestModifier_Attribute20_ClampedTo4(t *testing.T) {
	// Values above 18 clamp to +4 max
	if m := modifier(20); m != 4 {
		t.Errorf("attribute 20 should clamp modifier to +4, got %d", m)
	}
}

func TestModifier_Attribute1_ClampedToMinus4(t *testing.T) {
	// Values below 3 clamp to -4 min
	if m := modifier(1); m != -4 {
		t.Errorf("attribute 1 should clamp modifier to -4, got %d", m)
	}
}
