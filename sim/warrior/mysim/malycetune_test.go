package mysim

import (
	"testing"

	"github.com/wowsims/forever/sim/core"
)

func TestMalyceTune(t *testing.T) {
	sk := "35300003-05-050533120300001351"
	vs := []struct{ n, tal string }{
		{"Demo FB base (Suppression 1)", "011-0305003201220301351-0050055003"},
		{"Suppression 5 (-Imp Corruption, -Agonizing Flames 3)", "05-0305003201220301351-0050055"},
		{"Suppression 3 (-Imp Corruption, -Agonizing Flames 1)", "03-0305003201220301351-0050055002"},
	}
	for _, sc := range []string{"1 mob 60s", "4 mobs 45s", "4 mobs RoF", "trio boss 32k"} {
		for _, v := range vs {
			c := t2{v.n, sk, WY, "demo_fire_brand", sc == "4 mobs RoF"}
			s := sc
			if sc == "4 mobs RoF" { s = "4 mobs 45s" }
			// override talents
			ps := runT2Tal(c, s, v.tal)
			pr2("MT", sc, c, ps)
		}
	}
	_ = core.RunRaidSim
}
