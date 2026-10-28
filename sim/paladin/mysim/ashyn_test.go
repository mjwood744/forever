package mysim

import (
	"fmt"
	"os"
	"testing"

	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/proto"
)

func mageBuild(talents, apl string) *proto.Player {
	p := magePlayer()
	data, _ := os.ReadFile("../../../ui/specs/mage/dps/apls/" + apl + ".apl.json")
	p.Rotation = core.APLRotationFromJsonString(string(data))
	p.TalentsString = talents
	return p
}

func TestAshyn(t *testing.T) {
	builds := []struct{ name, talents, apl string }{
		{"Arcane/Frost", "0552205003100311531-03-0533", "arcaneFrostFill"},
		{"Arcane/Fire", "0502205003100311531-230510301001-003", "arcaneFireFill"},
		{"Fire (deep)", "05022050001001-23550000130135031-003", "deepFire"},
		{"Fire/Frost FFB", "-2355000013011503-2035010311013002", "ffb"},
		{"Frost (deep)", "05022050011003--053500030100331251", "frost"},
	}
	tank := player(build{"B", "5-5500013301301551-0522103", true})
	for _, sc := range []string{"1 mob 60s", "4 mobs 45s", "boss 32k hp"} {
		for _, b := range builds {
			m := mageBuild(b.talents, b.apl)
			party := &proto.Party{Players: []*proto.Player{priestPlayer("505025021330011531-00505103", "1316991"), tank, m}, Buffs: &proto.PartyBuffs{DevotionAura: true}}
			r := &proto.Raid{Parties: []*proto.Party{party}, Buffs: &proto.RaidBuffs{ArcaneBrilliance: true, PrayerOfFortitude: true}, Debuffs: &proto.Debuffs{},
				Tanks: []*proto.UnitReference{{Type: proto.UnitReference_Player, Index: 1}}}
			var e *proto.Encounter
			switch sc {
			case "1 mob 60s":
				e = encD(1, 60, 60, 350)
			case "4 mobs 45s":
				e = encD(4, 45, 60, 350)
			default:
				e = encD(1, 300, 62, 900); e.UseHealth = true
				e.Targets[0].Stats = make([]float64, 40); e.Targets[0].Stats[proto.Stat_StatHealth] = 32000
			}
			res := core.RunRaidSim(&proto.RaidSimRequest{Raid: r, Encounter: e, SimOptions: &proto.SimOptions{Iterations: 2000, RandomSeed: 9}})
			if res.Error != nil { t.Fatalf("%s: %v", b.name, res.Error.Message) }
			ps := res.RaidMetrics.Parties[0].Players
			fmt.Printf("AS|%s|%s|mageDPS=%.1f|mageTPS=%.1f|tankTPS=%.1f|oom=%.1f|trioDPS=%.1f|tankDeath=%.3f|nyxxOOM=%.1f\n", sc, b.name, ps[2].Dps.Avg, ps[2].Threat.Avg, ps[1].Threat.Avg, ps[2].SecondsOomAvg, res.RaidMetrics.Dps.Avg, ps[1].ChanceOfDeath, ps[0].SecondsOomAvg)
		}
	}
}
