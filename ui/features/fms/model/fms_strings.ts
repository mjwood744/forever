import { Class, EquipmentSpec, ItemSlot, ItemSpec, Race } from '@generated/proto/common';
import { classTalentsConfig } from '@sim/talents/factory';

// The strings Mark's Forever Multibox Setup addon trades with this site. `/fms export` gives a character
// string, `/fms import` takes a plan string. Each is a prefix and then base64 of `key=value` lines, with
// fields split by `|` and newlines and backslashes escaped in macro bodies. FMS's Site.lua is the other half.
export const CHARACTER_PREFIX = '!FMSCHAR1!';
export const PLAN_PREFIX = '!FMSPLAN1!';

export interface FmsTalent {
	tree: string;
	name: string;
	rank: number;
}

export interface FmsCharacter {
	name: string;
	realm: string;
	/** The game's class token, `WARRIOR`. */
	classToken: string;
	cls: Class;
	race: Race;
	level: number;
	role: 'main' | 'follower' | '';
	talents: FmsTalent[];
	equipment: EquipmentSpec;
}

const CLASS_TOKENS: Record<string, Class> = {
	DRUID: Class.ClassDruid,
	HUNTER: Class.ClassHunter,
	MAGE: Class.ClassMage,
	PALADIN: Class.ClassPaladin,
	PRIEST: Class.ClassPriest,
	ROGUE: Class.ClassRogue,
	SHAMAN: Class.ClassShaman,
	WARLOCK: Class.ClassWarlock,
	WARRIOR: Class.ClassWarrior,
};

// The game's race file names, as the second value of UnitRace.
const RACE_TOKENS: Record<string, Race> = {
	BloodElf: Race.RaceBloodElf,
	Dwarf: Race.RaceDwarf,
	Gnome: Race.RaceGnome,
	Human: Race.RaceHuman,
	NightElf: Race.RaceNightElf,
	Orc: Race.RaceOrc,
	Tauren: Race.RaceTauren,
	Troll: Race.RaceTroll,
	Scourge: Race.RaceUndead,
	Undead: Race.RaceUndead,
};

// The game's inventory slot numbers. The shirt (4) and tabard (19) have no sim slot.
const INVENTORY_SLOTS: Record<number, ItemSlot> = {
	1: ItemSlot.ItemSlotHead,
	2: ItemSlot.ItemSlotNeck,
	3: ItemSlot.ItemSlotShoulder,
	5: ItemSlot.ItemSlotChest,
	6: ItemSlot.ItemSlotWaist,
	7: ItemSlot.ItemSlotLegs,
	8: ItemSlot.ItemSlotFeet,
	9: ItemSlot.ItemSlotWrist,
	10: ItemSlot.ItemSlotHands,
	11: ItemSlot.ItemSlotFinger1,
	12: ItemSlot.ItemSlotFinger2,
	13: ItemSlot.ItemSlotTrinket1,
	14: ItemSlot.ItemSlotTrinket2,
	15: ItemSlot.ItemSlotBack,
	16: ItemSlot.ItemSlotMainHand,
	17: ItemSlot.ItemSlotOffHand,
	18: ItemSlot.ItemSlotRanged,
};

const toBase64 = (text: string) => btoa(String.fromCharCode(...new TextEncoder().encode(text)));
const fromBase64 = (b64: string) => new TextDecoder().decode(Uint8Array.from(atob(b64), c => c.charCodeAt(0)));

const escape = (s: string) => s.replace(/\\/g, '\\\\').replace(/\r/g, '').replace(/\n/g, '\\n');

/** Reads a `/fms export` string; throws with a sentence for Mark when it isn't one. */
export function parseCharacterString(input: string): FmsCharacter {
	const str = input.replace(/\s/g, '');
	if (str.startsWith(PLAN_PREFIX)) throw new Error('That is a plan string for /fms import. Paste the one /fms export gives you.');
	if (!str.startsWith(CHARACTER_PREFIX)) throw new Error(`That is not an /fms export string (they start with ${CHARACTER_PREFIX}).`);
	let text: string;
	try {
		text = fromBase64(str.slice(CHARACTER_PREFIX.length));
	} catch {
		throw new Error('That string is damaged or cut short. Copy the whole string again.');
	}

	const character: FmsCharacter = {
		name: '',
		realm: '',
		classToken: '',
		cls: Class.ClassUnknown,
		race: Race.RaceUnknown,
		level: 60,
		role: '',
		talents: [],
		equipment: EquipmentSpec.create({
			items: Array.from({ length: ItemSlot.ItemSlotRanged + 1 }, () => ItemSpec.create()),
		}),
	};
	for (const line of text.split('\n')) {
		const eq = line.indexOf('=');
		if (eq < 0) continue;
		const key = line.slice(0, eq);
		const value = line.slice(eq + 1);
		const fields = value.split('|');
		switch (key) {
			case 'name':
				character.name = value;
				break;
			case 'realm':
				character.realm = value;
				break;
			case 'class':
				character.classToken = value;
				character.cls = CLASS_TOKENS[value] ?? Class.ClassUnknown;
				break;
			case 'race':
				character.race = RACE_TOKENS[value] ?? Race.RaceUnknown;
				break;
			case 'level':
				character.level = Number(value) || 60;
				break;
			case 'role':
				character.role = value === 'main' || value === 'follower' ? value : '';
				break;
			case 'talent':
				if (fields.length >= 3)
					character.talents.push({
						tree: fields[0],
						name: fields[1],
						rank: Number(fields[2]) || 0,
					});
				break;
			case 'gear': {
				const slot = INVENTORY_SLOTS[Number(fields[0])];
				if (slot !== undefined)
					character.equipment.items[slot] = ItemSpec.create({
						id: Number(fields[1]) || 0,
						enchant: Number(fields[2]) || 0,
					});
				break;
			}
		}
	}
	if (character.cls === Class.ClassUnknown) throw new Error('The string has no class the sims know. Copy the whole string again.');
	return character;
}

/**
 * The talents string for a character's talents, by tree and talent name. Talents the sims' trees don't
 * have come back in `unknown`.
 */
export function talentsStringFor(cls: Class, talents: FmsTalent[]): { talentsString: string; unknown: string[] } {
	const trees = classTalentsConfig[cls];
	const unknown: string[] = [];
	const ranks = trees.map(tree => tree.talents.map(() => 0));
	for (const t of talents) {
		// Prefer the named tree; a talent name is unique within a class, so fall back to any tree.
		const byTree = trees.findIndex(tree => tree.name === t.tree);
		const order = byTree >= 0 ? [byTree, ...trees.keys()] : [...trees.keys()];
		const hit = order
			.map(treeIdx => ({
				treeIdx,
				talentIdx: trees[treeIdx].talents.findIndex(c => c.fancyName === t.name),
			}))
			.find(h => h.talentIdx >= 0);
		if (hit) ranks[hit.treeIdx][hit.talentIdx] = Math.min(t.rank, trees[hit.treeIdx].talents[hit.talentIdx].maxPoints);
		else unknown.push(t.name);
	}
	const talentsString = ranks
		.map(tree => tree.join('').replace(/0+$/, ''))
		.join('-')
		.replace(/-+$/, '');
	return { talentsString, unknown };
}

export interface PlanKeyVersion {
	level: number;
	name: string;
	/** One spell on the key, the spell itself rather than a macro. */
	spell?: string;
	/** A pet spell, kept as a macro since pet spells can't go on a bar from the spellbook. */
	pet?: string;
	body?: string;
}

export interface PlanKey {
	/** The key as FMS names it: `1`, `Shift+3`, `E`. */
	key: string;
	/** The action bar slot the key presses. */
	slot: number;
	name: string;
	versions: PlanKeyVersion[];
}

export interface FmsPlan {
	name: string;
	classToken: string;
	from: string;
	/** Talents string by character level. */
	talents?: Record<number, string>;
	keys?: PlanKey[];
}

/** The string `/fms import` reads. */
export function planString(plan: FmsPlan): string {
	const lines = [`for=${plan.name}`, `class=${plan.classToken}`, `from=${plan.from}`];
	for (const [level, talents] of Object.entries(plan.talents ?? {})) lines.push(`talents=${level}|${talents}`);
	for (const k of plan.keys ?? []) {
		lines.push(`key=${k.key}|${k.slot}|${k.name}`);
		for (const v of k.versions) {
			const what = v.spell ? `spell=${v.spell}` : v.pet ? `pet=${v.pet}` : `body=${escape(v.body ?? '')}`;
			lines.push(`v=${v.level}|${v.name}|${what}`);
		}
	}
	return PLAN_PREFIX + toBase64(lines.join('\n'));
}
