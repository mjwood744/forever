package mysim

import (
	"fmt"
	"testing"

	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/proto"
)

func TestNyxxWeights(t *testing.T) {
	tal := "505025021330011531-00505103"
	tank := player(build{"B", "5-5500013301301551-0522103", true})
	type st struct{ name string; s proto.Stat; amt float64 }
	sts := []st{{"base", -1, 0}, {"Intellect", proto.Stat_StatIntellect, 40}, {"Spirit", proto.Stat_StatSpirit, 40}, {"MP5", proto.Stat_StatMP5, 10},
		{"Healing power", proto.Stat_StatHealingPower, 40}, {"Spell damage (all)", proto.Stat_StatSpellDamage, 40}, {"Crit rating", proto.Stat_StatCritRating, 40},
		{"Hit rating", proto.Stat_StatHitRating, 40}, {"Haste rating", proto.Stat_StatHasteRating, 40}, {"Stamina", proto.Stat_StatStamina, 40}}
	for _, sc := range []string{"boss", "4mobs"} {
		var base [3]float64
		for _, x := range sts {
			p := priestPlayer(tal, "1316991")
			if x.s >= 0 {
				bs := make([]float64, 40)
				bs[x.s] = x.amt
				p.BonusStats = &proto.UnitStats{Stats: bs}
			}
			party := &proto.Party{Players: []*proto.Player{p, tank, magePlayer()}, Buffs: &proto.PartyBuffs{DevotionAura: true}}
			r := &proto.Raid{Parties: []*proto.Party{party}, Buffs: &proto.RaidBuffs{ArcaneBrilliance: true, PrayerOfFortitude: true}, Debuffs: &proto.Debuffs{},
				Tanks: []*proto.UnitReference{{Type: proto.UnitReference_Player, Index: 1}}}
			var e *proto.Encounter
			if sc == "boss" {
				e = encD(1, 300, 62, 900); e.UseHealth = true
				e.Targets[0].Stats = make([]float64, 40); e.Targets[0].Stats[proto.Stat_StatHealth] = 32000
			} else {
				e = encD(4, 45, 60, 350)
			}
			res := core.RunRaidSim(&proto.RaidSimRequest{Raid: r, Encounter: e, SimOptions: &proto.SimOptions{Iterations: 6000, RandomSeed: 5}})
			if res.Error != nil { t.Fatal(res.Error.Message) }
			ps := res.RaidMetrics.Parties[0].Players
			v := [3]float64{ps[0].Dps.Avg, ps[0].Hps.Avg, ps[0].SecondsOomAvg}
			if x.s < 0 { base = v }
			fmt.Printf("NW|%s|%s|dps=%.1f|hps=%.1f|oom=%.2f|dDPS/pt=%.3f|dHPS/pt=%.3f|dOOM/pt=%.4f|tdtps=%.1f|death=%.3f\n", sc, x.name, v[0], v[1], v[2],
				(v[0]-base[0])/max(1, x.amt), (v[1]-base[1])/max(1, x.amt), (v[2]-base[2])/max(1, x.amt), ps[1].Dtps.Avg, ps[1].ChanceOfDeath)
		}
	}
}
