package hunter

import (
	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/spelldata"
	"github.com/wowsims/forever/sim/core/stats"
)

// What a hunter has for a pet that tanks, added locally for trio planning: the Beast Training a tank
// pet is taught (Great Stamina, Natural Armor, Pet Hardiness), Endurance Training, Mend Pet, Improved
// Mend Pet and Intimidation's threat. Spirit Bond (1% health every 5 s) and Feign Death are not modelled.

// The pet's side: Beast Training ranks from the options, and Endurance Training's health. Called from
// HunterPet.ApplyTalents.
func (hp *HunterPet) applyTankTraining() {
	opts := hp.hunterOwner.Options
	for _, rank := range []*spelldata.Spell{
		spellData.GreatStaminaTriggered.Rank(opts.GreatStamina),
		spellData.NaturalArmorTriggered.Rank(opts.NaturalArmor),
		spellData.PetHardiness.Rank(opts.PetHardiness),
	} {
		if rank != spelldata.Nil {
			spelldata.ParseStatic(&hp.Character, rank)
		}
	}

	// ASSUMPTION: Endurance Training's +3% a rank is the pet's health, as Classic's tooltip reads. Its
	// row only names the pet passive it modifies, which the client's files do not carry.
	if r := hp.hunterOwner.Talents.EnduranceTraining; r > 0 {
		hp.MultiplyStat(stats.Health, 1+spellData.EnduranceTraining.FractionAt(r))
	}
}

// The hunter's side: Mend Pet and Intimidation, registered once the pet exists.
func (hunter *Hunter) registerPetTankSpells() {
	if hunter.Pet == nil {
		return
	}
	hunter.registerMendPet()
	hunter.registerIntimidation()
}

// Mend Pet: a 5 s channel healing the pet each second. Improved Mend Pet cuts its cost by 10% a rank.
func (hunter *Hunter) registerMendPet() {
	pet := hunter.Pet
	rank := spellData.MendPet.Known(hunter.Level)
	tick := rank.PeriodicEffect()
	config := spelldata.SpellConfig(&hunter.Unit, rank, spelldata.Magic(core.ProcMaskSpellHealing))
	config.Flags |= core.SpellFlagChanneled
	config.Hot = spelldata.DotConfig(rank, tick, spelldata.RankLabel(rank, &hunter.Unit))
	config.Hot.OnTick = func(sim *core.Simulation, target *core.Unit, dot *core.Dot) {
		dot.Spell.CalcAndDealPeriodicHealing(sim, target, tick.Average(), dot.OutcomeTick)
	}
	config.ExtraCastCondition = func(_ *core.Simulation, _ *core.Unit) bool {
		return pet.IsEnabled()
	}
	config.ApplyEffects = func(sim *core.Simulation, _ *core.Unit, spell *core.Spell) {
		spell.Hot(&pet.Unit).Apply(sim)
	}
	spell := hunter.RegisterSpell(config)
	if r := hunter.Talents.ImprovedMendPet; r > 0 && spell.Cost != nil {
		spell.Cost.PercentModifier *= 1 + spellData.ImprovedMendPet.EffectAt(2).FractionAt(r)
	}
}

// Intimidation: the pet's next attack stuns and adds flat threat (1,000 at level 60). Modelled as the
// threat landing at once on the pet's target; the stun is left out.
func (hunter *Hunter) registerIntimidation() {
	if !hunter.Talents.Intimidation {
		return
	}
	pet := hunter.Pet
	threatRank := spellData.IntimidationTriggered.Highest()
	threatConfig := pet.spellConfig(threatRank)
	threatConfig.ProcMask = core.ProcMaskEmpty
	threatConfig.ThreatMultiplier = 1
	threatConfig.Flags |= core.SpellFlagNoOnCastComplete
	threatConfig.ApplyEffects = func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
		spell.CalcAndDealOutcome(sim, target, spell.OutcomeAlwaysHit)
	}
	threat := pet.RegisterSpell(threatConfig)

	config := spelldata.SpellConfig(&hunter.Unit, spellData.Intimidation.Highest(), spelldata.Flags(core.SpellFlagAPL))
	config.ExtraCastCondition = func(_ *core.Simulation, _ *core.Unit) bool {
		return pet.IsEnabled() && pet.CurrentTarget != nil
	}
	config.ApplyEffects = func(sim *core.Simulation, _ *core.Unit, _ *core.Spell) {
		threat.Cast(sim, pet.CurrentTarget)
	}
	hunter.RegisterSpell(config)
}
