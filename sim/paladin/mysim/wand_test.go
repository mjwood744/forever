package mysim

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/proto"
)


func addShoot(apl string) string {
	var m map[string]any
	if err := json.Unmarshal([]byte(apl), &m); err != nil { panic(err) }
	var sh any
	_ = json.Unmarshal([]byte(`{"action":{"castSpell":{"spellId":{"spellId":5019},"target":{"type":"Target"}}}}`), &sh)
	m["priorityList"] = append(m["priorityList"].([]any), sh)
	out, _ := json.Marshal(m)
	return string(out)
}

func trioRaid(pr, m *proto.Player) *proto.Raid {
	tank := player(build{"B", "5-5500013301301551-0522103", true})
	party := &proto.Party{Players: []*proto.Player{pr, tank, m}, Buffs: &proto.PartyBuffs{DevotionAura: true}}
	return &proto.Raid{Parties: []*proto.Party{party}, Buffs: &proto.RaidBuffs{ArcaneBrilliance: true, PrayerOfFortitude: true}, Debuffs: &proto.Debuffs{},
		Tanks: []*proto.UnitReference{{Type: proto.UnitReference_Player, Index: 1}}}
}

func TestWands(t *testing.T) {
	type nb struct{ name, tal string; wand bool }
	nyxx := []nb{{"D2 no wand", "505025021330011531-00505103", false}, {"D2 + wanding", "505025021330011531-00505103", true}, {"D2 + wanding + Wand Spec 2 (for Holy Precision)", "525005021330011531-00505103", true}}
	type mb struct{ name, tal string; wand bool }
	mages := []mb{{"AE spam no wand", "0552205003100311531-03-0533", false}, {"AE + wand when dry", "0552205003100311531-03-0533", true}, {"AE + wand + Wand Spec 2", "2552205003100311531-01-0533", true}}
	encs := map[string]func() *proto.Encounter{
		"1 mob 60s": func() *proto.Encounter { return encD(1, 60, 60, 350) },
		"4 mobs 45s": func() *proto.Encounter { return encD(4, 45, 60, 350) },
		"boss 32k": func() *proto.Encounter {
			e := encD(1, 300, 62, 900); e.UseHealth = true
			e.Targets[0].Stats = make([]float64, 40); e.Targets[0].Stats[proto.Stat_StatHealth] = 32000; return e
		},
	}
	for _, sc := range []string{"1 mob 60s", "4 mobs 45s", "boss 32k"} {
		for _, n := range nyxx {
			pr := priestPlayer(n.tal, "1316991")
			if n.wand { pr.Rotation = core.APLRotationFromJsonString(addShoot(strings.ReplaceAll(strings.ReplaceAll(healAPL, "PENHEAL", "1316991"), "MANAGATE", "50%"))) }
			m := magePlayer(); m.TalentsString = "0552205003100311531-03-0533"; m.Rotation = core.APLRotationFromJsonString(aoeAE); m.DistanceFromTarget = 8
			res := core.RunRaidSim(&proto.RaidSimRequest{Raid: trioRaid(pr, m), Encounter: encs[sc](), SimOptions: &proto.SimOptions{Iterations: 2000, RandomSeed: 9}})
			if res.Error != nil { t.Fatal(res.Error.Message) }
			ps := res.RaidMetrics.Parties[0].Players
			fmt.Printf("WN|%s|Nyxx %s|dps=%.1f|hps=%.1f|oom=%.1f|tankDeath=%.3f\n", sc, n.name, ps[0].Dps.Avg, ps[0].Hps.Avg, ps[0].SecondsOomAvg, ps[1].ChanceOfDeath)
		}
		if sc == "1 mob 60s" { continue }
		for _, mm := range mages {
			pr := priestPlayer("505025021330011531-00505103", "1316991")
			m := magePlayer(); m.TalentsString = mm.tal; m.DistanceFromTarget = 8
			apl := aoeAE
			if mm.wand { apl = addShoot(apl) }
			m.Rotation = core.APLRotationFromJsonString(apl)
			m.Buffs = &proto.IndividualBuffs{}
			res := core.RunRaidSim(&proto.RaidSimRequest{Raid: trioRaid(pr, m), Encounter: encs[sc](), SimOptions: &proto.SimOptions{Iterations: 2000, RandomSeed: 9}})
			if res.Error != nil { t.Fatal(res.Error.Message) }
			ps := res.RaidMetrics.Parties[0].Players
			fmt.Printf("WM|%s|Ashyn %s|dps=%.1f|oom=%.1f\n", sc, mm.name, ps[2].Dps.Avg, ps[2].SecondsOomAvg)
		}
	}
}
