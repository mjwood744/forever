import { Class, ItemSlot, Race } from '@generated/proto/common';
import { describe, expect, it } from 'vitest';

import { CHARACTER_PREFIX, parseCharacterString, PLAN_PREFIX, planString, talentsStringFor } from './fms_strings';

const encode = (prefix: string, text: string) => prefix + btoa(text);
const decode = (prefix: string, str: string) => atob(str.slice(prefix.length));

const SKARR = [
	'name=Skarr',
	'realm=Forever',
	'class=WARRIOR',
	'race=Orc',
	'level=32',
	'role=main',
	'talent=Protection|Shield Specialization|5',
	'talent=Protection|Anticipation|5',
	'talent=Arms|Improved Heroic Strike|3',
	'talent=Protection|Not A Talent|1',
	'gear=1|7719|0',
	'gear=4|45|0',
	'gear=16|1234|56',
].join('\n');

describe('parseCharacterString', () => {
	it('reads an /fms export string', () => {
		const c = parseCharacterString(encode(CHARACTER_PREFIX, SKARR));
		expect(c.name).toBe('Skarr');
		expect(c.cls).toBe(Class.ClassWarrior);
		expect(c.race).toBe(Race.RaceOrc);
		expect(c.level).toBe(32);
		expect(c.role).toBe('main');
		expect(c.talents).toHaveLength(4);
		expect(c.equipment.items[ItemSlot.ItemSlotHead].id).toBe(7719);
		expect(c.equipment.items[ItemSlot.ItemSlotMainHand]).toMatchObject({
			id: 1234,
			enchant: 56,
		});
		// the shirt has no sim slot
		expect(c.equipment.items.filter(i => i.id).length).toBe(2);
	});

	it('explains a plan string pasted in the wrong place', () => {
		expect(() => parseCharacterString(`${PLAN_PREFIX}abc`)).toThrow(/\/fms import/);
	});

	it('explains a string that is not from FMS', () => {
		expect(() => parseCharacterString('hello')).toThrow(/not an \/fms export string/);
	});
});

describe('talentsStringFor', () => {
	it('places each talent by tree and name, and lists the ones it cannot place', () => {
		const c = parseCharacterString(encode(CHARACTER_PREFIX, SKARR));
		const { talentsString, unknown } = talentsStringFor(Class.ClassWarrior, c.talents);
		// Improved Heroic Strike is Arms #1; Shield Specialization and Anticipation are Protection #2 and #4
		expect(talentsString).toBe('3--0505');
		expect(unknown).toEqual(['Not A Talent']);
	});
});

describe('planString', () => {
	it('writes the lines Site.lua reads, with macro bodies escaped', () => {
		const str = planString({
			name: 'Skarr',
			classToken: 'WARRIOR',
			from: 'site',
			talents: { 10: '01', 11: '02' },
			keys: [
				{
					key: '1',
					slot: 7,
					name: 'Attack',
					versions: [
						{
							level: 1,
							name: 'Attack',
							body: '#showtooltip\n/cast Heroic Strike',
						},
						{ level: 4, name: 'Attack', spell: 'Heroic Strike' },
					],
				},
			],
		});
		expect(str.startsWith(PLAN_PREFIX)).toBe(true);
		expect(decode(PLAN_PREFIX, str).split('\n')).toEqual([
			'for=Skarr',
			'class=WARRIOR',
			'from=site',
			'talents=10|01',
			'talents=11|02',
			'key=1|7|Attack',
			'v=1|Attack|body=#showtooltip\\n/cast Heroic Strike',
			'v=4|Attack|spell=Heroic Strike',
		]);
	});
});
