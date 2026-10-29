package shaman

import (
	"sort"

	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/spelldata"
	"github.com/wowsims/forever/sim/core/stats"
)

// Shaman healing, added locally for trio planning. Talents that modify these spells (Improved
// Healing Wave, Tidal Focus, Purification, Healing Way) reach them through their class flags, as
// ParseStatic already applies them in talents_restoration.go.

// RegisterHealingSpells registers every rank of Healing Wave, Lesser Healing Wave and Chain Heal, and
// Riptide when talented. Called by the restoration spec's Initialize.
func (shaman *Shaman) RegisterHealingSpells() {
	spellData.HealingWave.Each(func(_ int32, rank *spelldata.Spell) { shaman.RegisterSpell(shaman.directHealConfig(rank)) })
	spellData.LesserHealingWave.Each(func(_ int32, rank *spelldata.Spell) { shaman.RegisterSpell(shaman.directHealConfig(rank)) })
	if shaman.Talents.Riptide {
		spellData.Riptide.Each(shaman.registerRiptide)
	}
	spellData.ChainHeal.Each(shaman.registerChainHeal)
	shaman.applyAncestralHealing()
}

func (shaman *Shaman) healTarget(target *core.Unit) *core.Unit {
	if target == nil || target.Type == core.EnemyUnit {
		return &shaman.Unit
	}
	return target
}

func (shaman *Shaman) directHealConfig(rank *spelldata.Spell) core.SpellConfig {
	heal := rank.HealEffect()
	config := spelldata.SpellConfig(&shaman.Unit, rank, spelldata.Magic(core.ProcMaskSpellHealing))
	config.ApplyEffects = func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
		spell.CalcAndDealHealing(sim, shaman.healTarget(target), heal.Average(), spell.OutcomeHealingCrit)
	}
	return config
}

// Riptide (SoD row): a direct heal plus a 15 sec HoT; the next Chain Heal on the target is 25%
// stronger and consumes it (effect 3's dummy).
func (shaman *Shaman) registerRiptide(_ int32, rank *spelldata.Spell) {
	heal := rank.HealEffect()
	tick := rank.PeriodicEffect()
	config := spelldata.SpellConfig(&shaman.Unit, rank, spelldata.Magic(core.ProcMaskSpellHealing))
	config.Hot = spelldata.DotConfig(rank, tick, spelldata.RankLabel(rank, &shaman.Unit))
	config.Hot.OnTick = func(sim *core.Simulation, target *core.Unit, dot *core.Dot) {
		dot.Spell.CalcAndDealPeriodicHealing(sim, target, tick.Average(), dot.Spell.OutcomeTickHealingCrit)
	}
	config.ApplyEffects = func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
		target = shaman.healTarget(target)
		spell.CalcAndDealHealing(sim, target, heal.Average(), spell.OutcomeHealingCrit)
		spell.Hot(target).Apply(sim)
	}
	shaman.riptides = append(shaman.riptides, shaman.RegisterSpell(config))
}

// Chain Heal: the target, then the most injured other players, each jump keeping ChainAmp of the last.
func (shaman *Shaman) registerChainHeal(_ int32, rank *spelldata.Spell) {
	heal := rank.HealEffect()
	maxHits := max(1, int(heal.ChainTargets))
	bounce := float64(heal.ChainAmp)
	riptideBonus := 1.0
	if r := spellData.Riptide.Known(shaman.Level); r != nil && r.EffectN(3) != nil {
		riptideBonus = 1 + r.EffectN(3).BasePoints/100
	}

	config := spelldata.SpellConfig(&shaman.Unit, rank, spelldata.Magic(core.ProcMaskSpellHealing))
	config.ApplyEffects = func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
		target = shaman.healTarget(target)
		targets := []*core.Unit{target}
		var others []*core.Unit
		for _, u := range shaman.Env.Raid.AllPlayerUnits {
			if u != target && u.CurrentHealth() > 0 {
				others = append(others, u)
			}
		}
		sort.SliceStable(others, func(i, j int) bool {
			return others[i].CurrentHealthPercent() < others[j].CurrentHealthPercent()
		})
		for _, u := range others {
			if len(targets) >= maxHits {
				break
			}
			targets = append(targets, u)
		}

		mult := 1.0
		for i, t := range targets {
			m := mult
			if i == 0 {
				for _, rt := range shaman.riptides {
					if hot := rt.Hot(t); hot.IsActive() {
						m *= riptideBonus
						hot.Deactivate(sim)
						break
					}
				}
			}
			spell.DamageMultiplier *= m
			spell.CalcAndDealHealing(sim, t, heal.Average(), spell.OutcomeHealingCrit)
			spell.DamageMultiplier /= m
			mult *= bounce
		}
	}
	shaman.RegisterSpell(config)
}

// Ancestral Healing: non-periodic crit heals raise the target's armor by the talent's share for 15 sec.
func (shaman *Shaman) applyAncestralHealing() {
	if shaman.Talents.AncestralHealing == 0 {
		return
	}
	pct := spellData.AncestralHealing.Rank(shaman.Talents.AncestralHealing).EffectN(1).BasePoints / 100
	buff := spellData.AncestralFortitude.Highest()
	auras := shaman.NewAllyAuraArray(func(unit *core.Unit) *core.Aura {
		dep := unit.NewDynamicMultiplyStat(stats.Armor, 1+pct)
		return unit.GetOrRegisterAura(spelldata.AuraConfig(buff, spelldata.RankLabel(buff, &shaman.Unit))).AttachStatDependency(dep)
	})
	shaman.MakeProcTriggerAura(core.ProcTrigger{
		Name:     "Ancestral Healing",
		Callback: core.CallbackOnHealDealt,
		ProcMask: core.ProcMaskSpellHealing,
		Handler: func(sim *core.Simulation, _ *core.Spell, result *core.SpellResult) {
			if result.DidCrit() {
				auras.Get(result.Target).Activate(sim)
			}
		},
	})
}
