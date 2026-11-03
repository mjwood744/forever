import { Player as PlayerProto } from '@generated/proto/api';
import type { APLRotation } from '@generated/proto/apl';
import { ConsumesSpec, EquipmentSpec, type Spec } from '@generated/proto/common';
import { getClassRaces } from '@sim/player/classes/capabilities';
import { PlayerSpecs } from '@sim/player/specs';
import { withSpec } from '@sim/proto/spec_functions';
import type { SpecDefinition } from '@sim/spec_config';
import { classTalentsConfig } from '@sim/talents/factory';

import type { GroupRole } from './roles';

/** One build on a tier board: a spec page's preset, ready to sim. */
export interface BoardEntry {
	key: string;
	/** The preset's name, or the spec's when it has one build. */
	label: string;
	spec: Spec;
	role: GroupRole;
	talents: string;
	/** The spec page has no gear preset for it, so it sims naked. */
	gearless: boolean;
	player: PlayerProto;
}

export const roleOfSpec = (spec: Spec): GroupRole => {
	const ps = PlayerSpecs.fromProto(spec);
	return ps.isTankSpec ? 'tank' : ps.isHealingSpec ? 'healer' : 'dps';
};

/** Points per tree, "0/20/31" for a three-tree talents string. */
export const talentSplit = (talents: string): string =>
	talents
		.split('-')
		.concat(['', '', ''])
		.slice(0, 3)
		.map(tree => [...tree].reduce((sum, c) => sum + (Number(c) || 0), 0))
		.join('/');

/** The talent tree a build spends most points in, by its name in the class's trees ("Fire"). */
export const mainTree = (spec: Spec, talents: string): string => {
	const points = talentSplit(talents).split('/').map(Number);
	const top = points.indexOf(Math.max(...points));
	return classTalentsConfig[PlayerSpecs.fromProto(spec).classID][top]?.name ?? '';
};

// A player in the spec's defaults with the build's gear, talents and rotation laid over them.
function presetPlayer(def: SpecDefinition<any>, gear?: EquipmentSpec, talents?: string, rotation?: APLRotation): PlayerProto {
	const ps = PlayerSpecs.fromProto(def.spec);
	const d = def.defaults;
	return withSpec(
		def.spec,
		PlayerProto.create({
			name: PlayerSpecs.getFullSpecName(ps),
			class: ps.classID,
			race: d.other?.race ?? getClassRaces(ps.classID)[0],
			equipment: EquipmentSpec.clone(gear ?? d.gear),
			consumables: ConsumesSpec.clone(d.consumables),
			buffs: d.individualBuffs,
			talentsString: talents ?? d.talents.talentsString,
			profession1: d.other?.profession1,
			profession2: d.other?.profession2,
			distanceFromTarget: d.other?.distanceFromTarget ?? 0,
			rotation,
		}),
		d.specOptions,
	);
}

/**
 * The builds a spec page offers: each preset build, else each talents preset (with the rotation
 * preset of the same name, if any), else the spec's defaults. Builds without talents are skipped, since
 * the board labels each entry by its talent split.
 */
export function boardEntries(def: SpecDefinition<any>): BoardEntry[] {
	const ps = PlayerSpecs.fromProto(def.spec);
	const role = roleOfSpec(def.spec);
	const specName = PlayerSpecs.getFullSpecName(ps);
	const entry = (label: string, gear?: EquipmentSpec, talents?: string, rotation?: APLRotation): BoardEntry => {
		const player = presetPlayer(def, gear, talents, rotation);
		player.name = label === specName ? specName : `${label} (${specName})`;
		const gearless = !player.equipment?.items.some(item => item.id);
		return { key: `${def.spec}:${label}`, label, spec: def.spec, role, talents: player.talentsString, gearless, player };
	};

	// The server runs priority lists only: a simple or automatic rotation is built from a live player on
	// the spec page. A build without its own list takes the rotation preset named like it or its group,
	// then the defaults', then the first the page offers.
	const apl = (rotation?: APLRotation) => (rotation?.priorityList.length ? rotation : undefined);
	const rotations = def.presets.rotations.map(r => ({ name: r.name, rotation: apl(r.rotation.rotation) })).filter(r => r.rotation);
	const rotationFor = (rotation: APLRotation | undefined, ...names: Array<string | undefined>) =>
		apl(rotation) ?? rotations.find(r => names.includes(r.name))?.rotation ?? apl(def.defaults.aplRotation) ?? rotations[0]?.rotation;

	const builds = (def.presets.builds ?? []).filter(b => b.talents);
	if (builds.length) {
		return builds.map(b =>
			entry(
				b.group ? `${b.name} ${b.group}` : b.name,
				b.gear?.gear,
				b.talents?.data.talentsString,
				rotationFor(b.rotation?.rotation.rotation, b.name, b.group),
			),
		);
	}
	const talents = def.presets.talents.filter(t => t.data.talentsString);
	if (talents.length) {
		return talents.map(t => entry(talents.length === 1 ? specName : t.name, undefined, t.data.talentsString, rotationFor(undefined, t.name)));
	}
	return [entry(specName, undefined, undefined, rotationFor(undefined))];
}

export type Tier = 'S' | 'A' | 'B' | 'C';

/** A build's tier by its share of the best build's figure. */
export const tierOf = (share: number): Tier => (share >= 0.97 ? 'S' : share >= 0.88 ? 'A' : share >= 0.78 ? 'B' : 'C');
