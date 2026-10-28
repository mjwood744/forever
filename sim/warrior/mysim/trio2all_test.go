package mysim

import (
	"fmt"
	"testing"

	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/proto"
)

const lockAoE = `{"type":"TypeAPL","priorityList":[
 {"action":{"condition":{"cmp":{"op":"OpLe","lhs":{"currentManaPercent":{}},"rhs":{"const":{"val":"15%"}}}},"castSpell":{"spellId":{"spellId":11689}}}},
 {"action":{"channelSpell":{"spellId":{"spellId":11678}}}}]}`

func malyceBuild(file string) *proto.Player {
	p := loadPlayer("../../../ui/specs/warlock/dps/builds/" + file + ".build.json")
	p.Race = proto.Race_RaceTroll
	p.Equipment = core.GetGearSet("../../../ui/specs/warlock/dps/gear_sets", "preraid").GearSet
	p.Consumables = &proto.ConsumesSpec{}
	p.Buffs = &proto.IndividualBuffs{}
	return p
}

type t2 struct{ name, skarr, wy, lock string; aoe bool }

func runT2(c t2, sc string) []*proto.UnitMetrics {
	sk := warrior(wb{"s", c.skarr, true})
	w := wyther(c.wy, "50%", true)
	m := malyceBuild(c.lock)
	if c.aoe { m.Rotation = core.APLRotationFromJsonString(lockAoE) }
	players := []*proto.Player{w, sk}
	if sc != "stress boss 90s" { players = append(players, m) }
	r := &proto.Raid{Parties: []*proto.Party{{Players: players, Buffs: &proto.PartyBuffs{BloodPact: true}}}, Buffs: &proto.RaidBuffs{}, Debuffs: &proto.Debuffs{},
		Tanks: []*proto.UnitReference{{Type: proto.UnitReference_Player, Index: 1}}}
	res := core.RunRaidSim(&proto.RaidSimRequest{Raid: r, Encounter: scen2(sc), SimOptions: &proto.SimOptions{Iterations: 2000, RandomSeed: 17}})
	if res.Error != nil { panic(c.name + ": " + res.Error.Message) }
	return res.RaidMetrics.Parties[0].Players
}

func pr2(tag, sc string, c t2, ps []*proto.UnitMetrics) {
	ld, lt, lo := 0.0, 0.0, 0.0
	if len(ps) > 2 { ld, lt, lo = ps[2].Dps.Avg, ps[2].Threat.Avg, ps[2].SecondsOomAvg }
	fmt.Printf("%s|%s|%s|groupDPS=%.1f|skDPS=%.1f|skTPS=%.1f|skDTPS=%.1f|death=%.3f|wyDPS=%.1f|wyHPS=%.1f|wyOOM=%.1f|lockDPS=%.1f|lockRatio=%.2f|lockOOM=%.1f\n",
		tag, sc, c.name, ps[0].Dps.Avg+ps[1].Dps.Avg+ld, ps[1].Dps.Avg, ps[1].Threat.Avg, ps[1].Dtps.Avg, ps[1].ChanceOfDeath, ps[0].Dps.Avg, ps[0].Hps.Avg, ps[0].SecondsOomAvg, ld, lt/ps[1].Threat.Avg, lo)
}

const SK, WY = "35300003-05-050531120300021351", "5503331--5530003015013131"

func TestMalyce(t *testing.T) {
	for _, sc := range []string{"1 mob 60s", "4 mobs 45s", "trio boss 32k"} {
		for _, f := range []string{"aff_wrack", "demo_fire_brand", "demo_shadow_brand", "destro", "sm_ruin"} {
			c := t2{f, SK, WY, f, false}
			pr2("ML", sc, c, runT2(c, sc))
			if sc == "4 mobs 45s" {
				c.aoe = true; c.name = f + " + Rain of Fire"
				pr2("ML", sc, c, runT2(c, sc))
			}
		}
	}
}

func TestT2Variants(t *testing.T) {
	cs := []t2{
		{"Base (Skarr B, Wyther A)", SK, WY, "sm_ruin", false},
		{"Skarr: Imp Shield Wall 2 -> Imp Thunder Clap +2", "35300003-05-050533120300001351", WY, "sm_ruin", false},
		{"Skarr: Cruelty 3 -> Imp Sunder Armor 3", "35300003-02-050531120330021351", WY, "sm_ruin", false},
		{"Skarr: C hybrid (Arms 20)", "04305113102--050531120300011351", WY, "sm_ruin", false},
		{"Wyther: Ele 15 / Resto 36 (+Restorative Totems 5)", SK, "5501301--5530003015513131", "sm_ruin", false},
	}
	for _, sc := range []string{"4 mobs 45s", "trio boss 32k", "stress boss 90s"} {
		for _, c := range cs { pr2("TV", sc, c, runT2(c, sc)) }
	}
}
