import type { Meta, StoryObj } from '@storybook/react-vite';

import { TierBoard, type TierRow } from './TierBoard';

const row = (
	key: string,
	label: string,
	classColor: string,
	subtitle: string,
	value: number,
	share: number,
	tier: TierRow['tier'],
	change?: number,
): TierRow => ({
	key,
	tier,
	label,
	tags: [],
	classColor,
	subtitle,
	icon: '',
	value,
	unit: 'DPS',
	share,
	change,
	details: <span>{key}</span>,
});

const meta = {
	title: 'App/Group/TierBoard',
	component: TierBoard,
	tags: ['autodocs'],
	parameters: { spec: 'none' },
	args: {
		title: 'Damage builds: Raid',
		note: 'DPS on the preset build, single target.',
		rows: [
			row('sv', 'Survival', 'hunter', 'Hunter · 0/20/31', 1288, 1, 'S', 0.082),
			row('ret', 'Retribution', 'paladin', 'Paladin · 12/8/31', 996, 0.77, 'C'),
			row('fire', 'Fire', 'mage', 'Mage · 16/32/3', 1220, 0.95, 'A', -0.02),
			row('destro', 'Destruction', 'warlock', 'Warlock · 20/0/31', 1050, 0.82, 'B'),
		],
	},
} satisfies Meta<typeof TierBoard>;

export default meta;
type Story = StoryObj<typeof meta>;

/** One build in each tier, two with a change since the last run. */
export const Default: Story = {};
