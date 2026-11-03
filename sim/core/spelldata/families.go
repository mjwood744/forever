package spelldata

import "github.com/wowsims/forever/sim/core"

// Ranks of one spell share a name and class flags in the game files; the rank's spell level orders
// them. Core reads this to move a rotation to the ranks a character below 60 has.
func init() {
	core.RegisterSpellFamilies(func(spellID int32) (core.SpellFamily, bool) {
		s := Find(spellID)
		if s == nil || s.Name == "" {
			return core.SpellFamily{}, false
		}
		return core.SpellFamily{Name: s.Name, Flags: s.ClassFlags, Level: int32(s.SpellLevel)}, true
	})
}
