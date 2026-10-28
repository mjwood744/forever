package mysim

import (
	"fmt"
	"testing"

	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/proto"
)

func TestSkarrHealed(t *testing.T) {
	builds := []wb{
		{"A Arms (2H)", "30325213132515201-0505-2", false},
		{"B Prot", "35300003-05-050531120300021351", true},
		{"C Hybrid", "04305113102--050531120300011351", true},
	}
	for _, sc := range []struct{ n int; lab string; dur float64; lvl int32; dmg float64 }{
		{4, "4 mobs 45s", 45, 60, 350}, {1, "dungeon boss 90s", 90, 62, 900}} {
		for _, hps := range []float64{100, 150, 200} {
			for _, b := range builds {
				p := warrior(b)
				p.HealingModel = &proto.HealingModel{Hps: hps, CadenceSeconds: 2.5, CadenceVariation: 0.5}
				r := core.SinglePlayerRaidProto(p, &proto.PartyBuffs{BloodPact: true}, &proto.RaidBuffs{StrengthOfEarthTotem: true}, &proto.Debuffs{})
				r.Tanks = append(r.Tanks, &proto.UnitReference{Type: proto.UnitReference_Player, Index: 0})
				e := enc(sc.n, true, sc.dur)
				for _, tg := range e.Targets { tg.Level = sc.lvl; tg.MinBaseDamage = sc.dmg }
				res := core.RunRaidSim(&proto.RaidSimRequest{Raid: r, Encounter: e, SimOptions: &proto.SimOptions{Iterations: 3000, RandomSeed: 7}})
				if res.Error != nil { t.Fatalf("%s: %v", b.name, res.Error.Message) }
				pm := res.RaidMetrics.Parties[0].Players[0]
				fmt.Printf("SH|%s|hps=%.0f|%s|dps=%.1f|tps=%.1f|dtps=%.1f|death=%.3f\n", sc.lab, hps, b.name, pm.Dps.Avg, pm.Threat.Avg, pm.Dtps.Avg, pm.ChanceOfDeath)
			}
		}
	}
}
