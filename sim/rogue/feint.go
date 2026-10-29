package rogue

import (
	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/spelldata"
)

// This exists purely to model the DPS loss of having to occasionally feint
func (rogue *Rogue) registerFeintSpell() {
	feintRank := spellData.Feint.Known(rogue.Level)
	config := spelldata.SpellConfig(&rogue.Unit, feintRank, spelldata.Melee(core.ProcMaskMeleeMHSpecial))

	config.ApplyEffects = func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
		rogue.BreakStealth(sim)
		result := spell.CalcAndDealDamage(sim, target, 0, spell.OutcomeMeleeSpecialNoBlockDodgeParryNoCrit)

		if !result.Landed() {
			spell.IssueRefund(sim)
		}
	}

	rogue.RegisterSpell(config)
}
