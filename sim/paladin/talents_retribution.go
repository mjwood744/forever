package paladin

import (
	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/buffs"
	"github.com/wowsims/forever/sim/core/dbcenums"
	"github.com/wowsims/forever/sim/core/spelldata"
)

func (paladin *Paladin) registerRetributionTalents() {
	// Tier 1
	paladin.applyDeflection()
	paladin.applyBenediction()

	// Tier 2
	paladin.applyImprovedJudgement()
	paladin.applyConviction()
	paladin.applySanctifiedJudgement()

	// Tier 3
	paladin.applyVindication()
	// Unyielding Faith shortens Fear and Disorient effects, which the sim never suffers.
	// Seal of Command registered in registerTalentSpells
	paladin.applyPursuitOfJustice()

	// Tier 4
	paladin.applyEyeForAnEye()
	paladin.applySacredArbiter()
	paladin.applyTwoHandedWeaponSpecialization()

	// Tier 5
	// Turn Evil changes Turn Undead, which the sim does not model.
	paladin.applyVengeance()

	// Tier 6
	paladin.applyChampionOfTheLight()

	// Tier 7
	paladin.applyUnsealedWrath()

	// Holy Conduit moved to the Holy tree with the 31 October hotfixes.
	paladin.applyHolyConduit()
}

// Deflection - Increases your Parry chance by 1/2/3/4/5%.
func (paladin *Paladin) applyDeflection() {
	if paladin.Talents.Deflection == 0 {
		return
	}

	spelldata.ParseStatic(&paladin.Character, spellData.Deflection.Rank(paladin.Talents.Deflection))
}

// Benediction - Reduces the Mana cost of all instant cast spells and abilities by 2/4/6/8/10%.
// TODO: Verify in game that Benediction does not reduce Divine Favor's cost.
func (paladin *Paladin) applyBenediction() {
	if paladin.Talents.Benediction == 0 {
		return
	}

	spelldata.ParseStatic(&paladin.Character, spellData.Benediction.Rank(paladin.Talents.Benediction))
}

// Improved Judgement - Decreases the cooldown of your Judgement ability by 1/2 sec.
func (paladin *Paladin) applyImprovedJudgement() {
	if paladin.Talents.ImprovedJudgement == 0 {
		return
	}

	spelldata.ParseStatic(&paladin.Character, spellData.ImprovedJudgement.Rank(paladin.Talents.ImprovedJudgement))
}

// Holy Conduit - Reduces the mana cost of your Consecration, Holy Wrath, Exorcism, and Hammer of
// Wrath spells by 20/40%.
func (paladin *Paladin) applyHolyConduit() {
	if paladin.Talents.HolyConduit == 0 {
		return
	}

	spelldata.ParseStatic(&paladin.Character, spellData.HolyConduit.Rank(paladin.Talents.HolyConduit))
}

// Conviction - Improves your chance to get a critical strike with melee attacks by 1/2/3/4/5%.
func (paladin *Paladin) applyConviction() {
	if paladin.Talents.Conviction == 0 {
		return
	}

	spelldata.ParseStatic(&paladin.Character, spellData.Conviction.Rank(paladin.Talents.Conviction))
}

// Vindication - Gives your damaging melee attacks a chance to reduce the target's Attack Power by
// 68/136/204, and increase your Attack Power by 1/2/3% for 30 sec.
//
// The game files state no rate: the ProcChance column reads 100. In game it is 5 procs a minute on
// the weapon's speed, measured on swings at 1.6 and 3.6 speed under Seal of Righteousness and Seal of
// the Crusader. The row's proc mask also hears melee specials, which roll the same chance and were not
// measured. The target's debuff is a magic spell of its own that can miss. Since the 31 October
// hotfixes it no longer raises the paladin's own attack power.
const vindicationPPM = 5

func (paladin *Paladin) applyVindication() {
	if paladin.Talents.Vindication == 0 {
		return
	}

	rank := spellData.VindicationTriggered.Highest()

	// The raid's Vindication row, priced at the paladin's ranks, so it bids against the raid's
	// attack power reductions rather than stacking with them.
	targetAuras := paladin.NewEnemyAuraArray(func(target *core.Unit) *core.Aura {
		return buffs.NewSpellDataVindicationAura(target, true, paladin.Talents.Vindication)
	})

	// Magic in SpellCategories: the debuff rolls spell hit.
	debuffConfig := spelldata.SpellConfig(&paladin.Unit, rank, spelldata.Magic(core.ProcMaskEmpty), spelldata.Proc())
	debuffConfig.ApplyEffects = func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
		result := spell.CalcOutcome(sim, target, spell.OutcomeMagicHit)
		if result.Landed() {
			targetAuras.Get(target).Activate(sim)
		}
		spell.DealOutcome(sim, result)
	}
	debuffConfig.RelatedAuraArrays = targetAuras.ToMap()
	debuff := paladin.RegisterSpell(debuffConfig)

	trigger := spelldata.ProcTrigger(&paladin.Character, spellData.Vindication.Rank(paladin.Talents.Vindication), func(sim *core.Simulation, _ *core.Spell, result *core.SpellResult) {
		debuff.Cast(sim, result.Target)
	}, spelldata.WeaponSpeedPPM(vindicationPPM))
	trigger.Name = "Vindication - Trigger" + paladin.Label
	paladin.MakeProcTriggerAura(trigger)
}

// Sanctified Judgement - Gives your Judgement ability a 33/66/100% chance to return 20/40/60% of
// the Mana cost of the judged seal.
//
// Judgement fires no on-cast callbacks, so the trigger hears the judgement the seal unleashes: one
// per Judgement, and it refunds whether or not that judgement lands.
//
// The refund is a share of the seal's base cost, and the seal's percent cost cut goes on top of it
// instead of coming off: Benediction 5/5 makes a 90 mana seal cost 81 and refund 60% of 99, measured
// in game at Benediction 0, 1, 3 and 5 with Sanctified Judgement 3/3. With Twist of Light as a second
// cut the two are multiplied before the flip, which no log has checked.
func (paladin *Paladin) applySanctifiedJudgement() {
	if paladin.Talents.SanctifiedJudgement == 0 {
		return
	}

	manaMetrics := paladin.NewManaMetrics(core.ActionID{SpellID: spellData.SanctifiedJudgement.Highest().ID})
	manaMetrics.NoThreat = spellData.SanctifiedJudgementTriggered.Highest().GeneratesNoThreat()
	refund := spellData.SanctifiedJudgement.EffectAt(2).FractionAt(paladin.Talents.SanctifiedJudgement)

	paladin.MakeProcTriggerAura(core.ProcTrigger{
		Name:               "Sanctified Judgement" + paladin.Label,
		Callback:           core.CallbackOnSpellHitDealt,
		ClassSpellMask:     SpellMaskAllJudgements,
		ProcChance:         spellData.SanctifiedJudgement.EffectAt(1).FractionAt(paladin.Talents.SanctifiedJudgement),
		TriggerImmediately: true,
		Handler: func(sim *core.Simulation, _ *core.Spell, _ *core.SpellResult) {
			if seal := paladin.activeSeal(); seal != nil {
				cost := seal.spell.Cost
				paladin.AddMana(sim, float64(cost.BaseCost)*(2-cost.PercentModifier)*refund, manaMetrics)
			}
		},
	})
}

// The spell that deals Eye for an Eye's damage. The talent's dummy names no trigger, so no ladder
// reaches it.
var eyeForAnEyeDamage = spelldata.MustFind(25997)

// Eye for an Eye - All critical strikes against you cause 5/10% of the damage taken to the attacker
// as well. The damage caused by Eye for an Eye will not exceed 50% of the Paladin's total health.
func (paladin *Paladin) applyEyeForAnEye() {
	if paladin.Talents.EyeForAnEye == 0 {
		return
	}

	share := spellData.EyeForAnEye.FractionAt(paladin.Talents.EyeForAnEye)

	var reflected float64
	config := spelldata.SpellConfig(&paladin.Unit, eyeForAnEyeDamage, spelldata.Magic(core.ProcMaskEmpty), spelldata.Proc(),
		spelldata.Flags(core.SpellFlagBinary|core.SpellFlagIgnoreModifiers))
	config.ApplyEffects = func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
		spell.CalcAndDealDamage(sim, target, reflected, spell.OutcomeAlwaysHit)
	}
	reflect := paladin.RegisterSpell(config)

	trigger := spelldata.ProcTrigger(&paladin.Character, spellData.EyeForAnEye.Rank(paladin.Talents.EyeForAnEye), func(sim *core.Simulation, spell *core.Spell, result *core.SpellResult) {
		reflected = min(result.Damage*share, paladin.MaxHealth()/2)
		reflect.Cast(sim, spell.Unit)
	})
	trigger.Name = "Eye for an Eye" + paladin.Label
	paladin.MakeProcTriggerAura(trigger)
}

// Pursuit of Justice - Increases movement speed and mounted movement speed by 8/15%. This does not
// stack with other movement speed increasing effects.
func (paladin *Paladin) applyPursuitOfJustice() {
	if paladin.Talents.PursuitOfJustice == 0 {
		return
	}

	rank := spellData.PursuitOfJustice.Highest()
	paladin.NewPassiveMovementSpeedAura(
		"Pursuit of Justice",
		core.ActionID{SpellID: rank.ID},
		spellData.PursuitOfJustice.Effect(dbcenums.A_MOD_INCREASE_SPEED, 0).FractionAt(paladin.Talents.PursuitOfJustice),
	)
}

// Sacred Arbiter - Increases the damage of your Holy Strike ability by 20% and causes it to refresh
// all Judgement effects on the target.
func (paladin *Paladin) applySacredArbiter() {
	if !paladin.Talents.SacredArbiter {
		return
	}

	spelldata.ParseStatic(&paladin.Character, spellData.SacredArbiter.Rank(1))

	paladin.MakeProcTriggerAura(core.ProcTrigger{
		Name:               "Sacred Arbiter" + paladin.Label,
		Callback:           core.CallbackOnSpellHitDealt,
		ClassSpellMask:     SpellMaskHolyStrike,
		Outcome:            core.OutcomeLanded,
		TriggerImmediately: true,
		Handler: func(sim *core.Simulation, _ *core.Spell, result *core.SpellResult) {
			for _, aura := range result.Target.GetAurasWithTag(buffs.JudgementAuraTag) {
				if aura.IsActive() {
					aura.Refresh(sim)
				}
			}
		},
	})
}

// Two-Handed Weapon Specialization - Increases the damage you deal with two-handed melee weapons
// by 2/4/6%.
func (paladin *Paladin) applyTwoHandedWeaponSpecialization() {
	if paladin.Talents.TwoHandedWeaponSpecialization == 0 {
		return
	}

	spelldata.ParseStatic(&paladin.Character,
		spellData.TwoHandedWeaponSpecialization.Rank(paladin.Talents.TwoHandedWeaponSpecialization))
}

// Vengeance - Increases your Physical and Holy damage dealt by 1/2/3% for 30 sec after landing a
// non-periodic critical strike. Stacks up to 3 times.
func (paladin *Paladin) applyVengeance() {
	if paladin.Talents.Vengeance == 0 {
		return
	}

	rank := spellData.VengeanceTriggered.Highest()
	perStack := spellData.Vengeance.FractionAt(paladin.Talents.Vengeance)

	damageMod := paladin.AddDynamicMod(core.SpellModConfig{
		School:     core.SpellSchoolHoly | core.SpellSchoolPhysical,
		Kind:       core.SpellMod_DamageDone_Pct,
		FloatValue: perStack,
	})

	vengeance := paladin.RegisterAura(core.Aura{
		Label:     "Vengeance" + paladin.Label,
		ActionID:  core.ActionID{SpellID: rank.ID},
		Duration:  rank.Duration(),
		MaxStacks: int32(rank.MaxStack),
		OnGain: func(_ *core.Aura, _ *core.Simulation) {
			damageMod.Activate()
		},
		OnExpire: func(_ *core.Aura, _ *core.Simulation) {
			damageMod.Deactivate()
		},
		OnStacksChange: func(_ *core.Aura, _ *core.Simulation, _, newStacks int32) {
			damageMod.UpdateFloatValue(perStack * float64(newStacks))
		},
	})

	trigger := spelldata.ProcTrigger(&paladin.Character, spellData.Vengeance.Rank(paladin.Talents.Vengeance), func(sim *core.Simulation, _ *core.Spell, _ *core.SpellResult) {
		vengeance.Activate(sim)
		vengeance.AddStack(sim)
	})
	trigger.Name = "Vengeance - Trigger" + paladin.Label
	// The row's harmful-spell bit hears every spell, item spells too, which the sim registers under
	// ProcMaskEmpty, so the listener hears any hit.
	trigger.ProcMask = core.ProcMaskUnknown
	paladin.MakeProcTriggerAura(trigger)
}

// Champion of the Light - Increases your spell damage by up to 10/20/30% of your Intellect.
func (paladin *Paladin) applyChampionOfTheLight() {
	if paladin.Talents.ChampionOfTheLight == 0 {
		return
	}

	spelldata.ParseStatic(&paladin.Character, spellData.ChampionOfTheLight.Rank(paladin.Talents.ChampionOfTheLight))
}

// Unsealed Wrath - Your melee attacks store a copy of your active Seal for 15 sec, stacking up to 3
// times; an attack under a different Seal spends them. Not modelled: no character in the trios takes it.
func (paladin *Paladin) applyUnsealedWrath() {
}
