package shaman

import (
	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/spelldata"
)

func (shaman *Shaman) registerLavaBurstSpells() {
	castableRanks(spellData.LavaBurst, shaman.Level).Each(func(_ int32, rank *spelldata.Spell) { shaman.registerLavaBurstSpell(rank) })
}

// Lightning Overload names Lightning Bolt and Chain Lightning only, so Lava Burst has no overload.
func (shaman *Shaman) registerLavaBurstSpell(rank *spelldata.Spell) {
	config := spelldata.SpellConfig(&shaman.Unit, rank,
		spelldata.Magic(core.ProcMaskSpellDamage),
		spelldata.Flags(SpellFlagShamanSpell|SpellFlagFocusable))

	// "If your Flame Shock is on the target, Lava Burst deals $s2% increased damage."
	damage := rank.DamageEffect()
	flameShockBonus := 1 + rank.EffectN(2).Average()/100

	config.ApplyEffects = func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
		multiplier := 1.0
		if shaman.flameShockActive(target) {
			multiplier = flameShockBonus
		}

		spell.DamageMultiplier *= multiplier
		result := spell.CalcDamage(sim, target, damage.Average(), spell.OutcomeMagicHitAndCrit)
		spell.DamageMultiplier /= multiplier

		spell.DealDamageAfterTravel(sim, result)
	}

	shaman.LavaBurst = append(shaman.LavaBurst, shaman.RegisterSpell(config))
}

// Whether a rank of the shaman's Flame Shock is on the target. The ranks share one dot per target.
func (shaman *Shaman) flameShockActive(target *core.Unit) bool {
	for _, flameShock := range shaman.FlameShock {
		if flameShock.Dot(target).IsActive() {
			return true
		}
	}
	return false
}
