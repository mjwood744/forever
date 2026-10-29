package shaman

import (
	"github.com/wowsims/forever/sim/core/spelldata"
)

// In game a rank deals less damage once the caster is far enough past the level it is taught at; a rank
// taught within rankLevelWindow levels of the caster's own is clear of that (44 or later at level 60).
const rankLevelWindow = 16 // TODO: Ingame test needed: the cutoff is believed to be 42 or 43 at level 60.

// The ranks of a damage spell the shaman registers: those known at the level and taught within
// rankLevelWindow of it, and always the highest known rank.
func castableRanks(ladder spelldata.Ladder, level int32) spelldata.Ladder {
	known := ladder.Known(level)
	var ids []int32
	ladder.Each(func(_ int32, rank *spelldata.Spell) {
		if rank == known || (int32(rank.SpellLevel) >= level-rankLevelWindow && int32(rank.SpellLevel) <= level) {
			ids = append(ids, rank.ID)
		}
	})
	return spelldata.Ranked(ids...)
}
