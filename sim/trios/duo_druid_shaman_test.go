package trios

import (
	"os"
	"strings"

	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/proto"
	"github.com/wowsims/forever/sim/core/stats"
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
	tr := trio{22, ms, &proto.UnitReference{Type: proto.UnitReference_Player, Index: 1}, unitPlayer(1), "", unitPlayer(1)}
	addCamp(tr)
	return tr
}

// CAMP=item,item gives every member the Camp Benefits buff of those tier 1 camp items, values from their
// "Boosted ..." spells: chair 1229519 (+2% weapon crit), faid 1230124 (+56 Stamina), candle 1229513 (+25
// Intellect), well 1230587 (29 mana every 5 s), lodestone 1230164 (+90 attack power), lute 1230653 (+308
// armour), banner 1229718 (+32 Spirit), bowl 1230098 (+8% all stats, added as 8% of the computed stats).
// The Camp Tent gives rested XP only, no combat stats.
func addCamp(tr trio) {
	camp := os.Getenv("CAMP")
	if camp == "" {
		return
	}
	bonus := stats.Stats{}
	var crit float64
	bowl := false
	for _, it := range strings.Split(camp, ",") {
		switch it {
		case "chair":
			crit += 2
		case "faid":
			bonus[stats.Stamina] += 56
		case "candle":
			bonus[stats.Intellect] += 25
		case "well":
			bonus[stats.MP5] += 29
		case "lodestone":
			bonus[stats.AttackPower] += 90
		case "lute":
			bonus[stats.BonusArmor] += 308
		case "banner":
			bonus[stats.Spirit] += 32
		case "bowl":
			bowl = true
		case "tent", "none":
		default:
			panic("CAMP: unknown item " + it)
		}
	}
	set := func(m member, b stats.Stats) {
		us := &proto.UnitStats{Stats: b.ToProtoArray(), PseudoStats: make([]float64, len(proto.PseudoStat_name))}
		us.PseudoStats[proto.PseudoStat_PseudoStatMeleeCritPercent] = crit
		m.player.BonusStats = us
	}
	for _, m := range tr.members {
		set(m, bonus)
	}
	if !bowl {
		return
	}
	ps := core.ComputeStats(&proto.ComputeStatsRequest{Raid: tr.raid(), SkipRotation: true}).RaidStats.Parties[0].Players
	for i, m := range tr.members {
		b := bonus
		for _, s := range []stats.Stat{stats.Strength, stats.Agility, stats.Stamina, stats.Intellect, stats.Spirit} {
			b[s] += 0.08 * ps[i].FinalStats.Stats[s]
		}
		set(m, b)
	}
}
