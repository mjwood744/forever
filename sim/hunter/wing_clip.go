package hunter

import (
	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/spelldata"
)

// Effect 1 is the snare, which a boss ignores; effect 2 is the flat hit. As a main-hand special it can
// proc Windfury Totem and other on-hit effects.

func (hunter *Hunter) registerWingClip() {
	wingClipRank := spellData.WingClip.Known(hunter.Level)
	wingClipHit := wingClipRank.EffectN(2)
	config := spelldata.SpellConfig(&hunter.Unit, wingClipRank, spelldata.Melee(core.ProcMaskMeleeMHSpecial))

	config.ApplyEffects = func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
		spell.CalcAndDealDamage(sim, target, wingClipHit.Roll(sim), spell.OutcomeMeleeSpecialHitAndCrit)
	}

	hunter.RegisterSpell(config)
}
