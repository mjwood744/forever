package mysim

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/proto"
	"github.com/wowsims/forever/sim/druid/restoration"
)

func init() { restoration.RegisterRestorationDruid() }

// druidAPL heals the unit at tankJSON; damage above the mana gate.
func druidAPL(tankJSON, gate string) string {
	hp := `{"currentHealthPercent":{"sourceUnit":` + tankJSON + `}}`
	le := func(v string) string { return `{"cmp":{"op":"OpLe","lhs":` + hp + `,"rhs":{"const":{"val":"` + v + `"}}}}` }
	notHot := func(id string) string { return `{"not":{"val":{"dotIsActive":{"spellId":{"spellId":` + id + `},"targetUnit":` + tankJSON + `}}}}` }
	and := func(a, b string) string { return `{"and":{"vals":[` + a + `,` + b + `]}}` }
	heal := func(cond, id string) string {
		return `{"action":{"condition":` + cond + `,"castSpell":{"spellId":{"spellId":` + id + `},"target":` + tankJSON + `}}}`
	}
	mana := `{"cmp":{"op":"OpGe","lhs":{"currentManaPercent":{}},"rhs":{"const":{"val":"` + gate + `"}}}}`
	pl := []string{
		heal(le("45%"), "18562"),
		heal(le("45%"), "25297"),
		heal(and(le("90%"), notHot("25299")), "25299"),
		heal(le("80%"), "1238215"),
		heal(and(le("70%"), notHot("9858")), "9858"),
		heal(le("60%"), "25297"),
		`{"action":{"condition":{"and":{"vals":[` + mana + `,{"not":{"val":{"dotIsActive":{"spellId":{"spellId":9835}}}}}]}},"castSpell":{"spellId":{"spellId":9835}}}}`,
		`{"action":{"condition":` + mana + `,"castSpell":{"spellId":{"spellId":9912}}}}`,
	}
	return `{"type":"TypeAPL","priorityList":[` + strings.Join(pl, ",") + `]}`
}

func restoDruid(talents, tankJSON, gate string) *proto.Player {
	return core.WithSpec(&proto.Player{
		Class: proto.Class_ClassDruid, Race: proto.Race_RaceWindshaperSkyborne,
		Equipment:   core.GetGearSet("../../../ui/specs/druid/balance/gear_sets", "prebis").GearSet,
		Consumables: &proto.ConsumesSpec{}, Buffs: &proto.IndividualBuffs{},
		TalentsString: talents, DistanceFromTarget: 20, ReactionTimeMs: 100,
		Rotation: core.APLRotationFromJsonString(druidAPL(tankJSON, gate)),
	}, &proto.Player_RestorationDruid{RestorationDruid: &proto.RestorationDruid{Options: &proto.RestorationDruid_Options{ClassOptions: &proto.DruidOptions{}}}})
}

func TestDruidHealsSkarr(t *testing.T) {
	d := restoDruid("55322201--5050035103113031", tankRef, "50%")
	r := &proto.Raid{Parties: []*proto.Party{{Players: []*proto.Player{d, warrior(wb{"s", SK2, true})}}}, Buffs: &proto.RaidBuffs{}, Debuffs: &proto.Debuffs{},
		Tanks: []*proto.UnitReference{{Type: proto.UnitReference_Player, Index: 1}}}
	if os.Getenv("DBG") != "" {
		cs := core.ComputeStats(&proto.ComputeStatsRequest{Raid: r, Encounter: scen2("stress boss 90s")})
		for i, a := range cs.RaidStats.Parties[0].Players[0].RotationStats.PriorityList { for _, v := range a.Validations { fmt.Printf("VAL %d %s\n", i, v.Validation) } }
	}
	for _, sc := range []string{"4 mobs 45s", "stress boss 90s"} {
		res := core.RunRaidSim(&proto.RaidSimRequest{Raid: r, Encounter: scen2(sc), SimOptions: &proto.SimOptions{Iterations: 1000, RandomSeed: 3}})
		if res.Error != nil { t.Fatal(res.Error.Message) }
		p := res.RaidMetrics.Parties[0].Players[0]
		fmt.Printf("DH|%s|dps=%.1f hps=%.1f oom=%.1f tankDTPS=%.1f death=%.3f\n", sc, p.Dps.Avg, p.Hps.Avg, p.SecondsOomAvg, res.RaidMetrics.Parties[0].Players[1].Dtps.Avg, res.RaidMetrics.Parties[0].Players[1].ChanceOfDeath)
		for _, a := range p.Actions {
			var c int32; var h float64
			for _, tg := range a.Targets { c += tg.Casts; h += tg.Healing }
			if c > 0 { fmt.Printf("   ACT %v casts=%.1f heal/cast=%.0f\n", a.Id, float64(c)/1000, h/float64(c)) }
		}
	}
}
