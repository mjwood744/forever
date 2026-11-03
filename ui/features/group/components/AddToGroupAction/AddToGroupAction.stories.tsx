import { SimHostProvider } from '@sim/context/SimHostContext';
import type { Meta, StoryObj } from '@storybook/react-vite';
import { makeStatSweepHost } from '@testing/storybook/fixtures/stat-sweep';
import { fn } from 'storybook/test';

import { AddToGroupAction } from './AddToGroupAction';

const fixture = makeStatSweepHost({ showPowerTools: true });

const meta = {
	title: 'Features/Group/AddToGroupAction',
	component: AddToGroupAction,
	tags: ['autodocs'],
	args: { onAdded: fn() },
	decorators: [
		Story => (
			<SimHostProvider host={fixture.host}>
				<Story />
			</SimHostProvider>
		),
	],
} satisfies Meta<typeof AddToGroupAction>;

export default meta;
type Story = StoryObj<typeof meta>;

/** The sidebar button; a click saves the character into the group roster. */
export const Default: Story = {};
