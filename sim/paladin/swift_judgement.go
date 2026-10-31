package paladin

import (
	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/spelldata"
)

// Swift Judgement (talent)
// https://www.wowhead.com/forever/spell=1310994
//
// Unleash the energy of a Seal spell upon an enemy. Does not consume the Seal. Refer to individual
// Seals for Judgement effect. Does not share a cooldown with your Judgement ability.
func (paladin *Paladin) registerSwiftJudgement() {
	rank := spellData.SwiftJudgement.Highest()

	config := spelldata.SpellConfig(&paladin.Unit, rank,
		spelldata.Flags(core.SpellFlagAPL|core.SpellFlagPassiveSpell|core.SpellFlagNoOnCastComplete))
	config.ProcMask = core.ProcMaskEmpty
	config.ClassSpellMask = SpellMaskSwiftJudgement
	config.DamageMultiplier = 1
	config.ThreatMultiplier = 1
	// The row states no cost and no global cooldown, so the cast is left empty.
	config.Cast.DefaultCast.NonEmpty = true

	config.ExtraCastCondition = func(_ *core.Simulation, _ *core.Unit) bool {
		return paladin.activeSeal() != nil
	}
	config.ApplyEffects = func(sim *core.Simulation, target *core.Unit, _ *core.Spell) {
		paladin.activeSeal().judgement.Cast(sim, target)
	}
	paladin.RegisterSpell(config)
}
