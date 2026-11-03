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

// Each healer spec, which saves no rotation, heals the Protection Warrior through open world.
func TestHealersHeal(t *testing.T) {
	healers := map[string]*proto.Player{
		"Holy Paladin": core.WithSpec(&proto.Player{Name: "Healer", Class: proto.Class_ClassPaladin, Race: proto.Race_RaceHuman, Equipment: &proto.EquipmentSpec{}},
			&proto.Player_HolyPaladin{HolyPaladin: &proto.HolyPaladin{Options: &proto.HolyPaladin_Options{ClassOptions: &proto.PaladinOptions{}}}}),
		"Restoration Druid": core.WithSpec(&proto.Player{Name: "Healer", Class: proto.Class_ClassDruid, Race: proto.Race_RaceTauren, Equipment: &proto.EquipmentSpec{}},
			&proto.Player_RestorationDruid{RestorationDruid: &proto.RestorationDruid{Options: &proto.RestorationDruid_Options{ClassOptions: &proto.DruidOptions{}}}}),
		"Restoration Shaman": core.WithSpec(&proto.Player{Name: "Healer", Class: proto.Class_ClassShaman, Race: proto.Race_RaceTauren, Equipment: &proto.EquipmentSpec{}},
			&proto.Player_RestorationShaman{RestorationShaman: &proto.RestorationShaman{Options: &proto.RestorationShaman_Options{ClassOptions: &proto.ShamanOptions{}}}}),
		"Healer Priest": group().Parties[0].Players[2],
	}
	// A healer that brings no rotation at all heals the same way.
	healers["Holy Paladin"].Rotation = nil
	for name, healer := range healers {
		raid := group()
		raid.Parties[0].Players = []*proto.Player{raid.Parties[0].Players[0], healer}
		res := run(t, &proto.ActivitySimRequest{Raid: raid, Activity: proto.Activity_ActivityOpenWorld, Level: 60})
		if hps := res.Players[1].Hps; hps <= 0 {
			t.Errorf("%s heals %.1f per second, want above 0", name, hps)
		}
	}
}

// Untanked, the mobs hit no one, so the Mage's damage is all that counts.
func TestUntanked(t *testing.T) {
	raid := group()
	raid.Parties[0].Players = raid.Parties[0].Players[1:2]
	res := run(t, &proto.ActivitySimRequest{Raid: raid, Activity: proto.Activity_ActivityRaid, Untanked: true})
	if p := res.Players[0]; p.Dtps != 0 || p.Dps <= 0 {
		t.Errorf("untanked mage: %.1f damage taken and %.1f dps per second, want none taken and some dealt", p.Dtps, p.Dps)
	}
}

// A slot holding an item the database lacks is emptied, with a warning, and the rest still sims.
func TestDropsUnknownItems(t *testing.T) {
	raid := group()
	raid.Parties[0].Players[1].Equipment.Items[0] = &proto.ItemSpec{Id: 999999999}
	res := run(t, &proto.ActivitySimRequest{Raid: raid, Activity: proto.Activity_ActivityRaid})
	if len(res.Warnings) != 1 {
		t.Fatalf("warnings %v, want one for the unknown item", res.Warnings)
	}
}

// An item the player carries its own row for counts as known, since the sim loads those rows first.
func TestKeepsCarriedItems(t *testing.T) {
	priest := group().Parties[0].Players[2]
	priest.Equipment.Items[0] = &proto.ItemSpec{Id: 999999998}
	priest.Database = &proto.SimDatabase{Items: []*proto.SimItem{{Id: 999999998}}}
	if warnings := dropUnknownItems([]*proto.Player{priest}); len(warnings) != 0 || priest.Equipment.Items[0].Id != 999999998 {
		t.Errorf("warnings %v and head item %d, want none and the carried item kept", warnings, priest.Equipment.Items[0].Id)
	}
}
