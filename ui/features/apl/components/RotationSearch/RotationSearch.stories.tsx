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
// A Destruction warlock's search, from the destro build in its preset gear.
export const Improved: Story = {
	args: {
		result: {
			startScore: 731.6,
			score: 746.3,
			candidates: 112,
			changes: ['Added Shoot at the bottom', 'Moved Bane of Doom above Your assigned curse', 'Moved Immolate to the top'],
			prepull: [
				{ ability: 'Incinerate', conditional: false, when: '', steps: [] },
				{ ability: 'Send your pet in', conditional: true, when: '1s before the pull', steps: [] },
				{ ability: 'Immolate', conditional: true, when: '0.5s before the pull', steps: [] },
			],
			steps: [
				{ ability: 'Immolate', conditional: false, when: '', steps: [] },
				{ ability: 'Any other cooldown that is ready', conditional: false, when: '', steps: [] },
				{ ability: 'Bane of Doom', conditional: true, when: 'the time left in the fight is above 60s', steps: [] },
				{ ability: 'Your assigned curse', conditional: true, when: 'your assigned curse is not on the target', steps: [] },
				{ ability: 'Bane of Havoc', conditional: true, when: 'the number of enemies is above 1 and Bane of Havoc is not active', steps: [] },
				{ ability: 'Amplify Curse', conditional: true, when: 'Bane of Doom is not on the target and Bane of Agony is not on the target', steps: [] },
				{ ability: 'Bane of Agony', conditional: true, when: 'Bane of Doom is not on the target and Bane of Agony is not on the target', steps: [] },
				{ ability: 'Corruption', conditional: false, when: '', steps: [] },
				{ ability: 'Conflagrate', conditional: true, when: 'Immolate is on the target', steps: [] },
				{ ability: 'Shadowburn', conditional: false, when: '', steps: [] },
				{ ability: 'Life Tap', conditional: true, when: 'your mana is below 30% and the time left in the fight is above 60s', steps: [] },
				{ ability: 'Incinerate', conditional: false, when: '', steps: [] },
				{ ability: 'Life Tap', conditional: false, when: '', steps: [] },
				{ ability: 'Shoot', conditional: false, when: '', steps: [] },
			],
		},
		onUndo: fn(),
	},
};
export const Unchanged: Story = {
	args: {
		unit: 'TPS',
		result: {
			startScore: 812.5,
			score: 812.5,
			candidates: 64,
			changes: [],
			prepull: [],
			steps: [
				{ ability: 'Sunder Armor', conditional: false, when: '', steps: [] },
				{ ability: 'Heroic Strike', conditional: true, when: 'your rage is at least 50', steps: [] },
			],
		},
	},
};
export const Failed: Story = {
	args: { error: 'HTTP 404 from /rotationSearch' },
};
