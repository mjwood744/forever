package warlock

func (warlock *Warlock) registerDrainSoul() {
	rankedSpell := spellData.DrainSoul.KnownUpToID(warlock.Level, 11675)
	drainSoulTick := rankedSpell.PeriodicEffect()

	config := warlock.soulSiphonChannelConfig(rankedSpell, drainSoulTick, nil)
	config.ClassSpellMask = WarlockSpellDrainSoul

	warlock.DrainSoul = warlock.RegisterSpell(config)
}
