package trios

import (
	"fmt"
	"os"
	"testing"

	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/proto"
)

// The Nyxx + Malyce duo at 10, 20 and 30 with the races set by DUO_PRIEST_RACE / DUO_LOCK_RACE.
func TestDuoRace(t *testing.T) {
	tag := os.Getenv("DUO_PRIEST_RACE") + " Priest + " + os.Getenv("DUO_LOCK_RACE") + " Warlock"
	iters := iterations()
	for _, lvl := range []int32{10, 20, 30} {
		for _, f := range fights[:3] {
			d := makeTrio(4, lvl)
			e := encounter(f, lvl)
			stats := core.ComputeStats(&proto.ComputeStatsRequest{Raid: d.raid(), Encounter: e, SkipRotation: true}).RaidStats.Parties[0].Players
			res, _ := runLevel(d, lvl, f, iters)
			ps := res.RaidMetrics.Parties[0].Players
			line := fmt.Sprintf("DR|%s|L%d|%s|kill=%.1fs", tag, lvl, f.name, res.AvgIterationDuration)
			for i, m := range d.members {
				var net float64
				for _, r := range ps[i].Resources {
					if r.Type == proto.ResourceType_ResourceTypeMana {
						net += r.ActualGain
					}
				}
				pool := stats[i].FinalStats.Stats[proto.Stat_StatMana]
				line += fmt.Sprintf("|%s mana=%.0f%% death=%.3f", m.name, -net/float64(iters)/max(pool, 1)*100, ps[i].ChanceOfDeath)
				for pi, pet := range ps[i].Pets {
					if pet.Dtps.Avg > 0 {
						line += fmt.Sprintf(" pet%d death=%.3f", pi, pet.ChanceOfDeath)
					}
				}
			}
			fmt.Println(line)
		}
	}
}
