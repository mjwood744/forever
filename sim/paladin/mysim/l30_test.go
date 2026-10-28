package mysim

import (
	"fmt"
	"strings"
	"testing"

	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/proto"
)

func run30(rel, nyx, ash, sc string) []*proto.UnitMetrics {
	tank := player(build{"r", rel, true})
	pr := priestPlayer(nyx, "1316991")
	pr.Rotation = core.APLRotationFromJsonString(nyxxAPL("50%"))
	m := ashyn()
	m.TalentsString = ash
	var r *proto.Raid
	if strings.HasPrefix(sc, "stress") {
		r = &proto.Raid{Parties: []*proto.Party{{Players: []*proto.Player{pr, tank}, Buffs: &proto.PartyBuffs{DevotionAura: true}}}, Buffs: &proto.RaidBuffs{ArcaneBrilliance: true, PrayerOfFortitude: true}, Debuffs: &proto.Debuffs{},
			Tanks: []*proto.UnitReference{{Type: proto.UnitReference_Player, Index: 1}}}
	} else {
		party := &proto.Party{Players: []*proto.Player{pr, tank, m}, Buffs: &proto.PartyBuffs{DevotionAura: true}}
		r = &proto.Raid{Parties: []*proto.Party{party}, Buffs: &proto.RaidBuffs{ArcaneBrilliance: true, PrayerOfFortitude: true}, Debuffs: &proto.Debuffs{},
			Tanks: []*proto.UnitReference{{Type: proto.UnitReference_Player, Index: 1}}}
	}
	res := core.RunRaidSim(&proto.RaidSimRequest{Raid: r, Encounter: scen(sc), SimOptions: &proto.SimOptions{Iterations: 3000, RandomSeed: 41}})
	if res.Error != nil { panic(res.Error.Message) }
	return res.RaidMetrics.Parties[0].Players
}

func TestL30(t *testing.T) {
	R1, R2 := "-5500012301301", "-25300133013"
	N1, N2, N3 := "505005020030001", "025005020330001", "525005000030001"
	A1, A2 := "0302205003100311", "2102205003100311"
	combos := []struct{ n, r, nx, a string }{
		{"Snapshot (all three from final level orders)", R1, N1, A1},
		{"Rellyk R2 (Precision+ImpRF3, no Bulwark)", R2, N1, A1},
		{"Nyxx N2 (Wand Spec+ImpPWS, no PiL)", R1, N2, A1},
		{"Nyxx N3 (Wand Spec instead of Mental Agility)", R1, N3, A1},
		{"Ashyn A2 (Wand Spec instead of 2 Arcane Focus)", R1, N1, A2},
	}
	for _, sc := range []string{"4 mobs 45s", "trio boss 32k", "stress boss 90s (Nyxx+Rellyk only)"} {
		for _, c := range combos {
			if strings.HasPrefix(sc, "stress") && strings.HasPrefix(c.n, "Ashyn") { continue }
			ps := run30(c.r, c.nx, c.a, sc)
			md, mt := 0.0, 0.0
			if len(ps) > 2 { md, mt = ps[2].Dps.Avg, ps[2].Threat.Avg }
			fmt.Printf("L30|%s|%s|groupDPS=%.1f|rellykDPS=%.1f|rellykTPS=%.1f|nyxxDPS=%.1f|nyxxOOM=%.1f|ashynDPS=%.1f|ashynTPSratio=%.2f|tankDTPS=%.1f|death=%.3f\n",
				sc, c.n, ps[0].Dps.Avg+ps[1].Dps.Avg+md, ps[1].Dps.Avg, ps[1].Threat.Avg, ps[0].Dps.Avg, ps[0].SecondsOomAvg, md, mt/ps[1].Threat.Avg, ps[1].Dtps.Avg, ps[1].ChanceOfDeath)
		}
	}
}
