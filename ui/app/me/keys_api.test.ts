import { describe, expect, it } from 'vitest';

import { type DuoKeys, keyFor } from './keys_api';

const skarr: DuoKeys = {
	from: 'Skarr',
	level: 32,
	keys: [
		{ key: 'Shift+2', when: 'your health is under 30%', main: false, extra: false, spells: ['Shield Wall'] },
		{ key: '2', when: '2 or more mobs are on you', main: false, extra: false, spells: ['Cleave', 'Thunder Clap'] },
		{ key: '1', when: '', main: true, extra: false, spells: ['Revenge', 'Thunder Clap'] },
		{ key: 'T', when: '', main: false, extra: true, spells: ['Taunt'] },
	],
};

describe('keyFor', () => {
	it('puts a spell the mashed key casts on the mashed key', () => {
		expect(keyFor(skarr, 'Thunder Clap')?.key).toBe('1');
	});
	it('puts a spell only another key casts on that key', () => {
		expect(keyFor(skarr, 'Cleave')?.key).toBe('2');
		expect(keyFor(skarr, 'Taunt')?.key).toBe('T');
	});
	it('finds no key for a spell no key casts', () => {
		expect(keyFor(skarr, 'Heroic Strike')).toBeUndefined();
	});
});
