import { ActivityFight, ActivityPlayerSummary, ActivitySimResult } from '@generated/proto/api';
import type { Meta, StoryObj } from '@storybook/react-vite';

import { GroupResults } from './GroupResults';

const player = (name: string, p: Partial<ActivityPlayerSummary>) => ActivityPlayerSummary.create({ name, ...p });

const players = [
	player('Rellyk', { dps: 121, tps: 265, dtps: 118, chanceOfDeath: 0.04 }),
	player('Ashyn', { dps: 254, tps: 288, manaSpent: 72272, secondsOom: 14 }),
	player('Nyxx', { hps: 96, manaSpent: 33710 }),
];

const result = ActivitySimResult.create({
	groupDps: 375,
	totalSeconds: 1126,
	players,
	fights: [
		ActivityFight.create({ label: '3 Gnolls', targetCount: 3, seconds: 41, players }),
		ActivityFight.create({ label: 'Boss: Faldrim Anvilmar', targetCount: 1, boss: true, seconds: 63, players }),
	],
});

const meta = {
	title: 'App/Group/GroupResults',
	component: GroupResults,
	tags: ['autodocs'],
	parameters: { spec: 'none' },
	args: { result, roles: ['tank', 'dps', 'healer'], tank: 0 },
} satisfies Meta<typeof GroupResults>;

export default meta;
type Story = StoryObj<typeof meta>;

/** A dungeon run where the mage's threat passes the tank's, so the warning shows. */
export const Default: Story = {};
