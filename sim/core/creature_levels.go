package core

// The creature health, armor and auto-attack dps of a level below 60: the typical normal mob of that level in
// the Classic server tables (the median over CMaNGOS classic-db creature_template, smoothed to rise with
// level), since this build's ExpectedStat rows below 60 are not extracted. Pet sheets read in game at levels
// 11, 17 and 30 put the game files' creature dps at 0.97 to 0.99 of those mobs', so dps is taken at 0.975;
// from level 50 each column blends into the game files' level-60 values.
var creatureLevelStats = [CharacterLevel - 1][3]float64{
	{42, 15, 1.60},      // 1
	{55, 16, 1.75},      // 2
	{71, 41, 2.09},      // 3
	{86, 76, 2.57},      // 4
	{102, 120, 3.62},    // 5
	{120, 174, 4.43},    // 6
	{137, 239, 5.71},    // 7
	{156, 316, 5.76},    // 8
	{176, 406, 6.76},    // 9
	{198, 512, 7.81},    // 10
	{222, 538, 8.60},    // 11
	{247, 573, 10.13},   // 12
	{273, 608, 10.13},   // 13
	{300, 642, 10.93},   // 14
	{328, 677, 12.47},   // 15
	{356, 713, 14.31},   // 16
	{386, 748, 14.31},   // 17
	{417, 782, 15.84},   // 18
	{449, 817, 15.84},   // 19
	{484, 852, 15.84},   // 20
	{542, 888, 17.93},   // 21
	{584, 922, 18.17},   // 22
	{629, 957, 18.97},   // 23
	{677, 992, 20.03},   // 24
	{726, 1026, 20.27},  // 25
	{826, 1061, 22.36},  // 26
	{882, 1097, 22.47},  // 27
	{940, 1130, 24.44},  // 28
	{998, 1165, 24.44},  // 29
	{1052, 1200, 24.69}, // 30
	{1217, 1234, 25.73}, // 31
	{1278, 1270, 26.22}, // 32
	{1343, 1304, 27.02}, // 33
	{1407, 1340, 28.07}, // 34
	{1476, 1373, 30.16}, // 35
	{1689, 1480, 30.40}, // 36
	{1766, 1593, 32.49}, // 37
	{1845, 1709, 32.73}, // 38
	{1929, 1834, 32.73}, // 39
	{2015, 1964, 36.11}, // 40
	{2282, 2101, 36.91}, // 41
	{2377, 2246, 38.20}, // 42
	{2471, 2397, 39.24}, // 43
	{2566, 2557, 42.94}, // 44
	{2660, 2725, 42.94}, // 45
	{2998, 2780, 46.24}, // 46
	{3109, 2835, 47.29}, // 47
	{3221, 2888, 48.33}, // 48
	{3340, 2944, 49.13}, // 49
	{3460, 2999, 50.42}, // 50
	{3922, 3103, 52.59}, // 51
	{4079, 3200, 53.66}, // 52
	{4210, 3284, 56.53}, // 53
	{4314, 3356, 57.65}, // 54
	{4391, 3418, 61.79}, // 55
	{4586, 3470, 62.46}, // 56
	{4577, 3510, 63.37}, // 57
	{4507, 3540, 62.40}, // 58
	{4468, 3558, 61.81}, // 59
}

// Health, armor and auto-attack dps of a creature of the level: the game files' level-60 values at 60 and
// above, creatureLevelStats below.
func CreatureStatsAt(level int32) (health, armor, autoAttackDps float64) {
	if level >= CharacterLevel || level < 1 {
		return CreatureHealthMaxLevel, CreatureArmorMaxLevel, CreatureAutoAttackDpsMaxLevel
	}
	row := creatureLevelStats[level-1]
	return row[0], row[1], row[2]
}
