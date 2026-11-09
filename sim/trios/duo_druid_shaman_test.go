package trios

import (
	"os"

	"github.com/wowsims/forever/sim/core/proto"
)

// Trio 22, a duo: the solo Druid tanking on the main screen (caster below 10, Bear Form from 10, Druid talents
// from SOLO_TALENTS) and a Windshaper Skyborne Shaman following, playing Wyther's keys. The Shaman heals the
// Druid (Wyther's Restoration talents). For an Elemental Shaman doing damage, give TALENTS an Elemental build
// and play it with ROLE_Wyther=dps: it keeps the Restoration spec, which differs from the Elemental one only
// in having the healing spells, so its emergency heal on the Druid works. DUO_GROUP=1 adds Ashyn as a stand-in third player for
// dungeons.
func makeDruidShamanDuo(lvl int32) trio {
	if soloTalents == nil {
		var file map[string]map[string]string
		load(os.Getenv("SOLO_TALENTS"), &file)
		soloTalents = file["druid"]
	}
	// DUO_DRUID=cat: the Druid levels in Cat Form from 20 (Bear Form 10-19), the mobs still on it
	spec := "bear"
	if os.Getenv("DUO_DRUID") == "cat" {
		spec = "cat"
	}
	form := soloForm(spec, lvl)
	os.Setenv("APL_Druid", "testdata/solo_dr_"+form+".apl.json")
	dr := makeSoloDruid(spec, lvl).members[0]
	sh := makeTrio(2, lvl).members[0]
	sh.player.Race = proto.Race_RaceWindshaperSkyborne
	ms := []member{sh, dr}
	if os.Getenv("DUO_GROUP") != "" {
		ms = append(ms, makeTrio(1, lvl).members[2])
	}
	return trio{22, ms, &proto.UnitReference{Type: proto.UnitReference_Player, Index: 1}, unitPlayer(1), "", unitPlayer(1)}
}
