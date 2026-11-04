package opener

import "testing"

// A player's cast line is read with the player's name, its party number left off.
func TestCastLineReadsThePlayer(t *testing.T) {
	m := castLine.FindStringSubmatch("[-1.50] [Skarr (#1)] Casting {SpellID: 2687} (Cost = 0.000, Cast Time = 0s)")
	if m == nil || m[1] != "-1.50" || m[2] != "Skarr" || m[3] != "2687" {
		t.Fatalf("read %q, want -1.50, Skarr, 2687", m)
	}
	if m := castLine.FindStringSubmatch("[2.00] [Skarr (#1) - Voidwalker] Casting {SpellID: 1}"); m != nil && m[2] == "Skarr" {
		t.Errorf("a pet's cast read as its owner's: %q", m)
	}
}
