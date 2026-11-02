package trios

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/wowsims/forever/sim/core/proto"
)

// The Nyxx + Malyce duo with Malyce's pet setup varied: PET_SAC=Succubus|Imp sacrifices that demon before the
// pull and keeps the Voidwalker out (Demonic Pact, 43+); LVLS picks the levels.
func TestDuoPetSetups(t *testing.T) {
	lvls := []int32{12, 14, 16, 18, 20, 22}
	if s := os.Getenv("LVLS"); s != "" {
		lvls = nil
		for _, x := range strings.Split(s, ",") {
			n, _ := strconv.Atoi(x)
			lvls = append(lvls, int32(n))
		}
	}
	tag := os.Getenv("TAG")
	pulls := append([]fight{}, fights...)
	pulls = append(pulls, fight{"4 mobs", 4, 0, 1, 0, false}, fight{"5 mobs", 5, 0, 1, 0, false})
	for _, lvl := range lvls {
		for _, f := range pulls {
			d := makeTrio(4, lvl)
			if v, ok := proto.WarlockOptions_Summon_value[os.Getenv("PET_SAC")]; ok {
				d.members[1].player.GetWarlock().Options.ClassOptions.SacrificedDemon = proto.WarlockOptions_Summon(v)
			}
			res, _ := runLevel(d, lvl, f, iterations())
			ps := res.RaidMetrics.Parties[0].Players
			line := fmt.Sprintf("PM|%s|L%d|%s|kill=%.1fs", tag, lvl, f.name, res.AvgIterationDuration)
			for i, m := range d.members {
				p := ps[i]
				line += fmt.Sprintf("|%s dps=%.0f oom=%.1f death=%.3f", m.name, p.Dps.Avg, p.SecondsOomAvg, p.ChanceOfDeath)
				for _, pet := range p.Pets {
					if pet.Dtps.Avg > 0 || pet.Dps.Avg > 0 {
						line += fmt.Sprintf(" pet(dps=%.0f death=%.3f)", pet.Dps.Avg, pet.ChanceOfDeath)
					}
				}
			}
			fmt.Println(line)
		}
	}
}

// Hawke + Nyxx with Hawke's pet family set by PET_FAMILY (the trio's Turtle training kept), on LVLS.
func TestHunterPetFamilies(t *testing.T) {
	lvls := []int32{20, 30, 44, 60}
	if s := os.Getenv("LVLS"); s != "" {
		lvls = nil
		for _, x := range strings.Split(s, ",") {
			n, _ := strconv.Atoi(x)
			lvls = append(lvls, int32(n))
		}
	}
	pulls := append([]fight{}, fights...)
	pulls = append(pulls, fight{"4 mobs", 4, 0, 1, 0, false})
	for _, lvl := range lvls {
		for _, f := range pulls {
			d := makeDuo("Hunter + Priest", lvl)
			if v, ok := proto.HunterOptions_PetType_value[os.Getenv("PET_FAMILY")]; ok {
				d.members[0].player.GetHunter().Options.ClassOptions.PetType = proto.HunterOptions_PetType(v)
			}
			res, _ := runLevel(d, lvl, f, iterations())
			ps := res.RaidMetrics.Parties[0].Players
			line := fmt.Sprintf("HP|%s|L%d|%s|kill=%.1fs", os.Getenv("PET_FAMILY"), lvl, f.name, res.AvgIterationDuration)
			for i, m := range d.members {
				p := ps[i]
				line += fmt.Sprintf("|%s dps=%.0f oom=%.1f death=%.3f", m.name, p.Dps.Avg, p.SecondsOomAvg, p.ChanceOfDeath)
				for _, pet := range p.Pets {
					if pet.Dtps.Avg > 0 || pet.Dps.Avg > 0 {
						line += fmt.Sprintf(" pet(dps=%.0f death=%.3f)", pet.Dps.Avg, pet.ChanceOfDeath)
					}
				}
			}
			fmt.Println(line)
		}
	}
}
