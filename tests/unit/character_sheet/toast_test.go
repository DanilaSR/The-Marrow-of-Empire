// Unit tests for Story 006: Inventory Toast Notifications.
//
// Evidence path: production/epics/character-inventory-ui/
// story-006-inventory-toast-notifications.md
// AC-13 (toast appears with "Inventory full" in amber for 2s ±200ms,
// dismisses automatically, no toast spam on rapid attempts) and the QA
// edge cases (at-capacity pickup, would-exceed pickup, queueing). The
// tests cover the pure display logic — duration, message, queue, color
// tokens, fade, reduced motion, trigger predicate — only; DOM rendering
// and the markup layout are verified by screenshot evidence per the UI
// story testing standard. No engine code runs here.
package character_sheet_test

import (
	"testing"

	"marrow-of-empire/character_sheet"
)

// AC-13: GIVEN a full inventory (weight at max capacity), WHEN the player
// attempts to pick up a world item, THEN a toast appears: "Inventory
// full" for 2 seconds (±200ms) in amber text.
func TestToast_AC13_InventoryFullAmberTwoSeconds(t *testing.T) {
	var toast character_sheet.Toast
	toast.InventoryFullToast()
	if !toast.Visible() {
		t.Fatal("toast not visible after InventoryFullToast()")
	}
	if got := toast.Text(); got != "Inventory full" {
		t.Errorf("toast text = %q, want %q", got, "Inventory full")
	}
	if r := toast.Remaining(); r < 1.8 || r > 2.2 {
		t.Errorf("toast duration = %.2fs, want 2s (±200ms)", r)
	}
	// Amber text: the semantic color token maps to the art-bible amber
	// hex — the presenter applies it to the toast label each frame
	// (amber on dark umber, never on parchment — control manifest).
	if got := character_sheet.ToastTextColorHex(character_sheet.ToastColorAmber); got != "#D9A441" {
		t.Errorf("ToastTextColorHex(amber) = %q, want %q", got, "#D9A441")
	}
}

// AC-13: the toast auto-dismisses — no click required. It holds at full
// opacity for its duration, then fades out and is gone.
func TestToast_AC13_DismissesAfterDuration(t *testing.T) {
	var toast character_sheet.Toast
	toast.Show("Inventory full", 2.0)
	toast.Update(1.5)
	if !toast.Visible() {
		t.Fatal("toast gone before its duration elapsed")
	}
	if o := toast.Opacity(); o != 1 {
		t.Errorf("toast opacity during hold = %f, want 1 (no fade before the hold ends)", o)
	}
	toast.Update(0.5) // hold elapsed: the fade-out begins
	if !toast.Visible() {
		t.Fatal("toast gone at the instant the duration elapsed — the fade-out should follow")
	}
	if o := toast.Opacity(); o != 1 {
		t.Errorf("toast opacity at fade start = %f, want 1", o)
	}
	toast.Update(0.15) // mid-fade
	if o := toast.Opacity(); o >= 1 || o <= 0 {
		t.Errorf("toast opacity mid-fade = %f, want strictly between 0 and 1", o)
	}
	toast.Update(0.15) // fade-out completes
	if toast.Visible() {
		t.Error("toast still visible after duration + fade-out, want dismissed")
	}
	if got := toast.Text(); got != "" {
		t.Errorf("dismissed toast text = %q, want empty", got)
	}
}

// Multiple toasts queue: at most one shows at a time, in FIFO order — the
// next toast starts only after the current one has fully dismissed
// (Story 006 Implementation Notes).
func TestToast_Queue_ShowsOneAtATime(t *testing.T) {
	var toast character_sheet.Toast
	toast.Show("Inventory full", 1.0)
	toast.Show("Quest item added", 1.0)
	if got := toast.Text(); got != "Inventory full" {
		t.Fatalf("first toast text = %q, want %q", got, "Inventory full")
	}
	toast.Update(1.0) // first hold ends
	toast.Update(0.3) // first fade-out completes
	if !toast.Visible() {
		t.Fatal("no toast visible after the first dismissed, want the queued toast")
	}
	if got := toast.Text(); got != "Quest item added" {
		t.Errorf("second toast text = %q, want %q", got, "Quest item added")
	}
	toast.Update(1.0) // second hold ends
	toast.Update(0.3) // second fade-out completes
	if toast.Visible() {
		t.Error("toast still visible after the queued toast dismissed, want idle")
	}
}

// Reduced motion: instant appear/disappear — the fade-out is skipped, so
// the toast is gone the moment its duration elapses and the queue
// advances instantly (the queued toast starts at full opacity with its
// full hold time).
func TestToast_ReducedMotion_InstantDismiss(t *testing.T) {
	var toast character_sheet.Toast
	toast.SetReducedMotion(true)
	toast.Show("Inventory full", 2.0)
	toast.Show("Quest item added", 2.0)
	toast.Update(2.0) // first toast's duration elapsed: no fade phase
	if got := toast.Text(); got != "Quest item added" {
		t.Errorf("toast text = %q, want %q (instant dismiss, queue advances instantly)", got, "Quest item added")
	}
	if r := toast.Remaining(); r < 1.8 || r > 2.2 {
		t.Errorf("queued toast duration = %.2fs, want full 2s (queue advanced instantly)", r)
	}
	toast.Update(2.0) // second toast's duration elapsed
	if toast.Visible() {
		t.Error("toast still visible after duration with reduced motion, want instant dismiss")
	}
}

// Reduced motion enabled mid-fade: the fading toast disappears instantly
// (Story 009 can flip the setting while a toast is up).
func TestToast_ReducedMotion_MidFadeDismissesInstantly(t *testing.T) {
	var toast character_sheet.Toast
	toast.Show("Inventory full", 2.0)
	toast.Update(2.0) // hold ends, fade-out begins
	if !toast.Visible() {
		t.Fatal("toast should be in its fade-out")
	}
	toast.SetReducedMotion(true)
	if toast.Visible() {
		t.Error("toast still visible after enabling reduced motion mid-fade, want instant dismiss")
	}
}

// QA AC-13 edge: rapid pickup attempts during the toast display must not
// spam — re-showing the same message re-triggers the timer (no extra
// queue entry, no second toast after dismissal).
func TestToast_SameMessage_RetriggersWithoutSpam(t *testing.T) {
	var toast character_sheet.Toast
	toast.Show("Inventory full", 2.0)
	toast.Show("Inventory full", 2.0) // rapid repeat attempt
	toast.Update(1.0)
	if r := toast.Remaining(); r < 0.9 || r > 1.1 {
		t.Errorf("remaining after re-trigger + 1s = %.2fs, want ~1s (timer restarted)", r)
	}
	toast.Update(1.0) // re-triggered hold ends
	toast.Update(0.3) // fade-out completes
	if toast.Visible() {
		t.Error("toast still visible after dismissal, want no second toast (no spam)")
	}
}

// AC-13 trigger predicate: PickupBlocked refuses the pickup when the pack
// is at or over capacity, or the item's weight would push it over. QA
// edge cases: at exactly 100% capacity the toast still fires (the pack
// cannot exceed capacity), and 99% + an item that pushes over fires too;
// a pickup that lands exactly on 100% is allowed.
func TestPickupBlocked_AC13_FullInventoryRefusesPickup(t *testing.T) {
	tests := []struct {
		name                         string
		current, itemWeight, maxCap  int
		want                         bool
	}{
		{"at capacity (exactly 100%)", 100, 1, 100, true},
		{"over capacity", 110, 1, 100, true},
		{"would exceed capacity (99% + 5)", 99, 5, 100, true},
		{"fits exactly to 100%", 99, 1, 100, false},
		{"fits with room", 95, 5, 100, false},
		{"empty pack", 0, 5, 100, false},
	}
	for _, tt := range tests {
		if got := character_sheet.PickupBlocked(tt.current, tt.itemWeight, tt.maxCap); got != tt.want {
			t.Errorf("PickupBlocked(%d, %d, %d) = %v, want %v (%s)",
				tt.current, tt.itemWeight, tt.maxCap, got, tt.want, tt.name)
		}
	}
}
