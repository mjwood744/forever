package trios

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/proto"
)

// A Druid levelling on its own: one Windshaper Skyborne Druid, tanking its own pulls, at every level.
// SOLO_SPEC picks the build and the way it plays:
//   - cat: caster below 10, Bear Form 10-19, Cat Form from 20 (the mob faces the cat, so no Shred)
//   - bear: caster below 10, Bear Form from 10
//   - balance: caster, Moonkin Form once the talent is taken
//   - resto: caster with the healing spells, healing itself below 40% health
//
// SOLO_TALENTS names a talents_arch_ file whose "druid" entry holds the points at each level.
// Cat and bear wear the hunter's leather and the best staff or two-handed mace of the level;
// the casters wear Wyldd's caster gear. Each line gives what the levelling sheet needs: kill time,
// mana spent, the mana pool, Spirit and mp5, the damage taken and the health pool.

var soloTalents map[string]string

func soloForm(spec string, lvl int32) string {
	switch spec {
	case "cat":
		switch {
		case lvl >= 20:
			return "cat"
		case lvl >= 10:
			return "bear"
		}
		return "caster"
	case "catc":
		// caster until Cat Form at 20
		if lvl >= 20 {
			return "cat"
		}
		return "caster"
	case "bear":
		if lvl >= 10 {
			return "bear"
		}
		return "caster"
	case "balance":
		return "balance"
	}
	return "caster"
}

// The best druid two-hander (staff or mace) any character's quest gear holds at or below the level.
func feralWeapon(lvl int32) int32 {
	var best int32
	bestDps := 0.0
	for _, byLvl := range questGear {
		for l := int32(1); l <= lvl; l++ {
			for _, id := range byLvl[strconv.Itoa(int(l))] {
				for _, it := range gearItems.Items {
					if it.Id != id || it.Type != proto.ItemType_ItemTypeWeapon || it.HandType != proto.HandType_HandTypeTwoHand {
						continue
					}
					if it.WeaponType != proto.WeaponType_WeaponTypeStaff && it.WeaponType != proto.WeaponType_WeaponTypeMace {
						continue
					}
					so := it.ScalingOptions[0]
					if so == nil || it.WeaponSpeed == 0 {
						continue
					}
					if dps := (so.WeaponDamageMin + so.WeaponDamageMax) / 2 / it.WeaponSpeed; dps > bestDps {
						best, bestDps = id, dps
					}
				}
			}
		}
	}
	return best
}

func soloGear(form string, lvl int32) *proto.EquipmentSpec {
	src := "Wyldd"
	if form == "cat" || form == "bear" {
		src = "Hawke"
	}
	var items []*proto.ItemSpec
	for _, id := range questGear[src][strconv.Itoa(int(lvl))] {
		for _, it := range gearItems.Items {
			if it.Id == id {
				if src == "Hawke" && (it.Type == proto.ItemType_ItemTypeWeapon || it.Type == proto.ItemType_ItemTypeRanged) {
					break
				}
				items = append(items, &proto.ItemSpec{Id: id})
			}
		}
	}
	if src == "Hawke" {
		if w := feralWeapon(lvl); w != 0 {
			items = append(items, &proto.ItemSpec{Id: w})
		}
	}
	return &proto.EquipmentSpec{Items: items}
}

func makeSoloDruid(spec string, lvl int32) trio {
	form := soloForm(spec, lvl)
	p := &proto.Player{
		Name: "Druid", Class: proto.Class_ClassDruid, Race: proto.Race_RaceWindshaperSkyborne, Level: lvl,
		Equipment: soloGear(form, lvl), Database: gearItems, Consumables: baseConsumes(), Buffs: &proto.IndividualBuffs{},
		TalentsString:  soloTalents[strconv.Itoa(int(lvl))],
		ReactionTimeMs: 100, ChannelClipDelayMs: 50,
		InFrontOfTarget: true, DistanceFromTarget: 5,
	}
	opts := &proto.DruidOptions{}
	switch form {
	case "cat":
		core.WithSpec(p, &proto.Player_FeralCatDruid{FeralCatDruid: &proto.FeralCatDruid{Options: &proto.FeralCatDruid_Options{ClassOptions: opts}}})
	case "bear":
		core.WithSpec(p, &proto.Player_FeralBearDruid{FeralBearDruid: &proto.FeralBearDruid{Options: &proto.FeralBearDruid_Options{ClassOptions: opts}}})
	case "balance":
		core.WithSpec(p, &proto.Player_BalanceDruid{BalanceDruid: &proto.BalanceDruid{Options: &proto.BalanceDruid_Options{ClassOptions: opts}}})
	default:
		// the healing spells come with the Restoration spec, which also registers the Balance spells
		core.WithSpec(p, &proto.Player_RestorationDruid{RestorationDruid: &proto.RestorationDruid{Options: &proto.RestorationDruid_Options{ClassOptions: opts}}})
	}
	return trio{20, []member{{"Druid", p}}, &proto.UnitReference{Type: proto.UnitReference_Player, Index: 0}, unitSelf(), "", ""}
}

func TestSoloDruid(t *testing.T) {
	spec := os.Getenv("SOLO_SPEC")
	var file map[string]map[string]string
	load(os.Getenv("SOLO_TALENTS"), &file)
	soloTalents = file["druid"]
	from, to := int32(6), int32(60)
	if s := os.Getenv("FROM"); s != "" {
		n, _ := strconv.Atoi(s)
		from = int32(n)
	}
	if s := os.Getenv("TO"); s != "" {
		n, _ := strconv.Atoi(s)
		to = int32(n)
	}
	pulls := []fight{{"1 mob", 1, 0, 1, 0, false}, {"2 mobs", 2, 0, 1, 0, false}, {"3 mobs", 3, 0, 1, 0, false}}
	if os.Getenv("SOLO_SINGLE") != "" {
		pulls = pulls[:1]
	}
	if os.Getenv("SOLO_BOSS") != "" {
		pulls = fights[2:]
	}
	iters := iterations()
	for lvl := from; lvl <= to; lvl++ {
		form := soloForm(spec, lvl)
		os.Setenv("APL_Druid", "testdata/solo_dr_"+form+".apl.json")
		if v := os.Getenv("SOLO_APL_" + strings.ToUpper(form)); v != "" {
			os.Setenv("APL_Druid", v)
		}
		for _, f := range pulls {
			d := makeSoloDruid(spec, lvl)
			e := encounter(f, lvl)
			stats := core.ComputeStats(&proto.ComputeStatsRequest{Raid: d.raid(), Encounter: e, SkipRotation: true}).RaidStats.Parties[0].Players
			res, dropped := runLevel(d, lvl, f, iters)
			for k := range dropped {
				fmt.Printf("DROPPED|%s|L%d %s\n", spec, lvl, k)
			}
			p := res.RaidMetrics.Parties[0].Players[0]
			var net float64
			for _, r := range p.Resources {
				if r.Type == proto.ResourceType_ResourceTypeMana {
					net += r.ActualGain
				}
			}
			st := stats[0].FinalStats.Stats
			taken := (p.Dtps.Avg - p.Hps.Avg) * res.AvgIterationDuration
			fmt.Printf("SOLO|%s|%s|L%d|%s|kill=%.2f|mana=%.0f|pool=%.0f|spirit=%.0f|mp5=%.0f|taken=%.0f|hp=%.0f|death=%.3f|oom=%.1f|dps=%.1f\n",
				spec, form, lvl, f.name, res.AvgIterationDuration, net/float64(iters), st[proto.Stat_StatMana], st[proto.Stat_StatSpirit],
				st[proto.Stat_StatMP5], max(taken, 0), st[proto.Stat_StatHealth], p.ChanceOfDeath, p.SecondsOomAvg, p.Dps.Avg)
		}
	}
}

// Trio 21: the solo Druid tanking a dungeon in Bear Form (Cat Form talents, SOLO_TALENTS), Nyxx healing it
// and Ashyn doing damage. The Druid plays the bear keys (APL_Druid).
func makeDruidTankTrio(lvl int32) trio {
	if soloTalents == nil {
		var file map[string]map[string]string
		load(os.Getenv("SOLO_TALENTS"), &file)
		soloTalents = file["druid"]
	}
	if os.Getenv("APL_Druid") == "" {
		os.Setenv("APL_Druid", "testdata/solo_dr_bear.apl.json")
	}
	t1 := makeTrio(1, lvl)
	dr := makeSoloDruid("bear", max(lvl, 10)).members[0]
	dr.player.Level = lvl
	return trio{21, []member{t1.members[0], dr, t1.members[2]}, &proto.UnitReference{Type: proto.UnitReference_Player, Index: 1}, unitPlayer(1), "", ""}
}
