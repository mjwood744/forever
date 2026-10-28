package mysim

import (
	"fmt"
	"os"
	"testing"

	"github.com/wowsims/forever/sim/common"
	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/proto"
	"github.com/wowsims/forever/sim/warrior/dps"
	"github.com/wowsims/forever/sim/warrior/protection"
)

func init() {
	protection.RegisterProtectionWarrior()
	dps.RegisterDpsWarrior()
	common.RegisterAllEffects()
}

type wb struct {
	name, talents string
	tank          bool
}

func warrior(b wb) *proto.Player {
	opts := &proto.WarriorOptions{QueueDelay: 250, UseBattleShout: true, StanceSnapshot: true, DefaultStance: proto.WarriorStance_WarriorStanceDefensive}
	base := &proto.Player{
		Class: proto.Class_ClassWarrior, Race: proto.Race_RaceOrc,
		Consumables: &proto.ConsumesSpec{}, Buffs: &proto.IndividualBuffs{},
		TalentsString: b.talents, InFrontOfTarget: true, DistanceFromTarget: 5, ReactionTimeMs: 100,
	}
	if b.tank {
		base.Equipment = core.GetGearSet("../../../ui/specs/warrior/protection/gear_sets", "preraid").GearSet
		base.Rotation = core.GetAplRotation("../../../ui/specs/warrior/protection/apls", "default").Rotation
		return core.WithSpec(base, &proto.Player_ProtectionWarrior{ProtectionWarrior: &proto.ProtectionWarrior{Options: &proto.ProtectionWarrior_Options{ClassOptions: opts}}})
	}
	opts.DefaultStance = proto.WarriorStance_WarriorStanceBattle
	base.Equipment = core.GetGearSet("../../../ui/specs/warrior/dps/gear_sets", "prebis_arms").GearSet
	base.Rotation = core.GetAplRotation("../../../ui/specs/warrior/dps/apls", "arms").Rotation
	return core.WithSpec(base, &proto.Player_DpsWarrior{DpsWarrior: &proto.DpsWarrior{Options: &proto.DpsWarrior_Options{ClassOptions: opts}}})
}

func enc(n int, mob bool, dur float64) *proto.Encounter {
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
	return &proto.Encounter{Duration: dur, DurationVariation: 5, Targets: ts}
}

func TestSkarr(t *testing.T) {
	builds := []wb{
		{"A Arms (2H, arms rotation)", "30325213132515201-0505-2", false},
		{"A Arms talents tanking (shield)", "30325213132515201-0505-2", true},
		{"B Prot (sim preset)", "35300003-05-050531120300021351", true},
		{"C Hybrid Arms 20 / Prot 31", "04305113102--050531120300011351", true},
		{"Ref Fury talents tanking", "30305013002-25053005052010501", true},
	}
	_ = os.Getenv
	for _, sc := range []struct{ n int; mob bool; lab string; dur float64 }{{1, true, "1 mob 60s", 60}, {4, true, "4 mobs 45s", 45}, {1, false, "1 boss 120s", 120}} {
		for _, b := range builds {
			r := core.SinglePlayerRaidProto(warrior(b), &proto.PartyBuffs{BloodPact: true}, &proto.RaidBuffs{StrengthOfEarthTotem: true}, &proto.Debuffs{})
			r.Tanks = append(r.Tanks, &proto.UnitReference{Type: proto.UnitReference_Player, Index: 0})
			res := core.RunRaidSim(&proto.RaidSimRequest{Raid: r, Encounter: enc(sc.n, sc.mob, sc.dur), SimOptions: &proto.SimOptions{Iterations: 3000, RandomSeed: 101}})
			if res.Error != nil { t.Fatalf("%s: %v", b.name, res.Error.Message) }
			p := res.RaidMetrics.Parties[0].Players[0]
			fmt.Printf("SK|%s|%s|dps=%.1f|tps=%.1f|dtps=%.1f\n", sc.lab, b.name, p.Dps.Avg, p.Threat.Avg, p.Dtps.Avg)
		}
	}
}
