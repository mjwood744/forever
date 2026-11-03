import { Player as PlayerProto } from '@generated/proto/api';
import { Class, Spec } from '@generated/proto/common';
import { describe, expect, it } from 'vitest';

import { addToRoster, GROUP_ROSTER_STORAGE_KEY, readRoster, removeFromRoster, renameInRoster, rosterPlayer } from './roster';

const memoryStorage = () => {
	const data = new Map<string, string>();
	return {
		getItem: (k: string) => data.get(k) ?? null,
		setItem: (k: string, v: string) => void data.set(k, v),
		removeItem: (k: string) => void data.delete(k),
	};
};

const mage = PlayerProto.create({ class: Class.ClassMage, talentsString: '230-0505-05' });

describe('the group roster', () => {
	it('starts empty, and reads a corrupt blob as empty', () => {
		const storage = memoryStorage();
		expect(readRoster(storage)).toEqual([]);
		storage.setItem(GROUP_ROSTER_STORAGE_KEY, '{not json');
		expect(readRoster(storage)).toEqual([]);
	});

	it('numbers a second character of the same name', () => {
		const storage = memoryStorage();
		addToRoster(storage, 'Mage', Spec.SpecMage, mage, 1);
		addToRoster(storage, 'Mage', Spec.SpecMage, mage, 2);
		expect(readRoster(storage).map(e => e.name)).toEqual(['Mage', 'Mage 2']);
	});

	it('gives the player back under the entry name', () => {
		const storage = memoryStorage();
		const entry = addToRoster(storage, 'Mage', Spec.SpecMage, mage, 1);
		renameInRoster(storage, entry.id, 'Ashyn');
		const player = rosterPlayer(readRoster(storage)[0]);
		expect(player.name).toBe('Ashyn');
		expect(player.talentsString).toBe('230-0505-05');
	});

	it('removes an entry', () => {
		const storage = memoryStorage();
		const entry = addToRoster(storage, 'Mage', Spec.SpecMage, mage, 1);
		removeFromRoster(storage, entry.id);
		expect(readRoster(storage)).toEqual([]);
	});
});
