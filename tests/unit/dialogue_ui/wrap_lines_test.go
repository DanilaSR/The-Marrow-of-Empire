// Unit tests for INT-2 dialogue log wrapping (dialogue_ui.WrapLines).
//
// Evidence path: prototypes/marrow-of-empire-vertical-slice (integration story INT-2)
package dialogue_ui_test

import (
	"reflect"
	"testing"

	"marrow-of-empire/dialogue_ui"
)

func TestWrapLines_ShortTextSingleLine(t *testing.T) {
	got := dialogue_ui.WrapLines("The ore is wrong.", 40)
	want := []string{"The ore is wrong."}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestWrapLines_BreaksAtWordBoundary(t *testing.T) {
	got := dialogue_ui.WrapLines("one two three four five", 13)
	// "one two three" fits exactly in 13 chars; "four five" starts line 2.
	want := []string{"one two three", "four five"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestWrapLines_LongWordHardSplit(t *testing.T) {
	got := dialogue_ui.WrapLines("supercalifragilistic", 8)
	want := []string{"supercal", "ifragili", "stic"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestWrapLines_Empty(t *testing.T) {
	if got := dialogue_ui.WrapLines("   ", 20); got != nil {
		t.Fatalf("whitespace-only text must wrap to nil, got %v", got)
	}
}

func TestWrapLines_ExactFit(t *testing.T) {
	got := dialogue_ui.WrapLines("abcd efgh", 9)
	want := []string{"abcd efgh"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("exact-fit line must stay intact: got %v, want %v", got, want)
	}
}
