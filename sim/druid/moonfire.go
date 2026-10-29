package druid

import (
	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/spelldata"
)

// One spell carries the hit and the dot, so a SPELLMOD_DAMAGE (direct hits only) and a SPELLMOD_DOT
// (ticks only) that both name Moonfire, as Improved Moonfire does, each reach their own half.
func (druid *Druid) registerMoonfireSpell() {
	moonfireRank := spellData.Moonfire.Known(druid.Level)
	moonfireDamage := moonfireRank.DamageEffect()
	config := spelldata.SpellConfig(&druid.Unit, moonfireRank, spelldata.Magic(core.ProcMaskSpellDamage))
	config.Dot = spelldata.DotConfig(moonfireRank, moonfireRank.PeriodicEffect())

	config.ApplyEffects = func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
		result := spell.CalcDamage(sim, target, moonfireDamage.Average(), spell.OutcomeMagicHitAndCrit)
		if result.Landed() {
			spell.Dot(target).Apply(sim)
		}
		spell.DealDamage(sim, result)
	}

	druid.Moonfire = druid.RegisterSpell(config)
}
