package mysim

import (
	"fmt"
	"testing"

	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/proto"
)

const SK2, LK2 = "35300003-05-050533120300001351", "03-0305003201220301351-0050055002"

type bonus struct {
	n      string
	s      proto.Stat
	a      float64
	pseudo bool
}

func runW(who int, b bonus, sc string) []*proto.UnitMetrics {
	sk := warrior(wb{"s", SK2, true})
	w := wyther(WY, "50%", true)
	m := malyceBuild("demo_fire_brand")
	m.TalentsString = LK2
	ps := []*proto.Player{w, sk, m}
	if b.a != 0 {
		us := &proto.UnitStats{Stats: make([]float64, 40)}
		if b.pseudo {
			us.PseudoStats = make([]float64, 64)
			us.PseudoStats[0] = b.a
		} else {
			us.Stats[b.s] = b.a
		}
		ps[who].BonusStats = us
	}
	r := &proto.Raid{Parties: []*proto.Party{{Players: ps, Buffs: &proto.PartyBuffs{BloodPact: true}}}, Buffs: &proto.RaidBuffs{}, Debuffs: &proto.Debuffs{},
		Tanks: []*proto.UnitReference{{Type: proto.UnitReference_Player, Index: 1}}}
	res := core.RunRaidSim(&proto.RaidSimRequest{Raid: r, Encounter: scen2(sc), SimOptions: &proto.SimOptions{Iterations: 4000, RandomSeed: 29}})
	if res.Error != nil { panic(res.Error.Message) }
	return res.RaidMetrics.Parties[0].Players
}

func TestT2Weights(t *testing.T) {
	S := proto.Stat_StatStrength
	sets := map[int][]bonus{
		1: {{"base", S, 0, false}, {"Strength", proto.Stat_StatStrength, 40, false}, {"Agility", proto.Stat_StatAgility, 40, false}, {"Stamina", proto.Stat_StatStamina, 40, false},
			{"Attack power", proto.Stat_StatAttackPower, 40, false}, {"Hit rating", proto.Stat_StatHitRating, 40, false}, {"Crit rating", proto.Stat_StatCritRating, 40, false},
			{"Haste rating", proto.Stat_StatHasteRating, 40, false}, {"Expertise rating", proto.Stat_StatExpertiseRating, 40, false}, {"Defense rating", proto.Stat_StatDefenseRating, 40, false},
			{"Block rating", proto.Stat_StatBlockRating, 40, false}, {"Block value", proto.Stat_StatBlockValue, 40, false}, {"Dodge rating", proto.Stat_StatDodgeRating, 40, false},
			{"Parry rating", proto.Stat_StatParryRating, 40, false}, {"Armor", proto.Stat_StatArmor, 400, false}, {"Weapon DPS", S, 5, true}},
		0: {{"base", S, 0, false}, {"Intellect", proto.Stat_StatIntellect, 40, false}, {"Spirit", proto.Stat_StatSpirit, 40, false}, {"MP5", proto.Stat_StatMP5, 10, false},
			{"Spell damage", proto.Stat_StatSpellDamage, 40, false}, {"Healing power", proto.Stat_StatHealingPower, 40, false}, {"Crit rating", proto.Stat_StatCritRating, 40, false},
			{"Hit rating", proto.Stat_StatHitRating, 40, false}, {"Haste rating", proto.Stat_StatHasteRating, 40, false}},
		2: {{"base", S, 0, false}, {"Intellect", proto.Stat_StatIntellect, 40, false}, {"Spirit", proto.Stat_StatSpirit, 40, false}, {"Stamina", proto.Stat_StatStamina, 40, false},
			{"Spell damage", proto.Stat_StatSpellDamage, 40, false}, {"Shadow damage", proto.Stat_StatShadowDamage, 40, false}, {"Fire damage", proto.Stat_StatFireDamage, 40, false},
			{"Crit rating", proto.Stat_StatCritRating, 40, false}, {"Hit rating", proto.Stat_StatHitRating, 40, false}, {"Haste rating", proto.Stat_StatHasteRating, 40, false}},
	}
	names := map[int]string{0: "Wyther", 1: "Skarr", 2: "Malyce"}
	for _, who := range []int{1, 0, 2} {
		for _, sc := range []string{"4 mobs 45s", "trio boss 32k"} {
			var b0 []*proto.UnitMetrics
			for _, b := range sets[who] {
				ps := runW(who, b, sc)
				if b.a == 0 { b0 = ps }
				a := b.a; if a == 0 { a = 1 }
				p, q := ps[who], b0[who]
				fmt.Printf("TW|%s|%s|%s|dDPS/pt=%.3f|dTPS/pt=%.3f|dDTPS/pt=%.3f|dOOM/pt=%.4f|dHPS/pt=%.3f|skDeath=%.3f|wyOOM=%.1f\n", names[who], sc, b.n,
					(p.Dps.Avg-q.Dps.Avg)/a, (p.Threat.Avg-q.Threat.Avg)/a, (p.Dtps.Avg-q.Dtps.Avg)/a, (p.SecondsOomAvg-q.SecondsOomAvg)/a, (p.Hps.Avg-q.Hps.Avg)/a, ps[1].ChanceOfDeath, ps[0].SecondsOomAvg)
			}
		}
	}
}
