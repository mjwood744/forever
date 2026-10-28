package mysim

import (
	"strings"
	"testing"
)

func TestT2Red(t *testing.T) {
	S, W, L := SK2, WY, LK2
	cs := []struct{ n, s, w, l string }{
		{"Locked", S, W, L},
		{"Skarr: drop Imp HS 3 + Deep Wounds 3 -> Imp Sunder 3, Vanguard, Imp Bloodrage 2", "05300-05-250533120331001351", W, L},
		{"Skarr: Cruelty 5 -> Imp Shield Wall 2, Imp Shield Bash 2, Imp Bloodrage 1", "35300003--150533120300221351", W, L},
		{"Skarr: Imp Revenge 3 -> Imp Sunder 3", "35300003-05-050503120330001351", W, L},
		{"Wyther: Call of Flame 3 -> Restorative Totems 3", S, "5503301--5530003015313131", L},
		{"Wyther: Ele 15 + Ancestral Knowledge 5", S, "5503101-005-5530003015013131", L},
		{"Malyce: Aftermath 5 -> Imp Shadow Bolt 5", S, W, "03-0305003201220301351-0550005002"},
		{"Malyce: Agonizing Flames 2 -> Cataclysm 2", S, W, "03-0305003201220301351-0050255"},
	}
	for _, sc := range []string{"4 mobs 45s", "4 mobs RoF", "trio boss 32k", "stress boss 90s"} {
		for _, c := range cs {
			if sc == "stress boss 90s" && strings.HasPrefix(c.n, "Malyce") { continue }
			if sc == "4 mobs RoF" && !(strings.HasPrefix(c.n, "Malyce") || c.n == "Locked") { continue }
			x := t2{c.n, c.s, c.w, "demo_fire_brand", sc == "4 mobs RoF"}
			s := sc
			if sc == "4 mobs RoF" { s = "4 mobs 45s" }
			pr2("T2R", sc, x, runT2Tal(x, s, c.l))
		}
	}
}
