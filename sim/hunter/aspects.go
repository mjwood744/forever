package hunter

import (
	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/spelldata"
)

// The hastes the aspects proc: Quick Shots off Hawk, Quick Strikes off Beast.
var quickShots = spellData.AspectOfTheHawkTriggered.Highest()
var quickStrikes = spellData.AspectOfTheBeastTriggered.Highest()

func (hunter *Hunter) registerAspects() {
	aspectOfTheBeastRank := spellData.AspectOfTheBeast.Known(hunter.Level)
	aspectOfTheHawkRank := spellData.AspectOfTheHawk.Known(hunter.Level)
	hunter.AspectOfTheHawkAura = hunter.registerAspect(aspectOfTheHawkRank)
	hunter.AspectOfTheBeastAura = hunter.registerAspect(aspectOfTheBeastRank)

	hunter.registerDeadlyAspects()
}

// One aspect at a time: a permanent self-buff the row's effects parse onto, and the spell that swaps
// to it.
func (hunter *Hunter) registerAspect(row *spelldata.Spell) *core.Aura {
	aura := hunter.RegisterAura(spelldata.AuraConfig(row))
	spelldata.ParseEffects(&hunter.Character, aura, row)
	aura.NewExclusiveEffect("Aspect", true, core.ExclusiveEffect{})

	config := spelldata.SpellConfig(&hunter.Unit, row, spelldata.Flags(core.SpellFlagAPL))

	config.ApplyEffects = func(sim *core.Simulation, _ *core.Unit, spell *core.Spell) {
		aura.Activate(sim)
	}

	config.RelatedSelfBuff = aura

	hunter.RegisterSpell(config)

	return aura
}

// Deadly Aspects is the chance the aspects' own procs fire with: the Hawk and Beast rows state the
// trigger and no rate, and the talent adds SPELLMOD_CHANCE_OF_SUCCESS to each - 2% a rank - which
// the parse has no row for. The tooltip names Auto Shot and melee auto attacks where the rows hear
// every ranged and melee hit, so each mask is narrowed to the white hits by hand.
func (hunter *Hunter) registerDeadlyAspects() {
	aspectOfTheBeastRank := spellData.AspectOfTheBeast.Known(hunter.Level)
	aspectOfTheHawkRank := spellData.AspectOfTheHawk.Known(hunter.Level)
	rank := hunter.Talents.DeadlyAspects
	if rank == 0 {
		return
	}

	quickShotsAura := hunter.RegisterAura(spelldata.AuraConfig(quickShots))
	spelldata.ParseEffects(&hunter.Character, quickShotsAura, quickShots)
	quickStrikesAura := hunter.RegisterAura(spelldata.AuraConfig(quickStrikes))
	spelldata.ParseEffects(&hunter.Character, quickStrikesAura, quickStrikes)

	hawk := spelldata.ProcTrigger(&hunter.Character, aspectOfTheHawkRank,
		func(sim *core.Simulation, spell *core.Spell, result *core.SpellResult) {
			quickShotsAura.Activate(sim)
		},
		spelldata.Chance(spellData.DeadlyAspects.EffectAt(1).FractionAt(rank)))
	hawk.Name = "Deadly Aspects - Quick Shots"
	hawk.ProcMask = core.ProcMaskRangedAuto
	// Quick Shots fires at the Auto Shot's release, a missed shot included.
	hawk.Callback = core.CallbackOnCastResult
	hawk.Outcome = core.OutcomeEmpty
	hawk.RequireDamageDealt = false
	hawk.TriggerImmediately = true
	hunter.AspectOfTheHawkAura.AttachProcTrigger(hawk)

	beast := spelldata.ProcTrigger(&hunter.Character, aspectOfTheBeastRank,
		func(sim *core.Simulation, spell *core.Spell, result *core.SpellResult) {
			quickStrikesAura.Activate(sim)
		},
		spelldata.Chance(spellData.DeadlyAspects.EffectAt(2).FractionAt(rank)))
	beast.Name = "Deadly Aspects - Quick Strikes"
	beast.ProcMask = core.ProcMaskMeleeWhiteHit
	hunter.AspectOfTheBeastAura.AttachProcTrigger(beast)
}
