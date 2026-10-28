package mysim

import (
	"fmt"
	"testing"

	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/proto"
)

const aoeBliz = `{"type":"TypeAPL","priorityList":[
 {"action":{"castSpell":{"spellId":{"spellId":13021}}}},
 {"action":{"condition":{"not":{"val":{"dotIsActive":{"spellId":{"spellId":10216}}}}},"castSpell":{"spellId":{"spellId":10216}}}},
 {"action":{"channelSpell":{"spellId":{"spellId":10187}}}}]}`
const aoeAE = `{"type":"TypeAPL","priorityList":[
 {"action":{"castSpell":{"spellId":{"spellId":13021}}}},
 {"action":{"castSpell":{"spellId":{"spellId":10161}}}},
 {"action":{"castSpell":{"spellId":{"spellId":10202}}}}]}`

func TestAshynAoE(t *testing.T) {
	builds := []struct{ name, talents string }{
		{"Arcane/Frost", "0552205003100311531-03-0533"},
		{"Arcane/Fire", "0502205003100311531-230510301001-003"},
		{"Fire (deep)", "05022050001001-23550000130135031-003"},
		{"Fire/Frost FFB", "-2355000013011503-2035010311013002"},
		{"Frost (deep)", "05022050011003--053500030100331251"},
	}
	tank := player(build{"B", "5-5500013301301551-0522103", true})
	for _, rot := range []struct{ n, apl string; dist float64 }{{"Flamestrike+Blizzard", aoeBliz, 25}, {"Arcane Explosion", aoeAE, 8}} {
		for _, b := range builds {
			m := magePlayer()
			m.TalentsString = b.talents
			m.Rotation = core.APLRotationFromJsonString(rot.apl)
			m.DistanceFromTarget = rot.dist
			party := &proto.Party{Players: []*proto.Player{priestPlayer("505025021330011531-00505103", "1316991"), tank, m}, Buffs: &proto.PartyBuffs{DevotionAura: true}}
			r := &proto.Raid{Parties: []*proto.Party{party}, Buffs: &proto.RaidBuffs{ArcaneBrilliance: true, PrayerOfFortitude: true}, Debuffs: &proto.Debuffs{},
				Tanks: []*proto.UnitReference{{Type: proto.UnitReference_Player, Index: 1}}}
			res := core.RunRaidSim(&proto.RaidSimRequest{Raid: r, Encounter: encD(4, 45, 60, 350), SimOptions: &proto.SimOptions{Iterations: 2000, RandomSeed: 9}})
			if res.Error != nil { t.Fatalf("%s: %v", b.name, res.Error.Message) }
			ps := res.RaidMetrics.Parties[0].Players
			fmt.Printf("AOE|%s|%s|mageDPS=%.1f|mageTPS=%.1f|tankTPS=%.1f|oom=%.1f\n", rot.n, b.name, ps[2].Dps.Avg, ps[2].Threat.Avg, ps[1].Threat.Avg, ps[2].SecondsOomAvg)
		}
	}
}
