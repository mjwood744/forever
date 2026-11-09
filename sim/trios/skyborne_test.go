package trios

import (
	"encoding/json"
	"os"

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
	// SKY_DPS=shaman: an Elemental Shaman does the damage in Hawke's place (Wyther's keys, played with ROLE_Wyther=dps).
	if os.Getenv("SKY_DPS") == "shaman" {
		sh := makeTrio(2, lvl).members[0]
		sh.player.Race = proto.Race_RaceWindshaperSkyborne
		core.WithSpec(sh.player, &proto.Player_ElementalShaman{ElementalShaman: &proto.ElementalShaman{Options: &proto.ElementalShaman_Options{ClassOptions: &proto.ShamanOptions{}}}})
		hk = sh
	}
	// WYLDD_FERAL_GEAR=1: Wyldd heals in the solo Feral Druid's levelling gear (leather and a staff or mace).
	if os.Getenv("WYLDD_FERAL_GEAR") != "" {
		wd.player.Equipment = soloGear("cat", lvl)
	}
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
