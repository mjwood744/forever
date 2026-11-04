package trios

import (
	_ "embed"
	"encoding/json"
	"sort"
	"strconv"
	"strings"

	"github.com/wowsims/forever/sim/core/spelldata"
)

// Each trio character's FMS macro keys by level, as the level sims press them. The sims read the same file
// in their tests; the site reads it to show a player their keys.
//
//go:embed testdata/keys.json
var keysJSON []byte

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

// The 6-level bracket a level's keys come from: the last one reached. Gear and talents are per level.
// The highest level at or below lvl that the character's keys are written for: every 6 levels, plus
// any level a respec adds keys at.
func bracket(ck charKeys, lvl int32) string {
	best := int32(6)
	for l := range ck.Levels {
		if n, err := strconv.Atoi(l); err == nil && int32(n) <= lvl && int32(n) > best {
			best = int32(n)
		}
	}
	return strconv.Itoa(int(best))
}

// When a key is pressed, in words. It says in words what presses() in levels_test.go has the level sims
// do, in the order the sims check them; a change there is made here too.
type keyPress struct {
	key  string
	when string
}

var pressWords = map[string][]keyPress{
	"Rellyk": {{"Shift+3", "before the pull, and again when your seal runs out"}, {"Shift+2", "your health is under 30%"},
		{"2", "2 or more mobs are on you"}, {"1", ""}},
	"Skarr":  {{"Shift+2", "your health is under 30%"}, {"2", "2 or more mobs are on you"}, {"1", ""}},
	"Shryke": {{"3", "on a boss"}, {"5", "2 or more mobs are on you"}, {"2", "you have 4 or more combo points"}, {"1", ""}},
	// Nyxx as the duo plays her: heals and the wand.
	"Nyxx": {{"Shift+2", "the tank is under 35%"}, {"Shift+6", "the tank is under 90% and has no shield"},
		{"Shift+1", "the tank is under 80%"}, {"Shift+3", "someone else is under 60%"}, {"Shift+4", "you are under 50%"}, {"2", ""}},
	"Wyther": {{"Shift+6", "before the pull, and when no totem is down"}, {"Shift+2", "the tank is under 35%"},
		{"Shift+1", "the tank is under 80%"}, {"Shift+3", "someone else is under 60%"}, {"Shift+4", "you are under 50%"},
		{"1", "your mana is above 50%"}},
	"Wyldd": {{"Shift+2", "the tank is under 35%"}, {"Shift+6", "your mana is under 25%"}, {"Shift+1", "the tank is under 80%"},
		{"Shift+3", "someone else is under 60%"}, {"Shift+4", "you are under 50%"}, {"1", "your mana is above 60%"}},
	"Ashyn": {{"Shift+6", "your mana is under 15%"}, {"2", "3 or more mobs are on you"}, {"1", "your mana is above 8%"}, {"5", ""}},
	"Malyce": {{"Shift+6", "your mana is under 35% and your health is above 60%"}, {"4", "your curse is not on the target"},
		{"2", "3 or more mobs are on you"}, {"3", "on a boss"}, {"1", "your mana is above 8%"}, {"5", ""}},
	"Hawke": {{"Shift+1", "your pet is under 80% and has no Mend Pet"}, {"3", "on a boss"}, {"Shift+3", "it is off cooldown"},
		{"2", "3 or more mobs are on you"}, {"1", ""}},
}

// One key on a player's bars and when to press it.
type KeyUse struct {
	Key string `json:"key"`
	// When to press it, in words; empty for the key pressed the rest of the time.
	When string `json:"when"`
	// The key pressed the rest of the time.
	Main bool `json:"main"`
	// A key the sims never press: one for when something happens, like a taunt or an interrupt.
	Extra bool `json:"extra"`
	// What the key casts, in the order its macro tries them.
	Spells []string `json:"spells"`
}

type DuoKeys struct {
	// The trio character whose keys these are: every character of the class uses them.
	From  string   `json:"from"`
	Level int32    `json:"level"`
	Keys  []KeyUse `json:"keys"`
}

var parsedKeys map[string]charKeys

func allKeys() map[string]charKeys {
	if parsedKeys == nil {
		if err := json.Unmarshal(keysJSON, &parsedKeys); err != nil {
			panic(err)
		}
	}
	return parsedKeys
}

// The keys a character of a class (its name in the game, "Warrior") has at a level, from the trio character
// of that class, pressed keys first in the order the sims check them.
func KeysFor(class string, level int32) (*DuoKeys, bool) {
	keys := allKeys()
	names := make([]string, 0, len(keys))
	for name := range keys {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		ck := keys[name]
		if !strings.EqualFold(ck.Class, class) {
			continue
		}
		at := ck.Levels[bracket(ck, level)].Keys
		out := &DuoKeys{From: name, Level: level}
		pressed := map[string]bool{}
		for _, p := range pressWords[name] {
			items, ok := at[p.key]
			if !ok || pressed[p.key] {
				continue
			}
			pressed[p.key] = true
			out.Keys = append(out.Keys, KeyUse{Key: p.key, When: p.when, Main: p.when == "", Spells: spellNames(items, level)})
		}
		extra := make([]string, 0, len(at))
		for key := range at {
			if !pressed[key] {
				extra = append(extra, key)
			}
		}
		sort.Strings(extra)
		for _, key := range extra {
			out.Keys = append(out.Keys, KeyUse{Key: key, Extra: true, Spells: spellNames(at[key], level)})
		}
		return out, true
	}
	return nil, false
}

// The spells a key's macro lines cast, by name, each once, in order. A line names every rank, highest first.
func spellNames(items []keyItem, level int32) []string {
	var out []string
	seen := map[string]bool{}
	for _, it := range items {
		for _, step := range it.Spells {
			for _, id := range step {
				s := spelldata.Find(id)
				if s == nil || s.Name == "" || int32(s.SpellLevel) > level {
					continue
				}
				if !seen[s.Name] {
					seen[s.Name] = true
					out = append(out, s.Name)
				}
				break
			}
		}
	}
	return out
}
