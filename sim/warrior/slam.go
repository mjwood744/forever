package warrior

import (
	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/spelldata"
)

func (warrior *Warrior) registerSlam() {
	slamRank := spellData.Slam.Known(warrior.Level)
	improvedSlamRank := slamRank.OverriddenBy(spellData.ImprovedSlam.Highest())
	rank := core.Ternary(warrior.Talents.ImprovedSlam > 0, improvedSlamRank, slamRank)

	config := spelldata.SpellConfig(&warrior.Unit, rank, spelldata.Melee(core.ProcMaskMeleeMHSpecial))
	// The APL names Slam by the spell the talent replaces.
	config.ActionID = core.ActionID{SpellID: slamRank.ID}

	config.ApplyEffects = func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
		baseDamage := rank.WeaponStrike(sim, spell, spell.Unit.MHWeaponDamage(sim, spell.MeleeAttackPower(target)), 0)
		result := spell.CalcAndDealDamage(sim, target, baseDamage, spell.OutcomeMeleeWeaponSpecialHitAndCrit)

		if !result.Landed() {
			spell.IssueRefund(sim)
		}
	}

	warrior.RegisterSpell(config)
}
