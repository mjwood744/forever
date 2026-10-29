package trios

import (
	"fmt"
	"os"
	"strconv"
	"testing"

	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/proto"
)

// Each trio at every level from 6 to 59 on the pulls that level you: 1 mob and 3 mobs. Each line gives the
// kill time and, per character, the mana a pull costs, the mana pool, Spirit and mp5, which the levelling
// sheet turns into drinking time and time to level.
func TestTrioLevelling(t *testing.T) {
	trios := []int{1, 2, 3}
	if s := os.Getenv("TRIO"); s != "" {
		n, _ := strconv.Atoi(s)
		trios = []int{n}
	}
	from, to := int32(6), int32(59)
	if s := os.Getenv("FROM"); s != "" {
		n, _ := strconv.Atoi(s)
		from = int32(n)
	}
	if s := os.Getenv("TO"); s != "" {
		n, _ := strconv.Atoi(s)
		to = int32(n)
	}
	iters := iterations()
	for _, n := range trios {
		for lvl := from; lvl <= to; lvl++ {
			for _, f := range fights[:2] {
				tr := makeTrio(n, lvl)
				e := encounter(f, lvl)
				stats := core.ComputeStats(&proto.ComputeStatsRequest{Raid: tr.raid(), Encounter: e, SkipRotation: true}).RaidStats.Parties[0].Players
				res, dropped := runLevel(tr, lvl, f, iters)
				for k := range dropped {
					fmt.Printf("DROPPED|T%d|L%d %s\n", n, lvl, k)
				}
				ps := res.RaidMetrics.Parties[0].Players
				line := fmt.Sprintf("LVLG|T%d|L%d|%s|kill=%.2f", n, lvl, f.name, res.AvgIterationDuration)
				for i, m := range tr.members {
					var net float64
					for _, r := range ps[i].Resources {
						if r.Type == proto.ResourceType_ResourceTypeMana {
							net += r.ActualGain
						}
					}
					st := stats[i].FinalStats.Stats
					line += fmt.Sprintf("|%s class=%d mana=%.0f pool=%.0f spirit=%.0f mp5=%.0f death=%.3f oom=%.1f",
						m.name, m.player.Class, net/float64(iters), st[proto.Stat_StatMana], st[proto.Stat_StatSpirit],
						st[proto.Stat_StatMP5], ps[i].ChanceOfDeath, ps[i].SecondsOomAvg)
				}
				fmt.Println(line)
			}
		}
	}
}
