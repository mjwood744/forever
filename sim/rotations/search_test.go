package rotations

import (
	"testing"

	"github.com/wowsims/forever/sim"
	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/proto"
)

func init() {
	sim.RegisterAll()
}

const (
	frostbolt = 10181 // Rank 10, the highest the game files give
	fireBlast = 10199 // Rank 7
)

func cast(id int32) *proto.APLListItem {
	return &proto.APLListItem{Action: &proto.APLAction{Action: &proto.APLAction_CastSpell{CastSpell: &proto.APLActionCastSpell{
		SpellId: &proto.ActionID{RawId: &proto.ActionID_SpellId{SpellId: id}},
	}}}}
}

// A level-60 mage in the spec page's gear casting only the list given.
func mageRequest(items ...*proto.APLListItem) *proto.RaidSimRequest {
	mage := core.WithSpec(&proto.Player{
		Name: "Mage", Class: proto.Class_ClassMage, Race: proto.Race_RaceGnome,
		Equipment: core.GetGearSet("../../ui/specs/mage/dps/gear_sets", "p1Arcane").GearSet,
		Rotation:  &proto.APLRotation{Type: proto.APLRotation_TypeAPL, PriorityList: items},
	}, &proto.Player_Mage{Mage: &proto.Mage{Options: &proto.Mage_Options{ClassOptions: &proto.MageOptions{}}}})
	return &proto.RaidSimRequest{
		Raid: &proto.Raid{
			Parties: []*proto.Party{{Players: []*proto.Player{mage}, Buffs: &proto.PartyBuffs{}}},
			Buffs:   &proto.RaidBuffs{}, Debuffs: &proto.Debuffs{},
		},
		Encounter:  core.MakeSingleTargetEncounter(0),
		SimOptions: &proto.SimOptions{RandomSeed: 3},
	}
}

func search(t *testing.T, req *proto.RaidSimRequest) *proto.RotationSearchResult {
	t.Helper()
	res := Search(&proto.RotationSearchRequest{Request: req, Goal: proto.RotationGoal_RotationGoalDps, Iterations: 30})
	if res.Error != nil {
		t.Fatalf("search failed: %s", res.Error.Message)
	}
	return res
}

// Fire Blast alone leaves the mage idle between its cooldowns; the search adds a spell to cast between.
func TestSearchAddsAFiller(t *testing.T) {
	res := search(t, mageRequest(cast(fireBlast)))
	if res.Score < res.StartScore*1.5 {
		t.Errorf("dps %.1f from %.1f, want the search to add a filler; changes %v", res.Score, res.StartScore, res.Changes)
	}
	if len(res.Rotation.PriorityList) < 2 {
		t.Errorf("rotation found has %d actions, want Fire Blast and a filler", len(res.Rotation.PriorityList))
	}
}

// Frostbolt above Fire Blast never leaves a gap for Fire Blast; the search moves Fire Blast above it.
func TestSearchMovesACooldownAboveTheFiller(t *testing.T) {
	res := search(t, mageRequest(cast(frostbolt), cast(fireBlast)))
	if res.Score <= res.StartScore {
		t.Fatalf("dps %.1f from %.1f, want a gain; changes %v", res.Score, res.StartScore, res.Changes)
	}
	at := map[int32]int{}
	for i, it := range res.Rotation.PriorityList {
		at[castSpellID(it)] = i
	}
	if at[fireBlast] > at[frostbolt] {
		t.Errorf("Fire Blast at %d, Frostbolt at %d, want Fire Blast first; changes %v", at[fireBlast], at[frostbolt], res.Changes)
	}
}

func TestSearchNeedsAList(t *testing.T) {
	res := Search(&proto.RotationSearchRequest{Request: mageRequest()})
	if res.Error == nil {
		t.Errorf("a player with an empty list got a rotation, want an error")
	}
}
