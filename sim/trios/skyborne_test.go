package trios

import (
	"encoding/json"

	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/proto"
)

// Trio 8, all Windshaper Skyborne: Skarr tanking on the main screen, Wyldd following and healing him, Hawke
// following and doing damage with a Cat. Party order Wyldd, Skarr, Hawke, so the healer's {main} is Skarr and
// {other} is Hawke.
func makeSkyborneTrio(lvl int32) trio {
	sk := makeTrio(2, lvl).members[1]
	sk.player.Race = proto.Race_RaceWindshaperSkyborne
	t3 := makeTrio(3, lvl)
	hk, wd := t3.members[0], t3.members[1]
	pet := proto.HunterOptions_PetNone
	if lvl >= 10 {
		pet = proto.HunterOptions_Cat
	}
	core.WithSpec(hk.player, &proto.Player_Hunter{Hunter: &proto.Hunter{Options: &proto.Hunter_Options{ClassOptions: &proto.HunterOptions{PetType: pet, PetUptime: 1}}}})
	return trio{8, []member{wd, sk, hk}, &proto.UnitReference{Type: proto.UnitReference_Player, Index: 1}, unitPlayer(1), "", ""}
}

// Wyldd's keys were written for Hawke's tanking pet: in this trio the pet's heals go to Skarr.
func skyborneTarget(raw string, t trio) string {
	var u struct {
		Type string `json:"type"`
	}
	if raw != "" && json.Unmarshal([]byte(raw), &u) == nil && u.Type == "Pet" {
		return t.tankRef
	}
	return raw
}
