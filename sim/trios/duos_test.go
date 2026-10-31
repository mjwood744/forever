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

// Duos made from the trio characters, at their per-level quest gear, talents and keys.
// Player 0 and player 1; pet tanks where the pair has one.

func petAt(owner, idx int) string {
	return fmt.Sprintf(`{"type":"Pet","index":%d,"owner":{"type":"Player","index":%d}}`, idx, owner)
}

func trioMember(n int, name string, lvl int32) *proto.Player {
	for _, m := range makeTrio(n, lvl).members {
		if m.name == name {
			return m.player
		}
	}
	panic(name)
}

func charOf(name string, lvl int32) *proto.Player {
	switch name {
	case "Nyxx", "Rellyk", "Ashyn":
		return trioMember(1, name, lvl)
	case "Wyther", "Skarr", "Malyce":
		return trioMember(2, name, lvl)
	}
	return trioMember(3, name, lvl)
}

func voidwalker(p *proto.Player, lvl int32) {
	o := p.GetWarlock().Options.ClassOptions
	o.Summon = proto.WarlockOptions_Voidwalker
	if lvl >= 20 {
		o.Armor = proto.WarlockOptions_DemonArmor
	}
}

func makeDuo(name string, lvl int32) trio {
	petTank := func(owner, idx int32) *proto.UnitReference {
		return &proto.UnitReference{Type: proto.UnitReference_Pet, Index: idx, Owner: &proto.UnitReference{Type: proto.UnitReference_Player, Index: owner}}
	}
	switch name {
	case "Hunter + Priest", "Hunter + Shaman", "Druid + Hunter":
		healer := map[string]string{"Hunter + Priest": "Nyxx", "Hunter + Shaman": "Wyther", "Druid + Hunter": "Wyldd"}[name]
		return trio{0, []member{{"Hawke", charOf("Hawke", lvl)}, {healer, charOf(healer, lvl)}}, petTank(0, 0), petAt(0, 0), petAt(0, 0), unitPlayer(0)}
	case "Hunter + Warlock":
		return trio{0, []member{{"Hawke", charOf("Hawke", lvl)}, {"Malyce", charOf("Malyce", lvl)}}, petTank(0, 0), petAt(0, 0), petAt(0, 0), unitPlayer(0)}
	case "Warlock + Priest", "Shaman + Warlock":
		healer := map[string]string{"Warlock + Priest": "Nyxx", "Shaman + Warlock": "Wyther"}[name]
		m := charOf("Malyce", lvl)
		voidwalker(m, lvl)
		return trio{0, []member{{"Malyce", m}, {healer, charOf(healer, lvl)}}, petTank(0, 1), petAt(0, 1), "", unitPlayer(0)}
	case "Paladin + Priest":
		return trio{0, []member{{"Rellyk", charOf("Rellyk", lvl)}, {"Nyxx", charOf("Nyxx", lvl)}},
			&proto.UnitReference{Type: proto.UnitReference_Player, Index: 0}, unitPlayer(0), "", unitPlayer(1)}
	}
	panic(name)
}

var duoNames = []string{"Hunter + Priest", "Warlock + Priest", "Hunter + Shaman", "Paladin + Priest", "Shaman + Warlock", "Hunter + Warlock", "Druid + Hunter"}

func TestDuoLevels(t *testing.T) {
	lvls := []int32{10, 20, 30}
	if s := os.Getenv("LVLS"); s != "" {
		lvls = nil
		for _, x := range strings.Split(s, ",") {
			n, _ := strconv.Atoi(x)
			lvls = append(lvls, int32(n))
		}
	}
	iters := iterations()
	for _, lvl := range lvls {
		for _, f := range fights[:3] {
			for _, dn := range duoNames {
				d := makeDuo(dn, lvl)
				e := encounter(f, lvl)
				stats := core.ComputeStats(&proto.ComputeStatsRequest{Raid: d.raid(), Encounter: e, SkipRotation: true}).RaidStats.Parties[0].Players
				res, dropped := runLevel(d, lvl, f, iters)
				for k := range dropped {
					fmt.Printf("DROPPED|%s|L%d %s\n", dn, lvl, k)
				}
				ps := res.RaidMetrics.Parties[0].Players
				line := fmt.Sprintf("DUOL|L%d|%s|%s|kill=%.1fs", lvl, f.name, dn, res.AvgIterationDuration)
				for i, m := range d.members {
					var net float64
					for _, r := range ps[i].Resources {
						if r.Type == proto.ResourceType_ResourceTypeMana {
							net += r.ActualGain
						}
					}
					pool := stats[i].FinalStats.Stats[proto.Stat_StatMana]
					pct := 0.0
					if pool > 0 {
						pct = -net / float64(iters) / pool * 100
					}
					line += fmt.Sprintf("|%s mana=%.0f%% death=%.3f", m.name, pct, ps[i].ChanceOfDeath)
					for pi, pet := range ps[i].Pets {
						if pet.Dtps.Avg > 0 {
							line += fmt.Sprintf(" pet%d(death=%.3f)", pi, pet.ChanceOfDeath)
						}
					}
				}
				fmt.Println(line)
			}
		}
	}
}
