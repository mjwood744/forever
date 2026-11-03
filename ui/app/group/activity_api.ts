import {
	Activity,
	type ActivityDungeonInfo,
	ActivityDungeonsRequest,
	ActivityDungeonsResult,
	ActivitySimRequest,
	ActivitySimResult,
	Party,
	type Player,
	Raid,
	SimOptions,
} from '@generated/proto/api';
import { Debuffs, PartyBuffs, RaidBuffs } from '@generated/proto/buffs';
import { UnitReference, UnitReference_Type } from '@generated/proto/common';

// The Go server answers these on the page's own origin; the group page has no wasm fallback.
async function post(endpoint: string, body: Uint8Array): Promise<Uint8Array> {
	const response = await fetch(`/${endpoint}`, { method: 'POST', headers: { 'Content-Type': 'application/x-protobuf' }, body: body as BodyInit });
	if (!response.ok) throw new Error(`HTTP ${response.status} from /${endpoint}`);
	return new Uint8Array(await response.arrayBuffer());
}

export async function fetchDungeons(level: number): Promise<ActivityDungeonInfo[]> {
	const bytes = await post('activityDungeons', ActivityDungeonsRequest.toBinary(ActivityDungeonsRequest.create({ level })));
	return ActivityDungeonsResult.fromBinary(bytes).dungeons;
}

export interface GroupRun {
	players: Player[];
	/** Index into `players`. */
	tank: number;
	activity: Activity;
	level: number;
	dungeon: string;
	iterations: number;
}

export async function runActivity(run: GroupRun): Promise<ActivitySimResult> {
	const raid = Raid.create({
		parties: [Party.create({ players: run.players, buffs: PartyBuffs.create() })],
		buffs: RaidBuffs.create(),
		debuffs: Debuffs.create(),
		tanks: [UnitReference.create({ type: UnitReference_Type.Player, index: run.tank })],
	});
	const request = ActivitySimRequest.create({
		raid,
		activity: run.activity,
		level: run.level,
		dungeon: run.activity === Activity.ActivityDungeon ? run.dungeon : '',
		simOptions: SimOptions.create({ iterations: run.iterations, randomSeed: BigInt(Math.floor(Math.random() * 2 ** 31)) }),
	});
	return ActivitySimResult.fromBinary(await post('activitySim', ActivitySimRequest.toBinary(request)));
}
