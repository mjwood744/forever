package mysim

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/wowsims/forever/sim/common"
	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/proto"
	"github.com/wowsims/forever/sim/paladin/protection"
	"github.com/wowsims/forever/sim/paladin/retribution"
)

func init() {
	protection.RegisterProtectionPaladin()
	retribution.RegisterRetributionPaladin()
	common.RegisterAllEffects()
}

type build struct {
	name, talents string
	tank          bool
}

func player(b build) *proto.Player {
	if b.tank {
		return core.WithSpec(&proto.Player{
			Class: proto.Class_ClassPaladin, Race: proto.Race_RaceUndead,
			Equipment:     core.GetGearSet("../../../ui/specs/paladin/protection/gear_sets", "default").GearSet,
			Consumables:   &proto.ConsumesSpec{},
			Buffs:         &proto.IndividualBuffs{GreaterBlessingOfKings: true},
			TalentsString: b.talents,
			Profession1:   proto.Profession_Engineering,
			Rotation:      tankAPL(),
			InFrontOfTarget: true, DistanceFromTarget: 5, ReactionTimeMs: 100, ChannelClipDelayMs: 50,
		}, &proto.Player_ProtectionPaladin{ProtectionPaladin: &proto.ProtectionPaladin{Options: &proto.ProtectionPaladin_Options{ClassOptions: &proto.PaladinOptions{}}}})
	}
	return core.WithSpec(&proto.Player{
		Class: proto.Class_ClassPaladin, Race: proto.Race_RaceUndead,
		Equipment:     core.GetGearSet("../../../ui/specs/paladin/retribution/gear_sets", "twist_of_light").GearSet,
		Consumables:   &proto.ConsumesSpec{},
		Buffs:         &proto.IndividualBuffs{GreaterBlessingOfKings: true},
		TalentsString: b.talents,
		Profession1:   proto.Profession_Engineering,
		Rotation:      core.GetAplRotation("../../../ui/specs/paladin/retribution/apls", "twist_of_light").Rotation,
		InFrontOfTarget: true, DistanceFromTarget: 5, ReactionTimeMs: 100, ChannelClipDelayMs: 50,
	}, &proto.Player_RetributionPaladin{RetributionPaladin: &proto.RetributionPaladin{Options: &proto.RetributionPaladin_Options{ClassOptions: &proto.PaladinOptions{}}}})
}

func tankAPL() *proto.APLRotation {
	data, _ := os.ReadFile("../../../ui/specs/paladin/protection/apls/default.apl.json")
	js := strings.Replace(string(data), `"prepullActions": [`, `"prepullActions": [{"action": {"castSpell": {"spellId": {"spellId": 25780}}}, "doAtValue": {"const": {"val": "-20s"}}}, `, 1)
	return core.APLRotationFromJsonString(js)
}

func enc(n int, mob bool) *proto.Encounter {
	ts := make([]*proto.Target, n)
	for i := range ts {
		ts[i] = core.FreshDefaultTargetConfig()
		if mob {
			ts[i].Level = 60
			ts[i].MinBaseDamage = 350
			ts[i].CanCrush = false
			ts[i].ParryHaste = false
		}
	}
	return &proto.Encounter{Duration: 120, DurationVariation: 10, Targets: ts}
}

func raid(b build) *proto.Raid {
	r := core.SinglePlayerRaidProto(player(b), &proto.PartyBuffs{DevotionAura: true}, &proto.RaidBuffs{ArcaneBrilliance: true, PrayerOfFortitude: true}, &proto.Debuffs{})
	r.Tanks = append(r.Tanks, &proto.UnitReference{Type: proto.UnitReference_Player, Index: 0})
	return r
}

func TestPaladinBuilds(t *testing.T) {
	builds := []build{
		{"A Ret levelling (2H, ret rotation)", "5-50303-25225331001331301", false},
		{"A Ret talents tanking (shield, tank rotation)", "5-50303-25225331001331301", true},
		{"B Prot (shield, tank rotation)", "5-5500013301301551-0522103", true},
		{"C Hybrid (shield, tank rotation)", "-5530013301301500-0522513100101", true},
		{"Ref: sim default Prot", "50003-0530313321301531-5022", true},
		{"Ref: sim default Ret", "55003003--05225331101330321", false},
	}
	for _, sc := range []struct{ n int; mob bool; lab string }{{1, true, "1 normal mob"}, {4, true, "4 normal mobs"}, {1, false, "1 boss"}} {
		n := sc.n
		for _, b := range builds {
			rsr := &proto.RaidSimRequest{
				Raid:       raid(b),
				Encounter:  enc(n, sc.mob),
				SimOptions: &proto.SimOptions{Iterations: 3000, RandomSeed: 101},
			}
			res := core.RunRaidSim(rsr)
			if res.Error != nil {
				t.Fatalf("%s: %v", b.name, res.Error.Message)
			}
			p := res.RaidMetrics.Parties[0].Players[0]
			fmt.Printf("RESULT|%s|%s|dps=%.1f|tps=%.1f|dtps=%.1f|oom=%.1f|death=%.3f\n", sc.lab, b.name, p.Dps.Avg, p.Threat.Avg, p.Dtps.Avg, p.SecondsOomAvg, p.ChanceOfDeath)
		}
	}
}
