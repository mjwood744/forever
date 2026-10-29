package trios

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/wowsims/forever/sim"
	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/proto"
	"github.com/wowsims/forever/sim/core/spelldata"
	"google.golang.org/protobuf/encoding/protojson"
)

// Each trio at levels 6, 12 ... 60 in quest gear, pressing the FMS macro keys of that level.
// testdata: quest_gear.json and quest_gear_items.json (best Horde quest reward per slot),
// talents.json (each character's points at the level) and keys.json (each key's casts at the level,
// every rank known at the level, highest first).

type keyItem struct {
	Kind   string          `json:"kind"`
	Spells [][]int32       `json:"spells"`
	Target json.RawMessage `json:"target"`
	Cond   *string         `json:"cond"`
}

type charKeys struct {
	Class  string `json:"class"`
	Trio   int    `json:"trio"`
	Levels map[string]struct {
		Keys    map[string][]keyItem `json:"keys"`
		Missing []string             `json:"missing"`
	} `json:"levels"`
}

var (
	questGear  map[string]map[string][]int32
	talentsAt  map[string]map[string]string
	keysAt     map[string]charKeys
	gearItems  *proto.SimDatabase
	keyIDs     = map[int32]bool{} // every spell id a key names at any level
	levelsList = []int32{6, 12, 18, 24, 30, 36, 42, 48, 54, 60}
)

func load(name string, v any) {
	data, err := os.ReadFile("testdata/" + name)
	if err != nil {
		panic(err)
	}
	if err := json.Unmarshal(data, v); err != nil {
		panic(err)
	}
}

func init() {
	sim.RegisterAll()
	load("quest_gear.json", &questGear)
	load("talents.json", &talentsAt)
	// TALENTS=other.json swaps in another build for the characters that file lists.
	if f := os.Getenv("TALENTS"); f != "" {
		var other map[string]map[string]string
		load(f, &other)
		for name, at := range other {
			talentsAt[name] = at
		}
	}
	load("keys.json", &keysAt)
	for _, ck := range keysAt {
		for _, lv := range ck.Levels {
			for _, items := range lv.Keys {
				for _, it := range items {
					for _, step := range it.Spells {
						for _, id := range step {
							keyIDs[id] = true
						}
					}
				}
			}
		}
	}
	data, err := os.ReadFile("testdata/quest_gear_items.json")
	if err != nil {
		panic(err)
	}
	gearItems = &proto.SimDatabase{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(data, gearItems); err != nil {
		panic(err)
	}
}

// ---- APL JSON ----

func js(v any) string { b, _ := json.Marshal(v); return string(b) }

func unitSelf() string   { return `{"type":"Self"}` }
func unitTarget() string { return `{"type":"Target"}` }
func unitPlayer(i int) string {
	return fmt.Sprintf(`{"type":"Player","index":%d}`, i)
}
func unitPet(owner int) string {
	return fmt.Sprintf(`{"type":"Pet","index":0,"owner":{"type":"Player","index":%d}}`, owner)
}

func cmp(op, lhs, val string) string {
	return fmt.Sprintf(`{"cmp":{"op":"%s","lhs":%s,"rhs":{"const":{"val":"%s"}}}}`, op, lhs, val)
}
func hpBelow(unit string, p int) string {
	return cmp("OpLe", `{"currentHealthPercent":{"sourceUnit":`+unit+`}}`, strconv.Itoa(p)+"%")
}
func hpAbove(unit string, p int) string {
	return cmp("OpGe", `{"currentHealthPercent":{"sourceUnit":`+unit+`}}`, strconv.Itoa(p)+"%")
}
func manaBelow(p int) string { return cmp("OpLe", `{"currentManaPercent":{}}`, strconv.Itoa(p)+"%") }
func manaAbove(p int) string { return cmp("OpGe", `{"currentManaPercent":{}}`, strconv.Itoa(p)+"%") }
func targetsAtLeast(n int) string {
	return cmp("OpGe", `{"numberTargets":{}}`, strconv.Itoa(n))
}
func comboAtLeast(n int) string { return cmp("OpGe", `{"currentComboPoints":{}}`, strconv.Itoa(n)) }
func aura(unit string, id int32) string {
	return fmt.Sprintf(`{"auraIsActive":{"sourceUnit":%s,"auraId":{"spellId":%d}}}`, unit, id)
}
func not(v string) string { return `{"not":{"val":` + v + `}}` }
func and(vs ...string) string {
	var keep []string
	for _, v := range vs {
		if v != "" {
			keep = append(keep, v)
		}
	}
	switch len(keep) {
	case 0:
		return ""
	case 1:
		return keep[0]
	}
	return `{"and":{"vals":[` + strings.Join(keep, ",") + `]}}`
}

func castJSON(id int32, target string) string {
	if target == "" {
		return fmt.Sprintf(`{"castSpell":{"spellId":{"spellId":%d}}}`, id)
	}
	return fmt.Sprintf(`{"castSpell":{"spellId":{"spellId":%d},"target":%s}}`, id, target)
}

func listItem(action, cond string) string {
	if cond == "" {
		return `{"action":` + action + `}`
	}
	return `{"action":` + strings.Replace(action, "{", `{"condition":`+cond+`,`, 1) + `}`
}

// ---- the trios ----

type member struct {
	name   string
	player *proto.Player
}

type trio struct {
	n       int
	members []member // party order
	tank    *proto.UnitReference
	tankRef string // the tank as an APL unit
	petRef  string // Hawke's pet, "" without one
}

// The 6-level bracket a level's keys come from: the last one reached. Gear and talents are per level.
func bracket(lvl int32) string { return strconv.Itoa(int(max(6, lvl/6*6))) }

func baseConsumes() *proto.ConsumesSpec { return &proto.ConsumesSpec{} }

func equip(name string, lvl int32) *proto.EquipmentSpec {
	ids := questGear[name][strconv.Itoa(int(lvl))]
	items := make([]*proto.ItemSpec, len(ids))
	for i, id := range ids {
		items[i] = &proto.ItemSpec{Id: id}
	}
	return &proto.EquipmentSpec{Items: items}
}

func newPlayer(name string, lvl int32, class proto.Class, race proto.Race, spec any) *proto.Player {
	p := &proto.Player{
		Name: name, Class: class, Race: race, Level: lvl,
		Equipment:      equip(name, lvl),
		Database:       gearItems,
		Consumables:    baseConsumes(),
		Buffs:          &proto.IndividualBuffs{},
		TalentsString:  talentsAt[strings.ToLower(name)][strconv.Itoa(int(lvl))],
		ReactionTimeMs: 100, ChannelClipDelayMs: 50,
		DistanceFromTarget: 20,
	}
	return core.WithSpec(p, spec)
}

func makeTrio(n int, lvl int32) trio {
	switch n {
	case 1:
		var armor proto.PriestOptions_Armor
		if lvl >= 12 {
			armor = proto.PriestOptions_InnerFire
		}
		nyxx := newPlayer("Nyxx", lvl, proto.Class_ClassPriest, proto.Race_RaceUndead,
			&proto.Player_HealerPriest{HealerPriest: &proto.HealerPriest{Options: &proto.HealerPriest_Options{ClassOptions: &proto.PriestOptions{Armor: armor}}}})
		rel := newPlayer("Rellyk", lvl, proto.Class_ClassPaladin, proto.Race_RaceUndead,
			&proto.Player_ProtectionPaladin{ProtectionPaladin: &proto.ProtectionPaladin{Options: &proto.ProtectionPaladin_Options{ClassOptions: &proto.PaladinOptions{}}}})
		rel.InFrontOfTarget, rel.DistanceFromTarget = true, 5
		ash := newPlayer("Ashyn", lvl, proto.Class_ClassMage, proto.Race_RaceUndead,
			&proto.Player_Mage{Mage: &proto.Mage{Options: &proto.Mage_Options{ClassOptions: &proto.MageOptions{DefaultMageArmor: proto.MageArmor(1)}}}})
		return trio{1, []member{{"Nyxx", nyxx}, {"Rellyk", rel}, {"Ashyn", ash}},
			&proto.UnitReference{Type: proto.UnitReference_Player, Index: 1}, unitPlayer(1), ""}
	case 2:
		wy := newPlayer("Wyther", lvl, proto.Class_ClassShaman, proto.Race_RaceTroll,
			&proto.Player_RestorationShaman{RestorationShaman: &proto.RestorationShaman{Options: &proto.RestorationShaman_Options{ClassOptions: &proto.ShamanOptions{}}}})
		stance := proto.WarriorStance_WarriorStanceDefensive
		if lvl < 10 {
			stance = proto.WarriorStance_WarriorStanceBattle
		}
		sk := newPlayer("Skarr", lvl, proto.Class_ClassWarrior, proto.Race_RaceOrc,
			&proto.Player_ProtectionWarrior{ProtectionWarrior: &proto.ProtectionWarrior{Options: &proto.ProtectionWarrior_Options{ClassOptions: &proto.WarriorOptions{
				QueueDelay: 250, UseBattleShout: true, StanceSnapshot: true, DefaultStance: stance}}}})
		sk.InFrontOfTarget, sk.DistanceFromTarget = true, 5
		summon, armor := proto.WarlockOptions_Imp, proto.WarlockOptions_NoArmor
		if lvl >= 20 {
			summon, armor = proto.WarlockOptions_Succubus, proto.WarlockOptions_DemonArmor
		}
		ma := newPlayer("Malyce", lvl, proto.Class_ClassWarlock, proto.Race_RaceTroll,
			&proto.Player_Warlock{Warlock: &proto.Warlock{Options: &proto.Warlock_Options{ClassOptions: &proto.WarlockOptions{Summon: summon, Armor: armor}}}})
		return trio{2, []member{{"Wyther", wy}, {"Skarr", sk}, {"Malyce", ma}},
			&proto.UnitReference{Type: proto.UnitReference_Player, Index: 1}, unitPlayer(1), ""}
	}
	pet := proto.HunterOptions_PetNone
	if lvl >= 10 {
		pet = proto.HunterOptions_Turtle
	}
	ho := &proto.HunterOptions{PetType: pet, PetUptime: 1}
	if lvl >= 10 {
		// The Turtle's training as far as the level allows: Great Stamina, Natural Armor, Hardiness.
		ho.GreatStamina, ho.NaturalArmor, ho.PetHardiness = min(10, (lvl-6)/6*2), min(10, (lvl-6)/6*2), min(5, (lvl-6)/12)
	}
	hk := newPlayer("Hawke", lvl, proto.Class_ClassHunter, proto.Race_RaceWindshaperSkyborne,
		&proto.Player_Hunter{Hunter: &proto.Hunter{Options: &proto.Hunter_Options{ClassOptions: ho}}})
	wd := newPlayer("Wyldd", lvl, proto.Class_ClassDruid, proto.Race_RaceWindshaperSkyborne,
		&proto.Player_RestorationDruid{RestorationDruid: &proto.RestorationDruid{Options: &proto.RestorationDruid_Options{ClassOptions: &proto.DruidOptions{}}}})
	sh := newPlayer("Shryke", lvl, proto.Class_ClassRogue, proto.Race_RaceWindshaperSkyborne,
		&proto.Player_Rogue{Rogue: &proto.Rogue{Options: &proto.Rogue_Options{ClassOptions: &proto.RogueOptions{}}}})
	sh.DistanceFromTarget = 5
	t := trio{3, []member{{"Hawke", hk}, {"Wyldd", wd}, {"Shryke", sh}}, nil, "", ""}
	if lvl >= 10 {
		t.tank = &proto.UnitReference{Type: proto.UnitReference_Pet, Index: 0, Owner: &proto.UnitReference{Type: proto.UnitReference_Player, Index: 0}}
		t.tankRef, t.petRef = unitPet(0), unitPet(0)
	} else {
		sh.InFrontOfTarget = true
		t.tank = &proto.UnitReference{Type: proto.UnitReference_Player, Index: 2}
		t.tankRef = unitPlayer(2)
	}
	return t
}

func (t trio) raid() *proto.Raid {
	ps := make([]*proto.Player, len(t.members))
	for i, m := range t.members {
		ps[i] = m.player
	}
	return &proto.Raid{Parties: []*proto.Party{{Players: ps, Buffs: &proto.PartyBuffs{}}}, Buffs: &proto.RaidBuffs{}, Debuffs: &proto.Debuffs{},
		Tanks: []*proto.UnitReference{t.tank}}
}

// ---- the fights ----

type fight struct {
	name    string
	count   int
	levelUp int32
	health  float64 // a multiple of a normal mob's health
	damage  float64 // a multiple of an elite's damage over a normal mob's; 0 for a normal mob
	boss    bool
}

var fights = []fight{
	{"1 mob", 1, 0, 1, 0, false},
	{"3 mobs", 3, 0, 1, 0, false},
	{"Elite", 1, 1, 3, 1, true},
	{"Dungeon boss", 1, 2, 6, 1.5, true},
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

func encounter(f fight, lvl int32) *proto.Encounter {
	mobLvl := lvl + f.levelUp
	health, armor, dps := core.CreatureStatsAt(mobLvl)
	if f.damage > 0 {
		dps *= f.damage * eliteDamage(mobLvl)
	}
	ts := make([]*proto.Target, f.count)
	for i := range ts {
		st := make([]float64, proto.Stat_StatMana+1)
		st[proto.Stat_StatHealth] = health * f.health
		st[proto.Stat_StatArmor] = armor
		ts[i] = &proto.Target{
			Level: mobLvl, Stats: st, MobType: proto.MobType_MobTypeHumanoid,
			SwingSpeed: 2, DamageSpread: 0.4, MinBaseDamage: dps * 2 / 1.2,
		}
	}
	return &proto.Encounter{Duration: 300, UseHealth: true, Targets: ts}
}

// ---- keys to an APL ----

type press struct {
	key      string
	cond     string
	prepull  bool
	bossOnly bool
}

// What each character presses, top first: the keys from the macro sheets in the order a player reaches
// for them. Keys left out are situational (interrupts, stuns, taunts, crowd control).
func presses(name string, t trio) []press {
	self := unitSelf()
	switch name {
	case "Rellyk":
		return []press{{key: "Shift+3", prepull: true}, {key: "Shift+3", cond: "SEAL"}, {key: "Shift+2", cond: hpBelow(self, 30)},
			{key: "2", cond: targetsAtLeast(2)}, {key: "1"}}
	case "Skarr":
		return []press{{key: "Shift+2", cond: hpBelow(self, 30)}, {key: "2", cond: targetsAtLeast(2)}, {key: "1"}}
	case "Shryke":
		return []press{{key: "3", bossOnly: true}, {key: "5", cond: targetsAtLeast(2)}, {key: "2", cond: comboAtLeast(4)}, {key: "1"}}
	case "Nyxx", "Wyther", "Wyldd":
		other := unitPlayer(2) // Ashyn, Malyce and Shryke all sit third
		mana := 50
		if name == "Wyldd" {
			mana = 60
		}
		ps := []press{
			{key: "Shift+2", cond: hpBelow(t.tankRef, 35)},
			{key: "Shift+6", cond: "SHIFT6"},
			{key: "Shift+1", cond: hpBelow(t.tankRef, 80)},
			{key: "Shift+3", cond: hpBelow(other, 60)},
			{key: "Shift+4", cond: hpBelow(self, 50)},
			{key: "1", cond: manaAbove(mana)},
		}
		if name == "Nyxx" {
			ps = append(ps, press{key: "2"})
		}
		if name == "Wyther" {
			ps = append([]press{{key: "Shift+6", prepull: true}}, ps...)
		}
		return ps
	case "Ashyn":
		return []press{{key: "Shift+6", cond: manaBelow(15)}, {key: "2", cond: targetsAtLeast(3)}, {key: "1", cond: manaAbove(8)}, {key: "5"}}
	case "Malyce":
		return []press{{key: "Shift+6", cond: and(manaBelow(35), hpAbove(self, 60))}, {key: "4", cond: "CURSE"},
			{key: "2", cond: targetsAtLeast(3)}, {key: "3", bossOnly: true}, {key: "1", cond: manaAbove(8)}, {key: "5"}}
	case "Hawke":
		ps := []press{{key: "3", bossOnly: true}, {key: "Shift+3"}, {key: "2", cond: targetsAtLeast(3)}, {key: "1"}}
		if t.petRef != "" {
			ps = append([]press{{key: "Shift+1", cond: and(hpBelow(t.petRef, 80), not(fmt.Sprintf(`{"dotIsActive":{"spellId":{"spellId":MENDPET},"targetUnit":%s}}`, t.petRef)))}}, ps...)
		}
		return ps
	}
	return nil
}

func targetJSON(raw json.RawMessage) string {
	if len(raw) == 0 || string(raw) == "null" {
		return ""
	}
	return string(raw)
}

// The highest known rank the character has registered. A macro names the spell, not the rank, so between
// brackets a rank learned after the key's level is the one pressed: the highest registered rank of the same
// name that a key names at some level or that costs mana, which leaves out a spell's triggered rows of the same
// name.
func pick(ids []int32, have map[int32]bool) int32 {
	for _, id := range ids {
		if have[id] {
			return id
		}
	}
	if len(ids) == 0 {
		return 0
	}
	var best *spelldata.Spell
	name := spelldata.Find(ids[0]).Name
	for id := range have {
		if s := spelldata.Find(id); (keyIDs[id] || s.Cost() > 0) && s.Name == name && (best == nil || s.SpellLevel > best.SpellLevel) {
			best = s
		}
	}
	if best == nil {
		return 0
	}
	return best.ID
}

// The APL for a character from its presses; unknown spells go to dropped.
func buildAPL(name string, t trio, lvl int32, f fight, have map[int32]bool, dropped map[string]bool) string {
	ck := keysAt[name]
	keys := ck.Levels[bracket(lvl)].Keys
	var prepull, prio []string
	seq := 0
	for _, p := range presses(name, t) {
		items, ok := keys[p.key]
		if !ok || (p.bossOnly && !f.boss) {
			continue
		}
		for _, it := range items {
			target := targetJSON(it.Target)
			cond := p.cond
			// Conditions that read the key's own spells.
			var ids []int32
			for _, step := range it.Spells {
				if id := pick(step, have); id != 0 {
					ids = append(ids, id)
				} else if len(step) > 0 {
					dropped[fmt.Sprintf("%s %s: spell %d", name, p.key, step[0])] = true
				}
			}
			if len(ids) == 0 {
				continue
			}
			switch cond {
			case "SEAL", "CURSE", "SHIFT6":
				unit := self(cond, t)
				var none []string
				for _, id := range ids {
					none = append(none, not(aura(unit, id)))
				}
				cond = and(none...)
				if p.cond == "SHIFT6" {
					cond = shift6(name, t, ids, cond)
				}
				if p.cond == "SEAL" {
					// the steady seal, once Judgement has used the opening one
					ids = ids[len(ids)-1:]
				}
			}
			if strings.Contains(cond, "MENDPET") {
				cond = strings.ReplaceAll(cond, "MENDPET", strconv.Itoa(int(ids[0])))
			}
			if it.Cond != nil && *it.Cond == "stealth" {
				cond = and(cond, `{"auraIsActive":{"auraId":{"spellId":STEALTH}}}`)
			}
			if p.prepull {
				for i, id := range ids {
					prepull = append(prepull, fmt.Sprintf(`{"action":%s,"doAtValue":{"const":{"val":"-%ds"}}}`, castJSON(id, target), len(ids)+1-i))
				}
				continue
			}
			if it.Kind == "cast" || len(ids) == 1 {
				prio = append(prio, listItem(castJSON(ids[0], target), cond))
				continue
			}
			seq++
			sname := fmt.Sprintf("s%d", seq)
			steps := make([]string, len(ids))
			for i, id := range ids {
				steps[i] = castJSON(id, target)
			}
			prio = append(prio,
				listItem(fmt.Sprintf(`{"resetSequence":{"sequenceName":"%s"}}`, sname), fmt.Sprintf(`{"sequenceIsComplete":{"sequenceName":"%s"}}`, sname)),
				listItem(fmt.Sprintf(`{"sequence":{"name":"%s","actions":[%s]}}`, sname, strings.Join(steps, ",")), cond))
		}
	}
	if name == "Shryke" {
		if id := stealthID(have); id != 0 {
			prepull = append(prepull, fmt.Sprintf(`{"action":%s,"doAtValue":{"const":{"val":"-1s"}}}`, castJSON(id, "")))
			for i := range prio {
				prio[i] = strings.ReplaceAll(prio[i], "STEALTH", strconv.Itoa(int(id)))
			}
		}
	}
	return `{"type":"TypeAPL","prepullActions":[` + strings.Join(prepull, ",") + `],"priorityList":[` + strings.Join(prio, ",") + `]}`
}

// Whose auras a "none of these is up" key reads: a seal and a mana key on the presser, a curse on the target,
// a shield on the tank.
func self(cond string, t trio) string {
	switch cond {
	case "CURSE":
		return unitTarget()
	}
	return unitSelf()
}

// Shift+6: Nyxx's shield on the tank when it is low and unshielded, Wyther's totems when none is down, Wyldd's
// Innervate when low on mana.
func shift6(name string, t trio, ids []int32, none string) string {
	switch name {
	case "Nyxx":
		return and(hpBelow(t.tankRef, 90), not(aura(t.tankRef, ids[0])))
	case "Wyldd":
		return manaBelow(25)
	}
	return none
}

var stealthIDs = []int32{1787, 1786, 1785, 1784}

func stealthID(have map[int32]bool) int32 { return pick(stealthIDs, have) }

// The spell ids each character has registered.
func registered(r *proto.Raid, e *proto.Encounter) []map[int32]bool {
	res := core.ComputeStats(&proto.ComputeStatsRequest{Raid: r, Encounter: e, SkipRotation: true})
	if res.ErrorResult != "" {
		panic(res.ErrorResult)
	}
	var out []map[int32]bool
	for _, p := range res.RaidStats.Parties[0].Players {
		have := map[int32]bool{}
		for _, s := range p.GetMetadata().GetSpells() {
			have[s.Id.GetSpellId()] = true
		}
		out = append(out, have)
	}
	return out
}

func runLevel(t trio, lvl int32, f fight, iters int32) (*proto.RaidSimResult, map[string]bool) {
	e := encounter(f, lvl)
	have := registered(t.raid(), e)
	dropped := map[string]bool{}
	for i, m := range t.members {
		m.player.Rotation = core.APLRotationFromJsonString(buildAPL(m.name, t, lvl, f, have[i], dropped))
	}
	res := core.RunRaidSim(&proto.RaidSimRequest{Raid: t.raid(), Encounter: e, SimOptions: &proto.SimOptions{Iterations: iters, RandomSeed: 61}})
	if res.Error != nil {
		panic(res.Error.Message)
	}
	return res, dropped
}

func iterations() int32 {
	if n, err := strconv.Atoi(os.Getenv("ITERS")); err == nil {
		return int32(n)
	}
	return 400
}

func TestTrioLevels(t *testing.T) {
	trios := []int{1, 2, 3}
	if v := os.Getenv("TRIO"); v != "" {
		n, _ := strconv.Atoi(v)
		trios = []int{n}
	}
	for _, n := range trios {
		allDropped := map[string]bool{}
		for _, lvl := range levelsList {
			for _, f := range fights {
				tr := makeTrio(n, lvl)
				res, dropped := runLevel(tr, lvl, f, iterations())
				for k := range dropped {
					allDropped[fmt.Sprintf("L%d %s", lvl, k)] = true
				}
				ps := res.RaidMetrics.Parties[0].Players
				var b strings.Builder
				fmt.Fprintf(&b, "LV|T%d|L%d|%s|kill=%.1fs|group=%.1f", n, lvl, f.name, res.AvgIterationDuration, res.RaidMetrics.Dps.Avg)
				for i, m := range tr.members {
					p := ps[i]
					fmt.Fprintf(&b, "|%s dps=%.1f hps=%.1f oom=%.1f death=%.3f dtps=%.1f", m.name, p.Dps.Avg, p.Hps.Avg, p.SecondsOomAvg, p.ChanceOfDeath, p.Dtps.Avg)
					for _, pet := range p.Pets {
						fmt.Fprintf(&b, " pet(dps=%.1f death=%.3f dtps=%.1f)", pet.Dps.Avg, pet.ChanceOfDeath, pet.Dtps.Avg)
					}
				}
				fmt.Println(b.String())
			}
		}
		var ks []string
		for k := range allDropped {
			ks = append(ks, k)
		}
		sort.Strings(ks)
		for _, k := range ks {
			fmt.Printf("DROPPED|T%d|%s\n", n, k)
		}
	}
}
