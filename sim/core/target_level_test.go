package core

import (
	"math"
	"testing"

	"github.com/wowsims/forever/sim/core/proto"
	"github.com/wowsims/forever/sim/core/stats"
)

func TestTargetsKeepTheirGapBelowSixty(t *testing.T) {
	st := make([]float64, stats.UnitStatsLen)
	st[stats.Armor] = CreatureArmorMaxLevel
	boss := &proto.Target{Level: 63, Stats: st, MinBaseDamage: 1000}
	mob := &proto.Target{Level: 20}

	got := targetsForLevel([]*proto.Target{boss, mob}, 20)
	if got[0].Level != 23 {
		t.Errorf("boss level %d against level-20 players, want 23", got[0].Level)
	}
	_, armor, dps := CreatureStatsAt(23)
	if math.Abs(got[0].Stats[stats.Armor]-armor) > 1e-6 {
		t.Errorf("boss armor %.0f, want the level-23 creature armor %.0f", got[0].Stats[stats.Armor], armor)
	}
	if want := 1000 * dps / CreatureAutoAttackDpsMaxLevel; math.Abs(got[0].MinBaseDamage-want) > 1e-6 {
		t.Errorf("boss base damage %.1f, want %.1f", got[0].MinBaseDamage, want)
	}
	if got[1] != mob {
		t.Errorf("a level-20 mob was changed, want it left as set")
	}
	if boss.Level != 63 || boss.Stats[stats.Armor] != CreatureArmorMaxLevel {
		t.Errorf("the request's boss was changed, want a scaled copy")
	}
	if at60 := targetsForLevel([]*proto.Target{boss}, 60); at60[0] != boss {
		t.Errorf("level-60 players got a changed boss, want it as set")
	}
}
