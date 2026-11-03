// Package activities runs a group of 1 to 5 players through the fights of an activity (open world,
// a dungeon's pulls, a raid boss) and sums each player's results over them.
package activities

import (
	"embed"
	"encoding/json"
	"fmt"
	"slices"
	"sort"

	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/proto"
	googleProto "google.golang.org/protobuf/proto"
)

// The dungeons' pulls, mob by mob: forever.json holds the dungeons new to Forever, classic.json the rest.
//
//go:embed dungeons/*.json
var dungeonFiles embed.FS

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
	Boss  bool         `json:"boss"`
	Mobs  []dungeonMob `json:"mobs"`
}

type dungeon struct {
	LastBoss int32         `json:"lastBoss"`
	Pulls    []dungeonPull `json:"pulls"`
}

var dungeons = loadDungeons()

func loadDungeons() map[string]dungeon {
	all := map[string]dungeon{}
	for _, file := range []string{"dungeons/forever.json", "dungeons/classic.json"} {
		data, err := dungeonFiles.ReadFile(file)
		if err != nil {
			panic(err)
		}
		var ds map[string]dungeon
		if err := json.Unmarshal(data, &ds); err != nil {
			panic(fmt.Sprintf("%s: %v", file, err))
		}
		for name, d := range ds {
			all[name] = d
		}
	}
	return all
}

// The dungeons an ActivitySimRequest can name for a group of the level, by the level of their last boss.
// A level-60 group can run every dungeon.
func Dungeons(req *proto.ActivityDungeonsRequest) *proto.ActivityDungeonsResult {
	res := &proto.ActivityDungeonsResult{}
	for name, d := range dungeons {
		if lvl := req.GetLevel(); lvl > 0 && lvl < core.CharacterLevel && (d.LastBoss < lvl-5 || d.LastBoss > lvl+5) {
			continue
		}
		res.Dungeons = append(res.Dungeons, &proto.ActivityDungeonInfo{Name: name, LastBossLevel: d.LastBoss, Pulls: int32(len(d.Pulls))})
	}
	sort.Slice(res.Dungeons, func(i, j int) bool {
		a, b := res.Dungeons[i], res.Dungeons[j]
		if a.LastBossLevel != b.LastBossLevel {
			return a.LastBossLevel < b.LastBossLevel
		}
		return a.Name < b.Name
	})
	return res
}

type fight struct {
	label     string
	boss      bool
	encounter *proto.Encounter
}

// How much harder an elite hits than a normal mob of its level: the median Classic elite hits twice as
// hard up to level 30 and about four times from 42.
func eliteDamage(lvl int32) float64 {
	switch {
	case lvl <= 30:
		return 2
	case lvl >= 42:
		return 4
	}
	return 2 + 2*float64(lvl-30)/12
}

func mob(name string, lvl int32, health, armor, dps, swing float64) *proto.Target {
	st := make([]float64, proto.Stat_StatMana+1)
	st[proto.Stat_StatHealth] = health
	st[proto.Stat_StatArmor] = armor
	return &proto.Target{
		Name: name, Level: lvl, Stats: st, MobType: proto.MobType_MobTypeHumanoid,
		SwingSpeed: swing, DamageSpread: 0.4, MinBaseDamage: dps * swing / 1.2,
	}
}

// Mobs fight until they die; the duration only caps a fight the group cannot finish.
func pullEncounter(targets ...*proto.Target) *proto.Encounter {
	return &proto.Encounter{Duration: 300, UseHealth: true, Targets: targets}
}

// Questing at the group's level: one, two and three normal mobs, and an elite a level up with three
// times the health.
func openWorldFights(lvl int32) []fight {
	health, armor, dps := core.CreatureStatsAt(lvl)
	normal := func(n int) fight {
		ts := make([]*proto.Target, n)
		for i := range ts {
			ts[i] = mob(fmt.Sprintf("Mob %d", i+1), lvl, health, armor, dps, 2)
		}
		label := "1 mob"
		if n > 1 {
			label = fmt.Sprintf("%d mobs", n)
		}
		return fight{label: label, encounter: pullEncounter(ts...)}
	}
	eh, ea, ed := core.CreatureStatsAt(lvl + 1)
	elite := mob("Elite", lvl+1, eh*3, ea, ed*eliteDamage(lvl+1), 2)
	return []fight{normal(1), normal(2), normal(3), {label: "Elite", boss: true, encounter: pullEncounter(elite)}}
}

func dungeonFights(name string) ([]fight, error) {
	d, ok := dungeons[name]
	if !ok {
		return nil, fmt.Errorf("no dungeon named %q", name)
	}
	fs := make([]fight, len(d.Pulls))
	for i, p := range d.Pulls {
		ts := make([]*proto.Target, len(p.Mobs))
		for j, m := range p.Mobs {
			ts[j] = mob(m.Name, m.Level, m.Health, m.Armor, m.Dps, m.Swing)
		}
		fs[i] = fight{label: p.Label, boss: p.Boss, encounter: pullEncounter(ts...)}
	}
	return fs, nil
}

// The sim's default raid boss for three minutes. It hits as hard as it does in a 40-player raid.
func raidFights() []fight {
	boss := googleProto.Clone(core.DefaultTargetProto).(*proto.Target)
	boss.Name = "Raid Boss"
	return []fight{{label: "Raid boss", boss: true, encounter: &proto.Encounter{Duration: 180, DurationVariation: 15, Targets: []*proto.Target{boss}}}}
}

var tankSpecs = []proto.Spec{proto.Spec_SpecProtectionWarrior, proto.Spec_SpecProtectionPaladin, proto.Spec_SpecFeralBearDruid}

// Who holds the mobs when the request names no tank: the first tank spec, else the first player.
func defaultTank(players []*proto.Player) *proto.UnitReference {
	for i, p := range players {
		if spec, err := core.PlayerProtoToSpecSafe(p); err == nil && slices.Contains(tankSpecs, spec) {
			return &proto.UnitReference{Type: proto.UnitReference_Player, Index: int32(i)}
		}
	}
	return &proto.UnitReference{Type: proto.UnitReference_Player, Index: 0}
}

func summarize(u *proto.UnitMetrics, iterations int32) *proto.ActivityPlayerSummary {
	var mana float64
	for _, r := range u.Resources {
		if r.Type == proto.ResourceType_ResourceTypeMana {
			mana -= r.ActualGain
		}
	}
	s := &proto.ActivityPlayerSummary{
		Name:          u.Name,
		Dps:           u.Dps.GetAvg(),
		Hps:           u.Hps.GetAvg(),
		Tps:           u.Threat.GetAvg(),
		Dtps:          u.Dtps.GetAvg(),
		ChanceOfDeath: u.ChanceOfDeath,
		SecondsOom:    u.SecondsOomAvg,
		ManaSpent:     mana / float64(max(iterations, 1)),
	}
	// A pet's damage is its owner's.
	for _, pet := range u.Pets {
		s.Dps += pet.Dps.GetAvg()
		s.Tps += pet.Threat.GetAvg()
	}
	return s
}

func errorResult(err error) *proto.ActivitySimResult {
	return &proto.ActivitySimResult{Error: &proto.ErrorOutcome{Message: err.Error()}}
}

// Each of the activity's fights as its own raid sim, then each player's results over all of them, each
// fight weighted by how long it lasts.
func RunActivitySim(req *proto.ActivitySimRequest) (result *proto.ActivitySimResult) {
	defer func() {
		if r := recover(); r != nil {
			result = errorResult(fmt.Errorf("%v", r))
		}
	}()
	if req.Raid == nil || len(req.Raid.Parties) == 0 {
		return errorResult(fmt.Errorf("the request has no group"))
	}
	players := req.Raid.Parties[0].Players
	if n := len(players); n < 1 || n > 5 {
		return errorResult(fmt.Errorf("a group holds 1 to 5 players, not %d", n))
	}
	lvl := req.Level
	if lvl <= 0 {
		lvl = core.CharacterLevel
	}
	var fights []fight
	switch req.Activity {
	case proto.Activity_ActivityOpenWorld:
		fights = openWorldFights(lvl)
	case proto.Activity_ActivityDungeon:
		var err error
		if fights, err = dungeonFights(req.Dungeon); err != nil {
			return errorResult(err)
		}
	case proto.Activity_ActivityRaid:
		fights = raidFights()
	default:
		return errorResult(fmt.Errorf("unknown activity %v", req.Activity))
	}

	raid := googleProto.Clone(req.Raid).(*proto.Raid)
	raid.Parties = raid.Parties[:1]
	warnings := dropUnknownItems(raid.Parties[0].Players)
	if req.Untanked {
		raid.Tanks = nil
	} else if len(raid.Tanks) == 0 {
		raid.Tanks = []*proto.UnitReference{defaultTank(players)}
	}
	// A player with no rotation at all stands idle rather than leaving the engine without one.
	for _, p := range raid.Parties[0].Players {
		if p.Rotation == nil {
			p.Rotation = &proto.APLRotation{Type: proto.APLRotation_TypeAPL}
		}
	}
	if err := addHealerRotations(raid, fights[0].encounter); err != nil {
		return errorResult(err)
	}
	opts := req.SimOptions
	if opts == nil {
		opts = &proto.SimOptions{}
	}
	if opts.Iterations <= 0 {
		opts = googleProto.Clone(opts).(*proto.SimOptions)
		opts.Iterations = 1000
	}

	res := &proto.ActivitySimResult{Warnings: warnings}
	totals := make([]*proto.ActivityPlayerSummary, len(players))
	for i := range totals {
		totals[i] = &proto.ActivityPlayerSummary{}
	}
	for _, f := range fights {
		r := core.RunRaidSim(&proto.RaidSimRequest{RequestId: req.RequestId, Raid: raid, Encounter: f.encounter, SimOptions: opts, Type: proto.SimType_SimTypeRaid})
		if r.Error != nil {
			r.Error.Message = f.label + ": " + r.Error.Message
			return &proto.ActivitySimResult{Error: r.Error}
		}
		secs := r.AvgIterationDuration
		out := &proto.ActivityFight{Label: f.label, TargetCount: int32(len(f.encounter.Targets)), Boss: f.boss, Seconds: secs}
		for i, u := range r.RaidMetrics.Parties[0].Players {
			s := summarize(u, opts.Iterations)
			out.Players = append(out.Players, s)
			t := totals[i]
			t.Name = s.Name
			t.Dps += s.Dps * secs
			t.Hps += s.Hps * secs
			t.Tps += s.Tps * secs
			t.Dtps += s.Dtps * secs
			t.ChanceOfDeath = max(t.ChanceOfDeath, s.ChanceOfDeath)
			t.SecondsOom += s.SecondsOom
			t.ManaSpent += s.ManaSpent
		}
		res.Fights = append(res.Fights, out)
		res.TotalSeconds += secs
	}
	for _, t := range totals {
		if res.TotalSeconds > 0 {
			t.Dps /= res.TotalSeconds
			t.Hps /= res.TotalSeconds
			t.Tps /= res.TotalSeconds
			t.Dtps /= res.TotalSeconds
		}
		res.GroupDps += t.Dps
	}
	res.Players = totals
	return res
}

// Gives each healer without a rotation one that heals the tank first, then whoever drops below half.
func addHealerRotations(raid *proto.Raid, e *proto.Encounter) error {
	players := raid.Parties[0].Players
	tank := 0
	if len(raid.Tanks) > 0 {
		tank = int(raid.Tanks[0].GetIndex())
	}
	res := core.ComputeStats(&proto.ComputeStatsRequest{Raid: raid, Encounter: e, SkipRotation: true})
	if res.ErrorResult != "" {
		return fmt.Errorf("%s", res.ErrorResult)
	}
	for i, ps := range res.RaidStats.Parties[0].Players[:len(players)] {
		have := map[int32]bool{}
		for _, sp := range ps.GetMetadata().GetSpells() {
			have[sp.Id.GetSpellId()] = true
		}
		if rot := healerRotation(players[i], i, tank, len(players), have); rot != nil {
			players[i].Rotation = rot
		}
	}
	return nil
}

// Empties the slots whose item neither the database nor the rows the player carries hold, so a preset
// that names an item the game files dropped still sims. Each emptied slot is a warning.
func dropUnknownItems(players []*proto.Player) []string {
	var warnings []string
	for _, p := range players {
		carried := map[int32]bool{}
		for _, item := range p.GetDatabase().GetItems() {
			carried[item.GetId()] = true
		}
		for i, item := range p.GetEquipment().GetItems() {
			if id := item.GetId(); id != 0 && !carried[id] && core.GetItemByID(id) == nil {
				warnings = append(warnings, fmt.Sprintf("%s: no item %d in the database, slot left empty", p.Name, id))
				p.Equipment.Items[i] = &proto.ItemSpec{}
			}
		}
	}
	return warnings
}
