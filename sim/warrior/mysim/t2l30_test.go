package mysim

import (
	"strings"
	"testing"
)

func TestT2L30(t *testing.T) {
	S1, W1, L1 := "--050532020300001", "--5530003014", "-03050032012203"
	cs := []struct{ n, s, w, l string }{
		{"Snapshot (all three from level orders)", S1, W1, L1},
		{"Skarr: Improved Bloodrage 2 instead of Master of Defense 2", "--250532000300001", W1, L1},
		{"Wyther: Mana Tide + Tidal Mastery 5 instead of Ancestral Healing 3", S1, "--553000011501", L1},
		{"Wyther: Ele 11 / Resto 10 (damage-first)", S1, "5500001--55", L1},
		{"Malyce: Suppression 3 early (no Decimation, Demonic Brand 2)", S1, W1, "03-03050032012002"},
		{"Malyce: Destruction 21 (Bane, Aftermath, Ruin, Imp SB 3)", S1, W1, "--0350055003"},
	}
	for _, sc := range []string{"4 mobs 45s", "trio boss 32k", "stress boss 90s"} {
		for _, c := range cs {
			if sc == "stress boss 90s" && strings.HasPrefix(c.n, "Malyce") { continue }
			x := t2{c.n, c.s, c.w, "demo_fire_brand", false}
			pr2("L30", sc, x, runT2Tal(x, sc, c.l))
		}
	}
}
