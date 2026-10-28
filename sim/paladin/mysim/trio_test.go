package mysim

import (
	"fmt"
	"os"
	"testing"

	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/proto"
	"github.com/wowsims/forever/sim/mage"
)

func init() { mage.RegisterMage() }

func magePlayer() *proto.Player {
	data, _ := os.ReadFile("../../../ui/specs/mage/dps/apls/frost.apl.json")
	return core.WithSpec(&proto.Player{
		Class: proto.Class_ClassMage, Race: proto.Race_RaceUndead,
		Equipment:     core.GetGearSet("../../../ui/specs/mage/dps/gear_sets", "p1Arcane").GearSet,
		Consumables:   &proto.ConsumesSpec{},
		Buffs:         &proto.IndividualBuffs{GreaterBlessingOfKings: true},
		TalentsString: "05022050011003--053500030100331251",
		Rotation:      core.APLRotationFromJsonString(string(data)),
		DistanceFromTarget: 25, ReactionTimeMs: 100, ChannelClipDelayMs: 50,
	}, &proto.Player_Mage{Mage: &proto.Mage{Options: &proto.Mage_Options{ClassOptions: &proto.MageOptions{DefaultMageArmor: proto.MageArmor(1)}}}})
}

func TestTrio(t *testing.T) {
	builds := []struct{ name, talents string }{
		{"A Smite Disc", "525235031030001001-30515100002-3"},
		{"B Healing Disc", "005025021330011531-335001030301"},
		{"D1 B+PiL", "505025021330011531-235001030"},
		{"D2 B+PiL+DivFury", "505025021330011531-00505103"},
	}
	tank := player(build{"B", "5-5500013301301551-0522103", true})
	for _, withMage := range []bool{false, true} {
		for _, b := range builds {
			players := []*proto.Player{priestPlayer(b.talents, "1316991"), tank}
			if withMage { players = append(players, magePlayer()) }
			party := &proto.Party{Players: players, Buffs: &proto.PartyBuffs{DevotionAura: true}}
			r := &proto.Raid{Parties: []*proto.Party{party}, Buffs: &proto.RaidBuffs{ArcaneBrilliance: true, PrayerOfFortitude: true}, Debuffs: &proto.Debuffs{},
				Tanks: []*proto.UnitReference{{Type: proto.UnitReference_Player, Index: 1}}}
			e := encD(1, 300, 62, 900)
			e.UseHealth = true
			e.Targets[0].Stats = make([]float64, 40)
			e.Targets[0].Stats[proto.Stat_StatHealth] = 32000
			res := core.RunRaidSim(&proto.RaidSimRequest{Raid: r, Encounter: e, SimOptions: &proto.SimOptions{Iterations: 2000, RandomSeed: 11}})
			if res.Error != nil { t.Fatalf("%s: %v", b.name, res.Error.Message) }
			ps := res.RaidMetrics.Parties[0].Players
			md := 0.0
			if withMage { md = ps[2].Dps.Avg }
			fmt.Printf("TR|mage=%v|%s|fight=%.1fs|priestDPS=%.1f|HPS=%.1f|oom=%.1f|mageDPS=%.1f|tankDeath=%.3f\n", withMage, b.name, 32000/max(1, res.RaidMetrics.Dps.Avg), ps[0].Dps.Avg, ps[0].Hps.Avg, ps[0].SecondsOomAvg, md, ps[1].ChanceOfDeath)
		}
	}
}
