package activities

import (
	"testing"

	"github.com/wowsims/forever/sim"
	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/proto"
)

func init() {
	sim.RegisterAll()
}

const specs = "../../ui/specs/"

// A Protection Warrior, a Mage and a Holy Priest in their spec pages' gear and rotations.
func group() *proto.Raid {
	warrior := core.WithSpec(&proto.Player{
		Name: "Warrior", Class: proto.Class_ClassWarrior, Race: proto.Race_RaceOrc,
		Equipment: core.GetGearSet(specs+"warrior/protection/gear_sets", "preraid").GearSet,
		Rotation:  core.GetAplRotation(specs+"warrior/protection/apls", "default").Rotation,
	}, &proto.Player_ProtectionWarrior{ProtectionWarrior: &proto.ProtectionWarrior{Options: &proto.ProtectionWarrior_Options{
		ClassOptions: &proto.WarriorOptions{DefaultStance: proto.WarriorStance_WarriorStanceDefensive},
	}}})
	mage := core.WithSpec(&proto.Player{
		Name: "Mage", Class: proto.Class_ClassMage, Race: proto.Race_RaceGnome,
		Equipment: core.GetGearSet(specs+"mage/dps/gear_sets", "p1Arcane").GearSet,
		Rotation:  core.GetAplRotation(specs+"mage/dps/apls", "frost").Rotation,
	}, &proto.Player_Mage{Mage: &proto.Mage{Options: &proto.Mage_Options{ClassOptions: &proto.MageOptions{}}}})
	priest := core.WithSpec(&proto.Player{
		Name: "Priest", Class: proto.Class_ClassPriest, Race: proto.Race_RaceDwarf,
		Equipment: core.GetGearSet(specs+"priest/healer/gear_sets", "preraid").GearSet,
		Rotation:  &proto.APLRotation{Type: proto.APLRotation_TypeAPL},
	}, &proto.Player_HealerPriest{HealerPriest: &proto.HealerPriest{Options: &proto.HealerPriest_Options{ClassOptions: &proto.PriestOptions{}}}})
	return &proto.Raid{
		Parties: []*proto.Party{{Players: []*proto.Player{warrior, mage, priest}, Buffs: &proto.PartyBuffs{}}},
		Buffs:   &proto.RaidBuffs{}, Debuffs: &proto.Debuffs{},
	}
}

func run(t *testing.T, req *proto.ActivitySimRequest) *proto.ActivitySimResult {
	t.Helper()
	req.SimOptions = &proto.SimOptions{Iterations: 20, RandomSeed: 7}
	res := RunActivitySim(req)
	if res.Error != nil {
		t.Fatalf("activity sim failed: %s", res.Error.Message)
	}
	return res
}

func TestOpenWorld(t *testing.T) {
	res := run(t, &proto.ActivitySimRequest{Raid: group(), Activity: proto.Activity_ActivityOpenWorld, Level: 60})
	if len(res.Fights) != 4 {
		t.Fatalf("open world has %d fights, want 4", len(res.Fights))
	}
	if len(res.Players) != 3 {
		t.Fatalf("summary holds %d players, want 3", len(res.Players))
	}
	warrior, mage := res.Players[0], res.Players[1]
	if mage.Dps <= 0 || res.GroupDps < mage.Dps {
		t.Errorf("mage dps %.1f, group dps %.1f: want the mage above 0 and within the group", mage.Dps, res.GroupDps)
	}
	// With no tank named, the Protection Warrior holds the mobs.
	if warrior.Dtps <= mage.Dtps {
		t.Errorf("warrior takes %.1f damage per second, mage %.1f: want the warrior tanking", warrior.Dtps, mage.Dtps)
	}
	if warrior.Tps <= 0 {
		t.Errorf("warrior threat per second is %.1f, want above 0", warrior.Tps)
	}
}

func TestDungeon(t *testing.T) {
	ds := Dungeons(&proto.ActivityDungeonsRequest{Level: 60}).Dungeons
	if len(ds) != len(dungeons) {
		t.Fatalf("a level-60 group sees %d dungeons, want all %d", len(ds), len(dungeons))
	}
	for _, d := range Dungeons(&proto.ActivityDungeonsRequest{Level: 20}).Dungeons {
		if d.LastBossLevel < 15 || d.LastBossLevel > 25 {
			t.Errorf("a level-20 group sees %s, whose last boss is level %d", d.Name, d.LastBossLevel)
		}
	}
	last := ds[len(ds)-1]
	res := run(t, &proto.ActivitySimRequest{Raid: group(), Activity: proto.Activity_ActivityDungeon, Dungeon: last.Name})
	if int32(len(res.Fights)) != last.Pulls {
		t.Fatalf("%s ran %d fights, want its %d pulls", last.Name, len(res.Fights), last.Pulls)
	}
}

func TestRaid(t *testing.T) {
	res := run(t, &proto.ActivitySimRequest{Raid: group(), Activity: proto.Activity_ActivityRaid})
	if len(res.Fights) != 1 || !res.Fights[0].Boss {
		t.Fatalf("raid ran %d fights, want one boss", len(res.Fights))
	}
}

func TestRejectsBadRequests(t *testing.T) {
	six := group()
	ps := six.Parties[0].Players
	six.Parties[0].Players = append(ps, ps...)
	for name, req := range map[string]*proto.ActivitySimRequest{
		"six players":     {Raid: six},
		"unknown dungeon": {Raid: group(), Activity: proto.Activity_ActivityDungeon, Dungeon: "Nowhere"},
		"no group":        {},
	} {
		if res := RunActivitySim(req); res.Error == nil {
			t.Errorf("%s: want an error", name)
		}
	}
}
