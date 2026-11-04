import { RaidSimRequest } from '@generated/proto/api';

/** One FMS macro key and when to press it, as the sims press it. */
export interface KeyUse {
	key: string;
	/** In words; empty for the key pressed the rest of the time. */
	when: string;
	main: boolean;
	/** A key the sims never press: one for when something happens, like a taunt. */
	extra: boolean;
	/** What the key's macro casts, in the order it tries them. */
	spells: string[] | null;
}

export interface DuoKeys {
	/** The trio character whose keys every character of the class uses. */
	from: string;
	level: number;
	keys: KeyUse[];
}

export interface OpenerCast {
	at: number;
	player: string;
	spell: string;
}

// The Go server answers on the page's own origin.
export async function fetchKeys(classToken: string, level: number): Promise<DuoKeys | undefined> {
	const response = await fetch(`/duoKeys?class=${encodeURIComponent(classToken)}&level=${level}`);
	if (response.status === 404) return undefined;
	if (!response.ok) throw new Error(`HTTP ${response.status} from /duoKeys`);
	return response.json();
}

export async function fetchOpener(request: RaidSimRequest): Promise<OpenerCast[]> {
	const response = await fetch('/opener', {
		method: 'POST',
		headers: { 'Content-Type': 'application/x-protobuf' },
		body: RaidSimRequest.toBinary(request) as BodyInit,
	});
	if (!response.ok) throw new Error(`HTTP ${response.status} from /opener`);
	const body: { casts: OpenerCast[] | null; error: string } = await response.json();
	if (body.error) throw new Error(body.error);
	return body.casts ?? [];
}

/**
 * The key a cast is pressed on: the mashed key when its macro casts it, else the first other key the sims
 * press that does, else a key kept for when it's needed.
 */
export function keyFor(keys: DuoKeys | undefined, spell: string): KeyUse | undefined {
	const on = (k: KeyUse) => (k.spells ?? []).includes(spell);
	return keys?.keys.find(k => k.main && on(k)) ?? keys?.keys.find(k => !k.extra && on(k)) ?? keys?.keys.find(on);
}
