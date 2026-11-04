package rotations

import (
	"testing"

	"github.com/wowsims/forever/sim/core/proto"
)

func spell(id int32) *proto.ActionID {
	return &proto.ActionID{RawId: &proto.ActionID_SpellId{SpellId: id}}
}

func castWhen(id int32, cond *proto.APLValue) *proto.APLListItem {
	it := cast(id)
	it.Action.Condition = cond
	return it
}

func constant(v string) *proto.APLValue {
	return &proto.APLValue{Value: &proto.APLValue_Const{Const: &proto.APLValueConst{Val: v}}}
}

// Each shown action is listed with its condition in words, and an action a switch turns off is left out.
func TestDescribeSaysWhenEachActionRuns(t *testing.T) {
	manaLow := &proto.APLValue{Value: &proto.APLValue_Cmp{Cmp: &proto.APLValueCompare{
		Op:  proto.APLValueCompare_OpLe,
		Lhs: &proto.APLValue{Value: &proto.APLValue_CurrentManaPercent{CurrentManaPercent: &proto.APLValueCurrentManaPercent{}}},
		Rhs: constant("10%"),
	}}}
	notReady := &proto.APLValue{Value: &proto.APLValue_Not{Not: &proto.APLValueNot{Val: &proto.APLValue{
		Value: &proto.APLValue_SpellIsReady{SpellIsReady: &proto.APLValueSpellIsReady{SpellId: spell(fireBlast)}},
	}}}}
	off := &proto.APLValue{Value: &proto.APLValue_VariableRef{VariableRef: &proto.APLValueVariableRef{Name: "Use Fire Blast"}}}
	hidden := cast(frostbolt)
	hidden.Hide = true
	rot := &proto.APLRotation{
		PriorityList:   []*proto.APLListItem{castWhen(fireBlast, off), castWhen(frostbolt, manaLow), castWhen(frostbolt, notReady), cast(fireBlast), hidden},
		ValueVariables: []*proto.APLValueVariable{{Name: "Use Fire Blast", Value: constant("false")}},
	}
	got := newDescriber(rot).steps(rot.PriorityList, 0)
	want := []struct{ ability, when string }{
		{"Frostbolt", "your mana is at most 10%"},
		{"Frostbolt", "Fire Blast is on cooldown"},
		{"Fire Blast", ""},
	}
	if len(got) != len(want) {
		t.Fatalf("%d steps %v, want %d", len(got), got, len(want))
	}
	for i, w := range want {
		if got[i].Ability != w.ability || got[i].When != w.when {
			t.Errorf("step %d is %q when %q, want %q when %q", i+1, got[i].Ability, got[i].When, w.ability, w.when)
		}
	}
}
