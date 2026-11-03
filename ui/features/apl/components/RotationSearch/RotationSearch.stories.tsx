import type { Meta, StoryObj } from '@storybook/react-vite';
import { fn } from 'storybook/test';

import { RotationSearchView } from './RotationSearch';

const meta = {
	title: 'Features/Apl/RotationSearch',
	component: RotationSearchView,
	tags: ['autodocs'],
	args: { unit: 'DPS', running: false, onRun: fn() },
} satisfies Meta<typeof RotationSearchView>;

export default meta;
type Story = StoryObj<typeof meta>;

export const Idle: Story = {};
export const Running: Story = { args: { running: true } };
export const Improved: Story = {
	args: {
		result: {
			startScore: 184.2,
			score: 199.4,
			candidates: 255,
			changes: ['Added Fire Blast at the top', 'Added Fireball at the top', 'Moved Fire Blast above Fireball', 'Added Shoot at the bottom'],
		},
	},
};
export const Unchanged: Story = { args: { unit: 'TPS', result: { startScore: 812.5, score: 812.5, candidates: 64, changes: [] } } };
export const Failed: Story = { args: { error: 'HTTP 404 from /rotationSearch' } };
