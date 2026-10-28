package mysim

import (
	"fmt"
	"testing"

	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/proto"
)

func TestAshynWeights(t *testing.T) {
	sts := []struct{ n string; s proto.Stat; a float64 }{{"base", -1, 0}, {"Intellect", proto.Stat_StatIntellect, 40}, {"Spirit", proto.Stat_StatSpirit, 40}, {"MP5", proto.Stat_StatMP5, 10},
		{"Spell damage", proto.Stat_StatSpellDamage, 40}, {"Arcane damage", proto.Stat_StatArcaneDamage, 40}, {"Frost damage", proto.Stat_StatFrostDamage, 40}, {"Crit rating", proto.Stat_StatCritRating, 40},
		{"Hit rating", proto.Stat_StatHitRating, 40}, {"Haste rating", proto.Stat_StatHasteRating, 40}, {"Stamina", proto.Stat_StatStamina, 40}}
	for _, sc := range []string{"trio boss 32k", "4 mobs AE 45s"} {
		var b0 []*proto.UnitMetrics
		for _, x := range sts {
			m := ashyn()
			var e *proto.Encounter
			if sc == "4 mobs AE 45s" {
				m.Rotation = core.APLRotationFromJsonString(addShoot(aoeAE)); m.DistanceFromTarget = 8; e = encD(4, 45, 60, 350)
			} else {
				e = scen(sc)
			}
			if x.s >= 0 { bs := make([]float64, 40); bs[x.s] = x.a; m.BonusStats = &proto.UnitStats{Stats: bs} }
			pr := priestPlayer("505025021330011531-00505103", "1316991")
			pr.Rotation = core.APLRotationFromJsonString(nyxxAPL("50%"))
			res := core.RunRaidSim(&proto.RaidSimRequest{Raid: trioRaid(pr, m), Encounter: e, SimOptions: &proto.SimOptions{Iterations: 6000, RandomSeed: 33}})
			if res.Error != nil { t.Fatal(res.Error.Message) }
			ps := res.RaidMetrics.Parties[0].Players
			if x.s < 0 { b0 = ps }
			a := x.a; if a == 0 { a = 1 }
			fmt.Printf("AW|%s|%s|dps=%.1f|tps=%.1f|oom=%.2f|dDPS/pt=%.3f|dTPS/pt=%.3f|dOOM/pt=%.4f\n", sc, x.n, ps[2].Dps.Avg, ps[2].Threat.Avg, ps[2].SecondsOomAvg, (ps[2].Dps.Avg-b0[2].Dps.Avg)/a, (ps[2].Threat.Avg-b0[2].Threat.Avg)/a, (ps[2].SecondsOomAvg-b0[2].SecondsOomAvg)/a)
		}
	}
}
