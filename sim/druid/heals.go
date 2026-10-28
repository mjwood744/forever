package druid

import (
	"sort"

	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/spelldata"
)

// Druid healing, added locally for trio planning. Talent mods (Gift of Nature, Improved
// Rejuvenation, Improved Regrowth, Tranquil Spirit, Naturalist, Gift of the Earthmother) reach these
// spells through their class flags via parseTalent. Tranquility is not modelled.

// RegisterHealingSpells is called by the restoration spec's Initialize.
func (druid *Druid) RegisterHealingSpells() {
	spellData.HealingTouch.Each(func(_ int32, r *spelldata.Spell) { druid.registerDirectHeal(r) })
	spellData.Rejuvenation.Each(func(_ int32, r *spelldata.Spell) { druid.rejuvs = append(druid.rejuvs, druid.registerHot(r)) })
	spellData.Regrowth.Each(func(_ int32, r *spelldata.Spell) { druid.regrowths = append(druid.regrowths, druid.registerRegrowth(r)) })
	if druid.Talents.Swiftmend {
		druid.registerSwiftmend()
	}
	if druid.Talents.WildGrowth {
		spellData.WildGrowth.Each(druid.registerWildGrowth)
	}
}

func (druid *Druid) healTarget(target *core.Unit) *core.Unit {
	if target == nil || target.Type == core.EnemyUnit {
		return &druid.Unit
	}
	return target
}

func (druid *Druid) registerDirectHeal(rank *spelldata.Spell) *core.Spell {
	heal := rank.HealEffect()
	config := spelldata.SpellConfig(&druid.Unit, rank, spelldata.Magic(core.ProcMaskSpellHealing))
	config.ApplyEffects = func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
		spell.CalcAndDealHealing(sim, druid.healTarget(target), heal.Average(), spell.OutcomeHealingCrit)
	}
	return druid.RegisterSpell(config).Spell
}

func (druid *Druid) hotConfig(rank *spelldata.Spell, tick *spelldata.Effect) core.SpellConfig {
	config := spelldata.SpellConfig(&druid.Unit, rank, spelldata.Magic(core.ProcMaskSpellHealing))
	config.Hot = spelldata.DotConfig(rank, tick, spelldata.RankLabel(rank, &druid.Unit))
	config.Hot.OnTick = func(sim *core.Simulation, target *core.Unit, dot *core.Dot) {
		dot.Spell.CalcAndDealPeriodicHealing(sim, target, tick.Average(), dot.Spell.OutcomeTickHealingCrit)
	}
	return config
}

func (druid *Druid) registerHot(rank *spelldata.Spell) *core.Spell {
	config := druid.hotConfig(rank, rank.PeriodicEffect())
	config.ApplyEffects = func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
		spell.Hot(druid.healTarget(target)).Apply(sim)
	}
	return druid.RegisterSpell(config).Spell
}

// Regrowth: a direct heal and a HoT.
func (druid *Druid) registerRegrowth(rank *spelldata.Spell) *core.Spell {
	heal := rank.HealEffect()
	config := druid.hotConfig(rank, rank.PeriodicEffect())
	config.ApplyEffects = func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
		target = druid.healTarget(target)
		spell.CalcAndDealHealing(sim, target, heal.Average(), spell.OutcomeHealingCrit)
		spell.Hot(target).Apply(sim)
	}
	return druid.RegisterSpell(config).Spell
}

// Swiftmend: heals a target that has Rejuvenation or Regrowth for 12 sec of Rejuvenation or 18 sec of
// Regrowth (the Classic amounts; the Forever row states a placeholder of 1). In Forever it does NOT
// consume the HoT (confirmed in game by the user).
func (druid *Druid) registerSwiftmend() {
	rank := spellData.Swiftmend.Highest()
	config := spelldata.SpellConfig(&druid.Unit, rank, spelldata.Magic(core.ProcMaskSpellHealing))
	config.ExtraCastCondition = func(_ *core.Simulation, target *core.Unit) bool {
		return druid.swiftmendSource(druid.healTarget(target)) != nil
	}
	config.ApplyEffects = func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
		target = druid.healTarget(target)
		src := druid.swiftmendSource(target)
		if src == nil {
			return
		}
		tick := src.rank.PeriodicEffect()
		ticks := float64(src.seconds) / tick.Period().Seconds()
		spell.BonusCoefficient = tick.Coeff() * ticks
		spell.CalcAndDealHealing(sim, target, tick.Average()*ticks, spell.OutcomeHealingCrit)
	}
	druid.RegisterSpell(config)
}

type swiftmendSrc struct {
	rank    *spelldata.Spell
	seconds int
}

func (druid *Druid) swiftmendSource(target *core.Unit) *swiftmendSrc {
	for i, s := range druid.rejuvs {
		if s.Hot(target).IsActive() {
			return &swiftmendSrc{spellData.Rejuvenation.Rank(int32(i + 1)), 12}
		}
	}
	for i, s := range druid.regrowths {
		if s.Hot(target).IsActive() {
			return &swiftmendSrc{spellData.Regrowth.Rank(int32(i + 1)), 18}
		}
	}
	return nil
}

// Wild Growth: a HoT on the target and the most injured other party members, up to the row's
// dummy target count (effect 2, 5 targets).
func (druid *Druid) registerWildGrowth(_ int32, rank *spelldata.Spell) {
	tick := rank.PeriodicEffect()
	maxTargets := 5
	if e := rank.EffectN(2); e != nil && e.BasePoints > 0 {
		maxTargets = int(e.BasePoints)
	}
	config := druid.hotConfig(rank, tick)
	config.ApplyEffects = func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
		target = druid.healTarget(target)
		targets := []*core.Unit{target}
		var others []*core.Unit
		for _, u := range druid.Env.Raid.AllUnits {
			if u != target && u.CurrentHealth() > 0 {
				others = append(others, u)
			}
		}
		sort.SliceStable(others, func(i, j int) bool { return others[i].CurrentHealthPercent() < others[j].CurrentHealthPercent() })
		for _, u := range others {
			if len(targets) >= maxTargets {
				break
			}
			targets = append(targets, u)
		}
		for _, t := range targets {
			spell.Hot(t).Apply(sim)
		}
	}
	druid.RegisterSpell(config)
}

// Forever restoration talents that only modify the spells above through class flags.
func (druid *Druid) applyHealingTalentMods() {
	druid.parseTalent(druid.Talents.GiftOfNature, spellData.GiftOfNature)
	if druid.Talents.GiftOfTheEarthmother {
		druid.parseTalent(1, spellData.GiftOfTheEarthmother)
	}
	druid.parseTalent(druid.Talents.TranquilSpirit, spellData.TranquilSpirit)
	druid.parseTalent(druid.Talents.ImprovedRejuvenation, spellData.ImprovedRejuvenation)
	druid.parseTalent(druid.Talents.ImprovedRegrowth, spellData.ImprovedRegrowth)
	druid.parseTalent(druid.Talents.ImprovedTranquility, spellData.ImprovedTranquility)
}
