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

type dungeonMob struct {
	Name   string  `json:"name"`
	Level  int32   `json:"level"`
	Health float64 `json:"health"`
	Armor  float64 `json:"armor"`
	Dps    float64 `json:"dps"`
	Swing  float64 `json:"swing"`
}

type dungeonPull struct {
	Label string       `json:"label"`
	Count int          `json:"count"`
	Boss  bool         `json:"boss"`
	Mobs  []dungeonMob `json:"mobs"`
}

type dungeon struct {
	LastBoss int32         `json:"lastBoss"`
	Pulls    []dungeonPull `json:"pulls"`
}

// A pull's mobs as the encounter's targets, each with its own level, health, armor and melee.
func pullEncounter(p dungeonPull) *proto.Encounter {
	ts := make([]*proto.Target, len(p.Mobs))
	for i, m := range p.Mobs {
		st := make([]float64, proto.Stat_StatMana+1)
		st[proto.Stat_StatHealth] = m.Health
		st[proto.Stat_StatArmor] = m.Armor
		ts[i] = &proto.Target{
			Name: m.Name, Level: m.Level, Stats: st, MobType: proto.MobType_MobTypeHumanoid,
			SwingSpeed: m.Swing, DamageSpread: 0.4, MinBaseDamage: m.Dps * m.Swing / 1.2,
		}
	}
	return &proto.Encounter{Duration: 300, UseHealth: true, Targets: ts}
}

// Each trio through each dungeon's pulls at the last boss's level, 2 over and 4 over. Each line gives the
// pull's kill time, each character's chance of dying and the mana the pull costs. DUNGEONS=forever_dungeons.json
// runs the stand-ins for Forever's new dungeons instead.
func TestTrioDungeons(t *testing.T) {
	var dungeons map[string]dungeon
	file := "dungeons.json"
	if s := os.Getenv("DUNGEONS"); s != "" {
		file = s
	}
	load(file, &dungeons)
	trios := []int{1, 2, 3}
	if s := os.Getenv("TRIO"); s != "" {
		n, _ := strconv.Atoi(s)
		trios = []int{n}
	}
	only := os.Getenv("DUNGEON")
	// FROM and TO keep only the runs at those trio levels.
	from, to := int32(0), int32(60)
	if s := os.Getenv("FROM"); s != "" {
		n, _ := strconv.Atoi(s)
		from = int32(n)
	}
	if s := os.Getenv("TO"); s != "" {
		n, _ := strconv.Atoi(s)
		to = int32(n)
	}
	iters := iterations()
	for name, d := range dungeons {
		if only != "" && !strings.Contains(name, only) {
			continue
		}
		for _, n := range trios {
			seen := map[int32]bool{}
			for _, over := range []int32{0, 2, 4} {
				lvl := min(d.LastBoss+over, 60)
				if seen[lvl] || lvl < from || lvl > to {
					continue
				}
				seen[lvl] = true
				for _, p := range d.Pulls {
					tr := makeTrio(n, lvl)
					e := pullEncounter(p)
					f := fight{name: p.Label, count: len(p.Mobs), boss: p.Boss}
					stats := core.ComputeStats(&proto.ComputeStatsRequest{Raid: tr.raid(), Encounter: e, SkipRotation: true}).RaidStats.Parties[0].Players
					have := registered(tr.raid(), e)
					dropped := map[string]bool{}
					for i, m := range tr.members {
						m.player.Rotation = core.APLRotationFromJsonString(buildAPL(m.name, tr, lvl, f, have[i], dropped))
					}
					res := core.RunRaidSim(&proto.RaidSimRequest{Raid: tr.raid(), Encounter: e, SimOptions: &proto.SimOptions{Iterations: iters, RandomSeed: 61}})
					ps := res.RaidMetrics.Parties[0].Players
					var lv []string
					for _, m := range p.Mobs {
						lv = append(lv, strconv.Itoa(int(m.Level)))
					}
					line := fmt.Sprintf("DGN|%s|T%d|L%d|%s|count=%d|boss=%t|mobs=%s|kill=%.2f", name, n, lvl, p.Label, p.Count, p.Boss, strings.Join(lv, ","), res.AvgIterationDuration)
					for i, m := range tr.members {
						var net float64
						for _, r := range ps[i].Resources {
							if r.Type == proto.ResourceType_ResourceTypeMana {
								net += r.ActualGain
							}
						}
						death := ps[i].ChanceOfDeath
						for _, pet := range ps[i].Pets {
							if pet.Name == "Turtle" || strings.Contains(pet.Name, "Pet") {
								death = max(death, pet.ChanceOfDeath)
							}
						}
						st := stats[i].FinalStats.Stats
						line += fmt.Sprintf("|%s mana=%.0f pool=%.0f spirit=%.0f mp5=%.0f death=%.3f", m.name, net/float64(iters),
							st[proto.Stat_StatMana], st[proto.Stat_StatSpirit], st[proto.Stat_StatMP5], death)
					}
					fmt.Println(line)
				}
			}
		}
	}
}
