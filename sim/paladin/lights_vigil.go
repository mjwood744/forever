package paladin

import (
	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/spelldata"
)

var LightsVigilRankMap = spellData.LightsVigil

// The vigil aura and party heal each rank triggers. They sit in LightsVigilTriggered in the order
// the game files list them, not by rank, so the pairing is by hand.
var lightsVigilTriggered = map[int32]struct{ aura, heal int32 }{
	1: {aura: 1310909, heal: 1310912},
	2: {aura: 1311593, heal: 1311591},
	3: {aura: 1311597, heal: 1311596},
}

// One rank's vigils on allies and the party heal that consumes one.
type lightsVigil struct {
	auras core.AuraArray
	heal  *core.Spell

	// Ends the vigil on the ally: their party is healed.
	consume func(sim *core.Simulation, target *core.Unit)
}

// Light's Vigil (talent)
// https://www.wowhead.com/forever/spell=1311595
//
// Applies Light's Vigil to a friendly target for 30 sec. Your next Holy Shock cast on them triggers
// no cooldown and heals their party. Since the 31 October hotfixes it cannot be put on an enemy.
// You may only have one Light's Vigil active per party.
func (paladin *Paladin) registerLightsVigil() {
	LightsVigilRankMap.Each(paladin.registerLightsVigilRank)
}

func (paladin *Paladin) registerLightsVigilRank(_ int32, rank *spelldata.Spell) {
	triggered := lightsVigilTriggered[rank.RankNumber()]
	auraRank := spellData.LightsVigilTriggered.ByID(triggered.aura)
	healRank := spellData.LightsVigilTriggered.ByID(triggered.heal)
	healEffect := healRank.HealEffect()

	vigil := &lightsVigil{}
	paladin.lightsVigils = append(paladin.lightsVigils, vigil)

	vigil.auras = paladin.NewAllyAuraArray(func(target *core.Unit) *core.Aura {
		return target.GetOrRegisterAura(core.Aura{
			Label:    rank.RankedLabel(&paladin.Unit),
			ActionID: core.ActionID{SpellID: auraRank.ID},
			Duration: auraRank.Duration(),
		})
	})

	healConfig := spelldata.SpellConfig(&paladin.Unit, healRank, spelldata.Magic(core.ProcMaskSpellHealing), spelldata.Proc())
	healConfig.Flags |= core.SpellFlagHelpful
	healConfig.ClassSpellMask = SpellMaskLightsVigilHeal
	healConfig.ApplyEffects = func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
		spell.CalcAndDealHealing(sim, target, healEffect.Average(), spell.OutcomeHealingCrit)
	}
	vigil.heal = paladin.RegisterSpell(healConfig)

	config := spelldata.SpellConfig(&paladin.Unit, rank, spelldata.Flags(core.SpellFlagAPL|core.SpellFlagHelpful))
	config.ProcMask = core.ProcMaskEmpty
	config.ClassSpellMask = SpellMaskLightsVigil
	config.ApplyEffects = func(sim *core.Simulation, target *core.Unit, _ *core.Spell) {
		// One vigil per paladin: a new one replaces whatever was up.
		for _, other := range paladin.lightsVigils {
			for _, aura := range other.auras {
				if aura != nil && aura.IsActive() {
					aura.Deactivate(sim)
				}
			}
		}
		vigil.auras.Get(target).Activate(sim)
	}
	config.RelatedAuraArrays = vigil.auras.ToMap()
	paladin.RegisterSpell(config)

	// The heal reaches the ally's party, which in the sim is the paladin's own.
	vigil.consume = func(sim *core.Simulation, target *core.Unit) {
		vigil.auras.Get(target).Deactivate(sim)
		if paladin.Party == nil {
			vigil.heal.Cast(sim, target)
			return
		}
		for _, member := range paladin.Party.Players {
			vigil.heal.Cast(sim, &member.GetCharacter().Unit)
		}
	}
}

// The vigil this paladin has on the ally, if any.
func (paladin *Paladin) activeLightsVigil(target *core.Unit) *lightsVigil {
	for _, vigil := range paladin.lightsVigils {
		if aura := vigil.auras.Get(target); aura != nil && aura.IsActive() {
			return vigil
		}
	}
	return nil
}
