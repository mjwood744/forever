package retribution

import (
	"testing"

	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/proto"
	"github.com/wowsims/forever/sim/core/simsignals"
	"github.com/wowsims/forever/sim/core/stats"
	"github.com/wowsims/forever/sim/paladin"
)

// Since the 31 October hotfixes Conviction is 3 ranks of an all-crit aura that holds while a melee
// weapon is equipped: 3/3 with the two-handed axe adds 5% physical and spell crit to the character,
// not to one weapon's attacks.
func TestConvictionHoldsWhileAMeleeWeaponIsEquipped(t *testing.T) {
	build := func(talents string) *paladin.Paladin {
		player := &proto.Player{
			Name:          "Conviction",
			Race:          proto.Race_RaceUndead,
			Class:         proto.Class_ClassPaladin,
			Equipment:     core.GetGearSet("../../../ui/specs/paladin/retribution/gear_sets", "twist_of_light").GearSet,
			TalentsString: talents,
			Spec: &proto.Player_RetributionPaladin{RetributionPaladin: &proto.RetributionPaladin{
				Options: &proto.RetributionPaladin_Options{ClassOptions: &proto.PaladinOptions{}},
			}},
			Rotation: core.GetAplRotation("../../../ui/specs/paladin/retribution/apls", "twist_of_light").Rotation,
		}
		sim := core.NewSim(&proto.RaidSimRequest{
			Raid:       core.SinglePlayerRaidProto(player, &proto.PartyBuffs{}, &proto.RaidBuffs{}, &proto.Debuffs{}),
			Encounter:  core.MakeSingleTargetEncounter(0),
			SimOptions: &proto.SimOptions{RandomSeed: 1},
		}, simsignals.CreateSignals())
		sim.Reset()
		return sim.Raid.Parties[0].Players[0].(paladin.PaladinAgent).GetPaladin()
	}

	with := build(TwistOfLightTalents)
	if with.Talents.Conviction != 3 {
		t.Fatalf("the talents take Conviction %d/3, want 3/3", with.Talents.Conviction)
	}
	without := build("")
	for _, stat := range []stats.Stat{stats.PhysicalCritPercent, stats.SpellCritPercent} {
		got := with.GetStats()[stat] - without.GetStats()[stat]
		if got < 4.999 || got > 5.001 {
			t.Errorf("Conviction 3/3 adds %0.3f to %s, want 5", got, stat.StatName())
		}
	}
}
