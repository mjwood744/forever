package mysim

import (
	"fmt"
	"testing"

	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/proto"
)

func TestBuildBWeights(t *testing.T) {
	b := build{"B", "5-5500013301301551-0522103", true}
	stats := []proto.Stat{proto.Stat_StatStrength, proto.Stat_StatAgility, proto.Stat_StatStamina, proto.Stat_StatIntellect,
		proto.Stat_StatSpellDamage, proto.Stat_StatHolyDamage, proto.Stat_StatAttackPower, proto.Stat_StatHitRating, proto.Stat_StatCritRating,
		proto.Stat_StatHasteRating, proto.Stat_StatExpertiseRating, proto.Stat_StatDefenseRating, proto.Stat_StatBlockRating, proto.Stat_StatBlockValue,
		proto.Stat_StatDodgeRating, proto.Stat_StatParryRating, proto.Stat_StatArmor, proto.Stat_StatSpirit}
	for _, sc := range []struct{ n int; lab string }{{1, "1 mob"}, {4, "4 mobs"}} {
		req := &proto.StatWeightsRequest{
			Player: player(b), PartyBuffs: &proto.PartyBuffs{DevotionAura: true},
			RaidBuffs: &proto.RaidBuffs{ArcaneBrilliance: true, PrayerOfFortitude: true}, Debuffs: &proto.Debuffs{},
			Encounter: enc(sc.n, true), SimOptions: &proto.SimOptions{Iterations: 4000, RandomSeed: 7},
			Tanks: []*proto.UnitReference{{Type: proto.UnitReference_Player, Index: 0}},
			StatsToWeigh: stats, PseudoStatsToWeigh: []proto.PseudoStat{proto.PseudoStat_PseudoStatMainHandDps},
			EpReferenceStat: proto.Stat_StatStrength, HpsReferenceStat: proto.Stat_StatStrength,
		}
		r := core.StatWeights(req)
		if r.Error != nil { t.Fatal(r.Error.Message) }
		for _, s := range stats {
			fmt.Printf("SW|%s|%s|dps=%.3f|tps=%.3f|dtps=%.3f\n", sc.lab, s, r.Dps.Weights.Stats[s], r.Tps.Weights.Stats[s], r.Dtps.Weights.Stats[s])
		}
		fmt.Printf("SW|%s|WeaponDPS|dps=%.3f|tps=%.3f|dtps=%.3f\n", sc.lab, r.Dps.Weights.PseudoStats[0], r.Tps.Weights.PseudoStats[0], r.Dtps.Weights.PseudoStats[0])
	}
}
