import { goalFor, goalUnit, searchRotation } from '@features/apl/model/rotation_search';
import { type FmsCharacter, parseCharacterString } from '@features/fms/model/fms_strings';
import { talentSplit } from '@features/group/model/boards';
import type { GroupRole } from '@features/group/model/roles';
import { MAX_GROUP_SIZE } from '@features/group/model/roster';
import {
	Activity,
	type ActivityDungeonInfo,
	type ActivityPlayerSummary,
	Party,
	Raid,
	RaidSimRequest,
	type RotationSearchResult,
	type RotationStep,
	SimOptions,
	SimType,
} from '@generated/proto/api';
import { Debuffs, PartyBuffs, RaidBuffs } from '@generated/proto/buffs';
import { UnitReference, UnitReference_Type } from '@generated/proto/common';
import i18n from '@i18n/config';
import { PlayerClasses } from '@sim/player/classes/index';
import { PlayerSpecs } from '@sim/player/specs';
import { Encounter } from '@sim/raid/encounter';
import { Button } from '@ui-kit/Button';
import { ChoiceField } from '@ui-kit/ChoiceField';
import { Notice } from '@ui-kit/Notice';
import { StringField } from '@ui-kit/TextField';
import { ToggleGroup } from '@ui-kit/ToggleGroup';
import { useEffect, useState } from 'react';

import { ACTIVITIES, ACTIVITY_KEYS } from '../group/activities';
import { fetchDungeons, runActivity, withDatabase } from '../group/activity_api';
import { type CharacterPlayer, characterPlayer } from './character_player';
import { type DuoKeys, fetchKeys, fetchOpener, type OpenerCast } from './keys_api';
import { Opener, YourKeys } from './KeysView';

// The characters pasted so far, by name, so a reload or a second visit keeps them.
const STORAGE_KEY = '__meCharacters__';
const ITERATIONS = 500;

interface Saved {
	source: string;
	/** The job Mark picked; without one the talents pick it. */
	role?: GroupRole;
}

const readSaved = (): Saved[] => {
	try {
		const v: unknown = JSON.parse(window.localStorage.getItem(STORAGE_KEY) ?? '[]');
		return Array.isArray(v) ? v.filter((s): s is Saved => !!s && typeof (s as Saved).source === 'string') : [];
	} catch {
		return [];
	}
};
const writeSaved = (loaded: Loaded[]) => {
	try {
		window.localStorage.setItem(STORAGE_KEY, JSON.stringify(loaded.map(({ source, role }) => ({ source, role }))));
	} catch {
		// a private window: the characters last until the tab closes
	}
};

interface Loaded extends Saved {
	character: FmsCharacter;
	built: CharacterPlayer;
}

interface Ran {
	summary?: ActivityPlayerSummary;
	/** The other characters it ran with. */
	with: string[];
	rotation?: RotationSearchResult;
	/** The first seconds of the pull, every character's casts. */
	opener?: OpenerCast[];
	warnings: string[];
}

const Steps = ({ steps }: { steps: RotationStep[] }) => (
	<ol className="mb-0 flex flex-col gap-1 pl-8">
		{steps.map((step, i) => (
			<li key={i}>
				<span className="font-bold">{step.ability}</span>
				{step.when && <span className="block text-sm text-gray-400">{i18n.t('me.rotation.when', { when: step.when })}</span>}
				{step.steps.length > 0 && <Steps steps={step.steps} />}
			</li>
		))}
	</ol>
);

const Stat = ({ label, value, note }: { label: string; value: string; note?: string }) => (
	<div className="flex min-w-40 flex-1 flex-col rounded-lg border border-gray-700 bg-gray-900 p-3">
		<span className="text-sm text-gray-400">{label}</span>
		<span className="text-2xl font-bold">{value}</span>
		{note && <span className="text-sm text-green-400">{note}</span>}
	</div>
);

const fmt = (v: number, digits = 0) => v.toLocaleString(undefined, { maximumFractionDigits: digits, minimumFractionDigits: digits });

export const MePage = () => {
	const [loaded, setLoaded] = useState<Loaded[]>([]);
	const [current, setCurrent] = useState(0);
	const [activity, setActivity] = useState(Activity.ActivityDungeon);
	const [dungeons, setDungeons] = useState<ActivityDungeonInfo[]>([]);
	const [dungeon, setDungeon] = useState('');
	const [running, setRunning] = useState(false);
	const [error, setError] = useState('');
	const [ran, setRan] = useState<Record<string, Ran>>({});
	const [keys, setKeys] = useState<Record<string, DuoKeys | undefined>>({});

	const load = async (saved: Saved[]) => {
		const out: Loaded[] = [];
		for (const { source, role } of saved) {
			const character = parseCharacterString(source);
			out.push({ source, role, character, built: await characterPlayer(character, role) });
		}
		return out;
	};
	const keyOf = (l: Loaded) => `${l.source}:${l.built.role}`;

	useEffect(() => {
		load(readSaved())
			.then(setLoaded)
			.catch(() => writeSaved([]));
	}, []);

	useEffect(() => {
		for (const l of loaded) {
			const id = `${l.character.classToken}:${l.character.level}`;
			if (id in keys) continue;
			fetchKeys(l.character.classToken, l.character.level)
				.then(k => setKeys(old => ({ ...old, [id]: k })))
				.catch(e => setError(String(e)));
		}
	}, [loaded, keys]);
	const keysOf = (l: Loaded) => keys[`${l.character.classToken}:${l.character.level}`];

	const here = loaded[current];
	const level = here?.character.level ?? 60;

	useEffect(() => {
		fetchDungeons(level)
			.then(ds => {
				setDungeons(ds);
				setDungeon(d => (ds.some(x => x.name === d) ? d : (ds[0]?.name ?? '')));
			})
			.catch(e => setError(String(e)));
	}, [level]);

	// The field reports a paste on Enter or when it loses focus.
	const add = async (source: string) => {
		if (!source.trim()) return;
		setError('');
		try {
			const [next] = await load([{ source }]);
			// the same character pasted again replaces the old one
			const rest = loaded.filter(l => l.character.name !== next.character.name || l.character.realm !== next.character.realm);
			const all = [...rest, next];
			setLoaded(all);
			setCurrent(all.length - 1);
			writeSaved(all);
		} catch (e) {
			setError(e instanceof Error ? e.message : String(e));
		}
	};

	const remove = (i: number) => {
		const all = loaded.filter((_, j) => j !== i);
		setLoaded(all);
		setCurrent(0);
		writeSaved(all);
	};

	const pickRole = async (role: GroupRole) => {
		if (!here) return;
		const [next] = await load([{ source: here.source, role }]);
		const all = loaded.map((l, i) => (i === current ? next : l));
		setLoaded(all);
		writeSaved(all);
	};

	// Every pasted character runs together, as the duo plays: the first tank tanks, and a healer heals them.
	// The rotation is searched for the character on show, with the others beside it.
	const run = async () => {
		if (!here) return;
		const group = [here, ...loaded.filter(l => l !== here)].slice(0, MAX_GROUP_SIZE);
		if (group.every(l => l.built.role === 'healer')) {
			setError(i18n.t('me.needs_group'));
			return;
		}
		setRunning(true);
		setError('');
		const players = group.map(l => l.built.player);
		const tank = group.findIndex(l => l.built.role === 'tank');
		const goal = goalFor(PlayerSpecs.fromProto(here.built.entry.spec));
		try {
			const withDb = await withDatabase(players);
			const request = (searched?: RotationSearchResult) =>
				RaidSimRequest.create({
					raid: Raid.create({
						parties: [
							Party.create({
								// the character on show plays the priority list the search found
								players: searched?.rotation ? [{ ...withDb[0], rotation: searched.rotation }, ...withDb.slice(1)] : withDb,
								buffs: PartyBuffs.create(),
							}),
						],
						buffs: RaidBuffs.create(),
						debuffs: Debuffs.create(),
						tanks: tank >= 0 ? [UnitReference.create({ type: UnitReference_Type.Player, index: tank })] : [],
					}),
					encounter: Encounter.defaultEncounterProto(),
					simOptions: SimOptions.create({ iterations: 300, randomSeed: BigInt(Math.floor(Math.random() * 2 ** 31)) }),
					type: SimType.SimTypeRaid,
				});
			const [activityResult, rotation] = await Promise.all([
				runActivity({ players, tank: Math.max(0, tank), activity, level, dungeon, iterations: ITERATIONS, untanked: tank < 0 }),
				// The sims heal with a built-in plan per healer, which has no priority list to search.
				here.built.role === 'healer' ? Promise.resolve(undefined) : searchRotation(request(), goal),
			]);
			const opener = await fetchOpener(request(rotation?.error ? undefined : rotation)).catch((e: unknown) => {
				setError(String(e));
				return undefined;
			});
			if (activityResult.error) setError(activityResult.error.message);
			if (rotation?.error) setError(rotation.error.message);
			setRan(r => {
				const next = { ...r };
				group.forEach((l, i) => {
					const old = r[keyOf(l)];
					next[keyOf(l)] = {
						summary: activityResult.players[i],
						rotation: l === here ? (rotation?.error ? undefined : rotation) : old?.rotation,
						opener: l === here ? opener : old?.opener,
						warnings: activityResult.warnings,
						with: group.filter(o => o !== l).map(o => o.character.name),
					};
				});
				return next;
			});
		} catch (e) {
			setError(String(e));
		} finally {
			setRunning(false);
		}
	};

	const result = here ? ran[keyOf(here)] : undefined;
	const role = here?.built.role ?? 'dps';
	const s = result?.summary;
	const rot = result?.rotation;
	const unit = here ? goalUnit(goalFor(PlayerSpecs.fromProto(here.built.entry.spec))) : '';
	const gain = rot && rot.startScore > 0 && rot.score > rot.startScore * 1.001 ? ((rot.score / rot.startScore - 1) * 100).toFixed(1) : '';

	return (
		<div className="mx-auto flex max-w-6xl flex-col gap-6 p-6 text-gray-200">
			<header className="flex flex-wrap items-end gap-4">
				<h1 className="mb-0 text-3xl font-bold">{i18n.t('me.title')}</h1>
				<StringField className="min-w-80 flex-1" value="" placeholder={i18n.t('me.paste')} onChange={add} />
			</header>

			{!loaded.length && <Notice level="info">{i18n.t('me.empty')}</Notice>}

			{loaded.length > 0 && (
				<div className="flex flex-wrap items-center gap-3">
					{loaded.map((l, i) => {
						const ps = PlayerSpecs.fromProto(l.built.entry.spec);
						const cls = PlayerClasses.fromProto(ps.classID);
						const on = i === current;
						return (
							<button
								key={l.source}
								type="button"
								onClick={() => setCurrent(i)}
								className={`flex items-center gap-3 rounded-lg border px-3 py-2 text-left ${on ? 'border-blue-400' : 'border-gray-700'} bg-gray-900`}>
								<img src={ps.getIcon('small')} alt="" className="rounded size-8" />
								<span>
									<span className="block font-bold">{l.character.name}</span>
									<span className="block text-sm text-gray-400">
										{i18n.t('me.card', {
											cls: cls.friendlyName,
											level: l.character.level,
											role: l.character.role ? i18n.t(`me.roles.${l.character.role}`) : '',
											split: talentSplit(l.built.talentsString) || '0/0/0',
										})}
									</span>
								</span>
							</button>
						);
					})}
					<div className="ml-auto flex flex-wrap items-end gap-3">
						{here && here.built.roles.length > 1 && (
							<ToggleGroup
								aria-label={i18n.t('me.job')}
								value={here.built.role}
								onChange={pickRole}
								options={here.built.roles.map(r => ({ value: r, label: i18n.t(`group.roles.${r}`) }))}
							/>
						)}
						<ChoiceField
							className="min-w-40"
							label={i18n.t('me.activity')}
							value={activity}
							options={ACTIVITIES.map(a => ({ value: a, label: i18n.t(`group.activities.${ACTIVITY_KEYS[a]}`) }))}
							onChange={setActivity}
						/>
						{activity === Activity.ActivityDungeon && (
							<ChoiceField
								className="min-w-64"
								label={i18n.t('group.settings.dungeon')}
								value={dungeon}
								placeholder={i18n.t('group.settings.no_dungeon')}
								options={dungeons.map(d => ({
									value: d.name,
									label: i18n.t('group.settings.dungeon_option', { name: d.name, level: d.lastBossLevel, pulls: d.pulls }),
								}))}
								onChange={setDungeon}
							/>
						)}
						<Button
							variant="primary"
							loading={running}
							disabled={running || !here || (activity === Activity.ActivityDungeon && !dungeon)}
							onClick={run}>
							{i18n.t('me.run')}
						</Button>
					</div>
				</div>
			)}

			{error && <Notice level="error">{error}</Notice>}
			{here && here.built.unknownTalents.length > 0 && (
				<Notice level="warning">{i18n.t('me.unknown_talents', { names: here.built.unknownTalents.join(', ') })}</Notice>
			)}
			{here && here.built.presetGear && <Notice level="warning">{i18n.t('me.preset_gear')}</Notice>}
			{result?.warnings.map(w => (
				<Notice key={w} level="warning">
					{w}
				</Notice>
			))}

			{here && result && (
				<p className="mb-0 text-sm text-gray-400">
					{result.with.length ? i18n.t('me.ran_with', { names: result.with.join(', ') }) : i18n.t('me.ran_alone')}
				</p>
			)}
			{here && s && (
				<div className="flex flex-wrap gap-3">
					{role === 'tank' && <Stat label={i18n.t('me.stats.tps')} value={fmt(s.tps)} />}
					{role === 'healer' && <Stat label={i18n.t('me.stats.hps')} value={fmt(s.hps)} />}
					{role !== 'healer' && <Stat label={i18n.t('me.stats.dps')} value={fmt(s.dps)} />}
					{role === 'tank' && <Stat label={i18n.t('me.stats.dtps')} value={fmt(s.dtps)} />}
					{role === 'healer' && <Stat label={i18n.t('me.stats.oom')} value={fmt(s.secondsOom)} />}
					<Stat label={i18n.t('me.stats.death')} value={`${fmt(s.chanceOfDeath * 100, 1)}%`} />
				</div>
			)}

			{loaded.length > 0 && (
				<YourKeys
					characters={[...(here ? [here] : []), ...loaded.filter(l => l !== here)].map(l => ({
						name: l.character.name,
						detail: i18n.t('me.keys.detail', {
							role: i18n.t(`group.roles.${l.built.role}`),
							from: keysOf(l)?.from ?? '',
							level: l.character.level,
						}),
						keys: keysOf(l),
					}))}
				/>
			)}

			{here && result?.opener && (
				<Opener
					casts={result.opener}
					keysOf={player => {
						const l = loaded.find(x => x.character.name === player);
						return l && keysOf(l);
					}}
				/>
			)}

			{here && rot && (
				<details className="rounded-lg border border-gray-700 bg-gray-900 p-4">
					<summary className="cursor-pointer text-sm text-gray-400">{i18n.t('me.rotation.title')}</summary>
					<p className="mt-2 mb-3 text-sm text-gray-400">{gain ? i18n.t('me.rotation.gain', { gain, unit }) : i18n.t('me.rotation.same')}</p>
					{rot.prepull.length > 0 && (
						<>
							<p className="mb-1 font-bold">{i18n.t('me.rotation.prepull')}</p>
							<Steps steps={rot.prepull} />
						</>
					)}
					<Steps steps={rot.steps} />
				</details>
			)}

			{loaded.length > 0 && (
				<div className="flex justify-end">
					<Button variant="link-danger" size="sm" onClick={() => remove(current)}>
						{i18n.t('me.remove', { name: here?.character.name ?? '' })}
					</Button>
				</div>
			)}
		</div>
	);
};
