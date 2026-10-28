package mysim

import (
	"fmt"
	"strings"
	"testing"

	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/proto"
)

func runRed(rel, nyx, ash, sc string) []*proto.UnitMetrics {
	tank := player(build{"r", rel, true})
	pr := priestPlayer(nyx, "1316991")
	pr.Rotation = core.APLRotationFromJsonString(nyxxAPL("50%"))
	m := ashyn()
	m.TalentsString = ash
	var r *proto.Raid
	e := scen(sc)
	if sc == "4 mobs AE" {
		m.Rotation = core.APLRotationFromJsonString(addShoot(aoeAE)); m.DistanceFromTarget = 8; e = encD(4, 45, 60, 350)
	}
	if strings.HasPrefix(sc, "stress") {
		r = &proto.Raid{Parties: []*proto.Party{{Players: []*proto.Player{pr, tank}, Buffs: &proto.PartyBuffs{DevotionAura: true}}}, Buffs: &proto.RaidBuffs{ArcaneBrilliance: true, PrayerOfFortitude: true}, Debuffs: &proto.Debuffs{},
			Tanks: []*proto.UnitReference{{Type: proto.UnitReference_Player, Index: 1}}}
	} else {
		party := &proto.Party{Players: []*proto.Player{pr, tank, m}, Buffs: &proto.PartyBuffs{DevotionAura: true}}
		r = &proto.Raid{Parties: []*proto.Party{party}, Buffs: &proto.RaidBuffs{ArcaneBrilliance: true, PrayerOfFortitude: true}, Debuffs: &proto.Debuffs{},
			Tanks: []*proto.UnitReference{{Type: proto.UnitReference_Player, Index: 1}}}
	}
	res := core.RunRaidSim(&proto.RaidSimRequest{Raid: r, Encounter: e, SimOptions: &proto.SimOptions{Iterations: 3000, RandomSeed: 55}})
	if res.Error != nil { panic(res.Error.Message) }
	return res.RaidMetrics.Parties[0].Players
}

func TestRed(t *testing.T) {
	R, N, A := "5-5500013301301551-0522103", "505025021330011531-00505103", "0552205003100311531-03-0533"
	combos := []struct{ n, r, nx, a string }{
		{"Locked builds", R, N, A},
		{"R: Divine Strength 5 -> Anticipation 5", "-5500513301301551-0522103", N, A},
		{"R: Divine Strength 5 -> Precision 3 + Sacred Duty 2", "-5530013321301551-0522103", N, A},
		{"R: Sanctified Judgement 3 -> Precision 3", "5-5530013301301551-05221", N, A},
		{"N: Holy Precision 2 -> Wand Spec 2", R, "525005021330011531-00505103", A},
		{"N: Holy Nova -> Imp Renew 1", R, "505025021330011531-01505003", A},
		{"A: Incineration 3 -> Wand Spec 2 + Ice Shards 1", R, N, "2552205003100311531--0534"},
		{"A: Imp Channeling 5 -> Wand Spec 2 + Arcane Geometry 1 + Ice Shards 2", R, N, "2502205013100311531-03-0535"},
	}
	for _, sc := range []string{"4 mobs 45s", "4 mobs AE", "trio boss 32k", "stress boss 90s (Nyxx+Rellyk only)"} {
		for _, c := range combos {
			if strings.HasPrefix(sc, "stress") || !(strings.HasPrefix(c.n, "R:") || strings.HasPrefix(c.n, "Locked")) { continue }
			ps := runRed(c.r, c.nx, c.a, sc)
			md, mt := 0.0, 0.0
			if len(ps) > 2 { md, mt = ps[2].Dps.Avg, ps[2].Threat.Avg }
			fmt.Printf("RED|%s|%s|groupDPS=%.1f|rTPS=%.1f|rDTPS=%.1f|death=%.3f|nDPS=%.1f|nOOM=%.1f|aDPS=%.1f|aRatio=%.2f|aOOM=%.1f\n",
				sc, c.n, ps[0].Dps.Avg+ps[1].Dps.Avg+md, ps[1].Threat.Avg, ps[1].Dtps.Avg, ps[1].ChanceOfDeath, ps[0].Dps.Avg, ps[0].SecondsOomAvg, md, mt/ps[1].Threat.Avg, func() float64 { if len(ps) > 2 { return ps[2].SecondsOomAvg }; return 0 }())
		}
	}
}
