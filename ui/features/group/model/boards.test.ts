import { Spec } from '@generated/proto/common';
import { describe, expect, it } from 'vitest';

import mage from '../../../specs/mage/dps/spec';
import warlock from '../../../specs/warlock/dps/spec';
import warrior from '../../../specs/warrior/dps/spec';
import { boardEntries, mainTree, talentSplit, tierOf } from './boards';

describe('tier boards', () => {
	it('counts the points in each tree', () => {
		expect(talentSplit('0002052011003--053500030100331251')).toBe('14/0/32');
		expect(talentSplit('50003-0530313321301531-5022')).toBe('8/34/9');
	});

	it('tiers a build by its share of the best', () => {
		expect([1, 0.97, 0.9, 0.8, 0.5].map(tierOf)).toEqual(['S', 'S', 'A', 'B', 'C']);
	});

	it('takes a spec page preset builds', () => {
		const entries = boardEntries(warlock);
		expect(entries.length).toBeGreaterThan(1);
		for (const e of entries) {
			expect(e.spec).toBe(Spec.SpecWarlock);
			expect(e.role).toBe('dps');
			expect(e.player.talentsString).toBe(e.talents);
			expect(e.player.rotation?.priorityList.length).toBeGreaterThan(0);
			// The warlock page ships no gear preset.
			expect(e.gearless).toBe(true);
		}
	});

	it('pairs a talents preset with the rotation of the same name', () => {
		const fire = boardEntries(mage).find(e => e.label === 'Fire');
		expect(fire?.talents).toBe('0002052001001-35500001230135031-003');
		expect(fire?.player.rotation?.priorityList.length).toBeGreaterThan(0);
		expect(fire?.gearless).toBe(false);
	});

	it('gives a build with a simple rotation the priority list named like its group', () => {
		const builds = boardEntries(warrior);
		expect(builds.map(e => e.label)).toEqual(['Pre-BiS Fury', 'Pre-BiS Arms']);
		const [fury, arms] = builds.map(e => e.player.rotation);
		expect(fury?.priorityList.length).toBeGreaterThan(0);
		expect(arms?.priorityList.length).toBeGreaterThan(0);
		expect(fury).not.toEqual(arms);
	});

	it('names the tree a build spends most in', () => {
		expect(mainTree(Spec.SpecMage, '0002052001001-35500001230135031-003')).toBe('Fire');
		expect(mainTree(Spec.SpecMage, '0002052011003--053500030100331251')).toBe('Frost');
	});
});
