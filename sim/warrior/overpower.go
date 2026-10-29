package warrior

import (
	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/spelldata"
)

// The window a dodge opens: the aura Offensive State (DND) fires on the hit.
var overpowerWindow = spellData.OffensiveStateTriggered.Highest()

func (warrior *Warrior) registerOverpower() {
	overpowerRank := spellData.Overpower.KnownUpToID(warrior.Level, 11585)
	warrior.OverpowerAura = warrior.RegisterAura(spelldata.AuraConfig(overpowerWindow, spelldata.Label("Overpower Aura")))

	trigger := spelldata.ProcTrigger(&warrior.Character, spellData.OffensiveState.Highest(),
		func(sim *core.Simulation, spell *core.Spell, result *core.SpellResult) {
			warrior.OverpowerAura.Activate(sim)
		})
	trigger.Outcome = core.OutcomeDodge
	trigger.RequireDamageDealt = false
	trigger.TriggerImmediately = true
	warrior.MakeProcTriggerAura(trigger)

	config := spelldata.SpellConfig(&warrior.Unit, overpowerRank, spelldata.Melee(core.ProcMaskMeleeMHSpecial))

	config.ExtraCastCondition = func(sim *core.Simulation, target *core.Unit) bool {
		return warrior.OverpowerAura.IsActive()
	}

	config.ApplyEffects = func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
		baseDamage := overpowerRank.WeaponStrike(sim, spell, spell.Unit.MHNormalizedWeaponDamage(sim, spell.MeleeAttackPower(target)), 0)
		result := spell.CalcAndDealDamage(sim, target, baseDamage, spell.OutcomeMeleeSpecialNoBlockDodgeParry)
		warrior.OverpowerAura.Deactivate(sim)

		if !result.Landed() {
			spell.IssueRefund(sim)
		}
	}

	warrior.RegisterSpell(config)
}
