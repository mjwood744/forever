package mysim

import (
	"fmt"
	"os"
	"testing"

	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/proto"
)

func TestAshynSalv(t *testing.T) {
	builds := []struct{ name, talents, apl string }{
		{"Arcane/Frost", "0552205003100311531-03-0533", "arcaneFrostFill"},
		{"Arcane/Fire", "0502205003100311531-230510301001-003", "arcaneFireFill"},
		{"Fire (deep)", "05022050001001-23550000130135031-003", "deepFire"},
		{"Frost (deep)", "05022050011003--053500030100331251", "frost"},
	}
	tank := player(build{"B", "5-5500013301301551-0522103", true})
	for _, sc := range []string{"boss", "aoeAE", "aoeBliz"} {
		for _, salv := range []bool{false, true} {
			for _, b := range builds {
				m := magePlayer()
				m.TalentsString = b.talents
				var e *proto.Encounter
				switch sc {
				case "boss":
					data, _ := os.ReadFile("../../../ui/specs/mage/dps/apls/" + b.apl + ".apl.json")
					m.Rotation = core.APLRotationFromJsonString(string(data))
					e = encD(1, 300, 62, 900); e.UseHealth = true
					e.Targets[0].Stats = make([]float64, 40); e.Targets[0].Stats[proto.Stat_StatHealth] = 32000
				case "aoeAE":
					m.Rotation = core.APLRotationFromJsonString(aoeAE); m.DistanceFromTarget = 8; e = encD(4, 45, 60, 350)
				default:
					m.Rotation = core.APLRotationFromJsonString(aoeBliz); e = encD(4, 45, 60, 350)
				}
				if salv { m.Buffs = &proto.IndividualBuffs{GreaterBlessingOfSalvation: true} }
				party := &proto.Party{Players: []*proto.Player{priestPlayer("505025021330011531-00505103", "1316991"), tank, m}, Buffs: &proto.PartyBuffs{DevotionAura: true}}
				r := &proto.Raid{Parties: []*proto.Party{party}, Buffs: &proto.RaidBuffs{ArcaneBrilliance: true, PrayerOfFortitude: true}, Debuffs: &proto.Debuffs{},
					Tanks: []*proto.UnitReference{{Type: proto.UnitReference_Player, Index: 1}}}
				res := core.RunRaidSim(&proto.RaidSimRequest{Raid: r, Encounter: e, SimOptions: &proto.SimOptions{Iterations: 2000, RandomSeed: 9}})
				if res.Error != nil { t.Fatalf("%s: %v", b.name, res.Error.Message) }
				ps := res.RaidMetrics.Parties[0].Players
				fmt.Printf("SV|%s|salv=%v|%s|mageDPS=%.1f|mageTPS=%.1f|tankTPS=%.1f|ratio=%.2f|oom=%.1f\n", sc, salv, b.name, ps[2].Dps.Avg, ps[2].Threat.Avg, ps[1].Threat.Avg, ps[2].Threat.Avg/ps[1].Threat.Avg, ps[2].SecondsOomAvg)
			}
		}
	}
}
