import { Player as PlayerProto } from '@generated/proto/api';
import type { Spec } from '@generated/proto/common';
import type { Env } from '@sim/state/env';

// Shared by every spec page and the group page, so it carries no spec prefix.
export const GROUP_ROSTER_STORAGE_KEY = '__groupRoster__';

export const MAX_GROUP_SIZE = 5;

/** A character saved from its spec page, ready to sim: rotation resolved, consumables as simmed. */
export interface RosterEntry {
	id: string;
	name: string;
	spec: Spec;
	/** `Player` as proto JSON. */
	player: unknown;
	savedAt: number;
}

type Storage = Env['storage'];

export function readRoster(storage: Storage): RosterEntry[] {
	try {
		const parsed: unknown = JSON.parse(storage.getItem(GROUP_ROSTER_STORAGE_KEY) ?? '[]');
		return Array.isArray(parsed) ? parsed.filter(isEntry) : [];
	} catch {
		return [];
	}
}

function isEntry(v: unknown): v is RosterEntry {
	const e = v as RosterEntry;
	return !!e && typeof e.id === 'string' && typeof e.name === 'string' && typeof e.spec === 'number' && !!e.player;
}

function writeRoster(storage: Storage, roster: RosterEntry[]) {
	storage.setItem(GROUP_ROSTER_STORAGE_KEY, JSON.stringify(roster));
}

/** Adds the character under the first free name of the form `<base>`, `<base> 2`, `<base> 3`... */
export function addToRoster(storage: Storage, base: string, spec: Spec, player: PlayerProto, now = Date.now()): RosterEntry {
	const roster = readRoster(storage);
	const taken = new Set(roster.map(e => e.name));
	let name = base;
	for (let n = 2; taken.has(name); n++) name = `${base} ${n}`;
	const entry: RosterEntry = { id: `${now.toString(36)}-${roster.length}`, name, spec, player: PlayerProto.toJson(player), savedAt: now };
	writeRoster(storage, [...roster, entry]);
	return entry;
}

export function renameInRoster(storage: Storage, id: string, name: string) {
	writeRoster(
		storage,
		readRoster(storage).map(e => (e.id === id ? { ...e, name } : e)),
	);
}

export function removeFromRoster(storage: Storage, id: string) {
	writeRoster(
		storage,
		readRoster(storage).filter(e => e.id !== id),
	);
}

/** The entry's player, named after the entry so the results name it too. */
export function rosterPlayer(entry: RosterEntry): PlayerProto {
	const player = PlayerProto.fromJson(entry.player as never, { ignoreUnknownFields: true });
	player.name = entry.name;
	return player;
}
