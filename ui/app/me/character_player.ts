import { type FmsCharacter, talentsStringFor } from '@features/fms/model/fms_strings';
import { boardEntries, type BoardEntry, mainTree, roleOfSpec } from '@features/group/model/boards';
import type { GroupRole } from '@features/group/model/roles';
import { Player as PlayerProto } from '@generated/proto/api';
import { ConsumesSpec, Profession, Race } from '@generated/proto/common';
import type { SpecDefinition } from '@sim/spec_config';

const specModules = import.meta.glob<{ default: SpecDefinition<any> }>('../../specs/*/*/spec.{ts,tsx}');

export interface CharacterPlayer {
	player: PlayerProto;
	entry: BoardEntry;
	/** The job the sims run the character in, and the jobs its class can do. */
	role: GroupRole;
	roles: GroupRole[];
	talentsString: string;
	/** Talents the export has and the sims' trees don't. */
	unknownTalents: string[];
	/** The export carried no gear the sim knows, so the preset's gear stands in. */
	presetGear: boolean;
}

// The job a build's main tree points to, until Mark picks another on the page.
const HEALING_TREES = new Set(['Discipline', 'Holy', 'Restoration']);
export const defaultRole = (tree: string): GroupRole => (HEALING_TREES.has(tree) ? 'healer' : tree === 'Protection' ? 'tank' : 'dps');

/**
 * A pasted character as a player the sims can run: the class's preset build for the job whose main tree
 * matches the character's talents, with the character's name, level, race, talents and gear laid over it.
 * Without a job, the main tree picks one.
 */
export async function characterPlayer(c: FmsCharacter, role?: GroupRole): Promise<CharacterPlayer> {
	const folder = `/specs/${c.classToken.toLowerCase()}/`;
	const defs = await Promise.all(
		Object.entries(specModules)
			.filter(([path]) => path.includes(folder))
			.map(([, load]) => load().then(m => m.default)),
	);
	const entries = defs.flatMap(boardEntries);
	if (!entries.length) throw new Error(`The sims have no ${c.classToken.toLowerCase()} build to start from.`);

	const { talentsString, unknown } = talentsStringFor(c.cls, c.talents);
	const tree = talentsString ? mainTree(entries[0].spec, talentsString) : '';
	const job = role ?? defaultRole(tree);
	const forJob = entries.filter(e => roleOfSpec(e.spec) === job);
	const pool = forJob.length ? forJob : entries;
	const entry = pool.find(e => tree && mainTree(e.spec, e.talents) === tree) ?? pool[0];

	const player = PlayerProto.clone(entry.player);
	player.name = c.name || player.name;
	player.level = c.level;
	if (c.race !== Race.RaceUnknown) player.race = c.race;
	player.talentsString = talentsString;
	// The export has no bags, professions or swap gear, and the preset's are a level-60 raider's.
	player.consumables = ConsumesSpec.create();
	player.profession1 = Profession.ProfessionUnknown;
	player.profession2 = Profession.ProfessionUnknown;
	player.enableItemSwap = false;
	player.itemSwap = undefined;
	const presetGear = !c.equipment.items.some(i => i.id);
	if (!presetGear) player.equipment = c.equipment;
	const roles = (['tank', 'healer', 'dps'] as const).filter(r => entries.some(e => roleOfSpec(e.spec) === r));
	return { player, entry, role: roleOfSpec(entry.spec), roles, talentsString, unknownTalents: unknown, presetGear };
}
