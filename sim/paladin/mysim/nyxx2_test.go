package mysim

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/proto"
)

func nyxxAPL(gate string) string {
	return addShoot(strings.ReplaceAll(strings.ReplaceAll(healAPL, "PENHEAL", "1316991"), "MANAGATE", gate))
}

func ashyn() *proto.Player {
	m := magePlayer()
	m.TalentsString = "0552205003100311531-03-0533"
	data, _ := os.ReadFile("../../../ui/specs/mage/dps/apls/arcaneFrostFill.apl.json")
	m.Rotation = core.APLRotationFromJsonString(addShoot(string(data)))
	m.Buffs = &proto.IndividualBuffs{GreaterBlessingOfSalvation: true}
	return m
}

func scen(sc string) *proto.Encounter {
	switch sc {
	case "1 mob 60s":
		return encD(1, 60, 60, 350)
	case "4 mobs 45s":
		return encD(4, 45, 60, 350)
	case "stress boss 90s (Nyxx+Rellyk only)":
		return encD(1, 90, 62, 900)
	}
	e := encD(1, 300, 62, 900); e.UseHealth = true
	e.Targets[0].Stats = make([]float64, 40); e.Targets[0].Stats[proto.Stat_StatHealth] = 32000
	return e
}

func runNyxx(tal, gate, sc string, bonus proto.Stat, amt float64) []*proto.UnitMetrics {
	pr := priestPlayer(tal, "1316991")
	pr.Rotation = core.APLRotationFromJsonString(nyxxAPL(gate))
	if bonus >= 0 { bs := make([]float64, 40); bs[bonus] = amt; pr.BonusStats = &proto.UnitStats{Stats: bs} }
	var r *proto.Raid
	if strings.HasPrefix(sc, "stress") {
		tank := player(build{"B", "5-5500013301301551-0522103", true})
		r = &proto.Raid{Parties: []*proto.Party{{Players: []*proto.Player{pr, tank}, Buffs: &proto.PartyBuffs{DevotionAura: true}}}, Buffs: &proto.RaidBuffs{ArcaneBrilliance: true, PrayerOfFortitude: true}, Debuffs: &proto.Debuffs{},
			Tanks: []*proto.UnitReference{{Type: proto.UnitReference_Player, Index: 1}}}
	} else {
		r = trioRaid(pr, ashyn())
	}
	res := core.RunRaidSim(&proto.RaidSimRequest{Raid: r, Encounter: scen(sc), SimOptions: &proto.SimOptions{Iterations: 3000, RandomSeed: 21}})
	if res.Error != nil { panic(res.Error.Message) }
	return res.RaidMetrics.Parties[0].Players
}

var scs = []string{"1 mob 60s", "4 mobs 45s", "trio boss 32k", "stress boss 90s (Nyxx+Rellyk only)"}

func TestNyxxWand(t *testing.T) {
	builds := []struct{ name, tal string }{
		{"A Smite", "525235031030001001-30515100002-3"},
		{"B Healing", "005025021330011531-335001030301"},
		{"D1 B+PiL", "505025021330011531-235001030"},
		{"D2 B+PiL+DF", "505025021330011531-00505103"},
	}
	for _, rot := range []struct{ n, gate string }{{"wand, damage at >50% mana", "50%"}, {"heal-then-wand, damage at >90% mana", "90%"}} {
		for _, sc := range scs {
			for _, b := range builds {
				ps := runNyxx(b.tal, rot.gate, sc, -1, 0)
				fmt.Printf("NB|%s|%s|%s|dps=%.1f|hps=%.1f|oom=%.1f|tankDeath=%.3f\n", rot.n, sc, b.name, ps[0].Dps.Avg, ps[0].Hps.Avg, ps[0].SecondsOomAvg, ps[1].ChanceOfDeath)
			}
		}
	}
}

func TestNyxxSpirit(t *testing.T) {
	sts := []struct{ n string; s proto.Stat; a float64 }{{"base", -1, 0}, {"Intellect", proto.Stat_StatIntellect, 40}, {"Spirit", proto.Stat_StatSpirit, 40}, {"MP5", proto.Stat_StatMP5, 10}, {"Spell damage", proto.Stat_StatSpellDamage, 40}}
	for _, gate := range []string{"50%", "90%"} {
		for _, sc := range []string{"4 mobs 45s", "stress boss 90s (Nyxx+Rellyk only)"} {
			var b0 []*proto.UnitMetrics
			for _, x := range sts {
				ps := runNyxx("505025021330011531-00505103", gate, sc, x.s, x.a)
				if x.s < 0 { b0 = ps }
				a := x.a; if a == 0 { a = 1 }
				fmt.Printf("SP|gate=%s|%s|%s|dps=%.1f|oom=%.2f|dDPS/pt=%.3f|dOOM/pt=%.4f|death=%.3f\n", gate, sc, x.n, ps[0].Dps.Avg, ps[0].SecondsOomAvg, (ps[0].Dps.Avg-b0[0].Dps.Avg)/a, (ps[0].SecondsOomAvg-b0[0].SecondsOomAvg)/a, ps[1].ChanceOfDeath)
			}
		}
	}
}
