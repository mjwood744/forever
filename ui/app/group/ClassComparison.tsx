import { boardEntries, type BoardEntry, talentSplit, tierOf } from '@features/group/model/boards';
import type { GroupRole } from '@features/group/model/roles';
import { Activity, type ActivityDungeonInfo, type ActivityPlayerSummary } from '@generated/proto/api';
import { Spec } from '@generated/proto/common';
import i18n from '@i18n/config';
import { PlayerClasses } from '@sim/player/classes/index';
import { PlayerSpecs } from '@sim/player/specs';
import type { SpecDefinition } from '@sim/spec_config';
import { Button } from '@ui-kit/Button';
import { ChoiceField } from '@ui-kit/ChoiceField';
import { Notice } from '@ui-kit/Notice';
import { NumberField } from '@ui-kit/TextField';
import { ToggleGroup } from '@ui-kit/ToggleGroup';
import { useEffect, useMemo, useState } from 'react';

import { ACTIVITIES, ACTIVITY_KEYS } from './activities';
import { fetchDungeons, runActivity } from './activity_api';
import { TierBoard, type TierRow } from './TierBoard';

const specModules = import.meta.glob<{ default: SpecDefinition<any> }>('../../specs/*/*/spec.{ts,tsx}');

// The tank every healer heals on the healer board.
const REFERENCE_TANK = Spec.SpecProtectionWarrior;
const CONCURRENCY = 4;
const LAST_RUN_STORAGE_KEY = '__groupBoardsLastRun__';

type Results = Record<string, ActivityPlayerSummary>;
type Warnings = Record<string, string[]>;

const readLastRun = (key: string): Record<string, number> => {
	try {
		return JSON.parse(window.localStorage.getItem(`${LAST_RUN_STORAGE_KEY}${key}`) ?? '{}');
	} catch {
		return {};
	}
};

async function inPool<T>(items: ReadonlyArray<T>, run: (item: T) => Promise<void>) {
	let next = 0;
	const worker = async () => {
		while (next < items.length) await run(items[next++]);
	};
	await Promise.all(Array.from({ length: CONCURRENCY }, worker));
}

export const ClassComparison = () => {
	const [entries, setEntries] = useState<BoardEntry[]>([]);
	const [activity, setActivity] = useState(Activity.ActivityRaid);
	const [level, setLevel] = useState(60);
	const [dungeons, setDungeons] = useState<ActivityDungeonInfo[]>([]);
	const [dungeon, setDungeon] = useState('');
	const [iterations, setIterations] = useState(200);
	const [tankRank, setTankRank] = useState<'tps' | 'dps'>('tps');
	const [results, setResults] = useState<Results>({});
	const [lastRun, setLastRun] = useState<Record<string, number>>({});
	const [progress, setProgress] = useState<{ done: number; total: number } | null>(null);
	const [error, setError] = useState('');
	const [warnings, setWarnings] = useState<Warnings>({});
	const [failures, setFailures] = useState<string[]>([]);

	useEffect(() => {
		Promise.all(Object.values(specModules).map(load => load()))
			.then(mods => setEntries(mods.flatMap(m => boardEntries(m.default))))
			.catch(e => setError(String(e)));
	}, []);

	useEffect(() => {
		fetchDungeons(level)
			.then(ds => {
				setDungeons(ds);
				setDungeon(current => (ds.some(d => d.name === current) ? current : (ds[0]?.name ?? '')));
			})
			.catch(e => setError(String(e)));
	}, [level]);

	const runKey = `${activity}:${level}:${activity === Activity.ActivityDungeon ? dungeon : ''}`;

	const run = async () => {
		const tank = entries.find(e => e.spec === REFERENCE_TANK);
		setError('');
		setResults({});
		setWarnings({});
		setFailures([]);
		setLastRun(readLastRun(runKey));
		setProgress({ done: 0, total: entries.length });
		const next: Results = {};
		const nextWarnings: Warnings = {};
		const failed: string[] = [];
		try {
			// A build that fails is listed under the boards; the rest still rank.
			await inPool(entries, async entry => {
				const healer = entry.role === 'healer' && tank;
				try {
					const result = await runActivity({
						players: healer ? [tank.player, entry.player] : [entry.player],
						tank: 0,
						untanked: entry.role === 'dps',
						activity,
						level,
						dungeon,
						iterations,
					});
					if (result.error) throw new Error(result.error.message);
					next[entry.key] = result.players[healer ? 1 : 0];
					// The healer board's reference tank warns too; only the build's own warnings count.
					const own = result.warnings.filter(w => w.startsWith(`${entry.player.name}:`));
					if (own.length) nextWarnings[entry.key] = own;
					setResults({ ...next });
					setWarnings({ ...nextWarnings });
				} catch (e) {
					failed.push(`${entry.label}: ${e instanceof Error ? e.message : String(e)}`);
					setFailures([...failed]);
				}
				setProgress(p => p && { ...p, done: p.done + 1 });
			});
			const saved: Record<string, number> = {};
			for (const e of entries) if (next[e.key]) saved[e.key] = valueOf(e.role, next[e.key]);
			window.localStorage.setItem(`${LAST_RUN_STORAGE_KEY}${runKey}`, JSON.stringify(saved));
		} catch (e) {
			setError(String(e));
		} finally {
			setProgress(null);
		}
	};

	const valueOf = (role: GroupRole, r?: ActivityPlayerSummary) =>
		!r ? 0 : role === 'healer' ? r.hps : role === 'tank' ? (tankRank === 'tps' ? r.tps : r.dps) : r.dps;

	const rowsFor = (role: GroupRole): TierRow[] => {
		// Each class shows only its highest-simming build for the role.
		const topByClass = new Map<number, BoardEntry>();
		for (const e of entries.filter(e => e.role === role && results[e.key])) {
			const cls = PlayerSpecs.fromProto(e.spec).classID;
			const held = topByClass.get(cls);
			if (!held || valueOf(role, results[e.key]) > valueOf(role, results[held.key])) topByClass.set(cls, e);
		}
		const done = [...topByClass.values()];
		const best = Math.max(0, ...done.map(e => valueOf(role, results[e.key])));
		return done.map(e => {
			const r = results[e.key];
			const value = valueOf(role, r);
			const ps = PlayerSpecs.fromProto(e.spec);
			const cls = PlayerClasses.fromProto(ps.classID);
			const share = best > 0 ? value / best : 0;
			const last = lastRun[e.key];
			const tags = [...(e.gearless ? [i18n.t('group.boards.no_gear')] : []), ...(warnings[e.key] ? [i18n.t('group.boards.gear_incomplete')] : [])];
			return {
				key: e.key,
				tier: tierOf(share),
				label: e.label,
				tags,
				classColor: cls.friendlyName.toLowerCase(),
				subtitle: `${PlayerSpecs.getFullSpecName(ps)} · ${talentSplit(e.talents)}`,
				icon: ps.getIcon('medium'),
				value,
				unit: role === 'healer' ? 'HPS' : role === 'tank' && tankRank === 'tps' ? 'TPS' : 'DPS',
				secondary: role === 'tank' ? (tankRank === 'tps' ? { value: r.dps, unit: 'DPS' } : { value: r.tps, unit: 'TPS' }) : undefined,
				share,
				change: last && last > 0 ? value / last - 1 : undefined,
				details: (
					<dl className="mb-0 grid grid-cols-2 gap-x-4">
						<dt>{i18n.t('group.boards.talents')}</dt>
						<dd className="mb-0 font-mono">{e.talents || '-'}</dd>
						<dt>DPS / HPS / TPS</dt>
						<dd className="mb-0">{`${r.dps.toFixed(0)} / ${r.hps.toFixed(0)} / ${r.tps.toFixed(0)}`}</dd>
						<dt>DTPS</dt>
						<dd className="mb-0">{r.dtps.toFixed(0)}</dd>
						<dt>{i18n.t('group.results.columns.death')}</dt>
						<dd className="mb-0">{`${(r.chanceOfDeath * 100).toFixed(1)}%`}</dd>
						<dt>{i18n.t('group.results.columns.mana')}</dt>
						<dd className="mb-0">{`${r.manaSpent.toFixed(0)} (${r.secondsOom.toFixed(0)} s OOM)`}</dd>
						{warnings[e.key]?.map(w => (
							<dd key={w} className="col-span-2 mb-0 text-link-warning">
								{w}
							</dd>
						))}
					</dl>
				),
			};
		});
	};

	const counts = useMemo(
		() => ({
			dps: entries.filter(e => e.role === 'dps').length,
			healer: entries.filter(e => e.role === 'healer').length,
			tank: entries.filter(e => e.role === 'tank').length,
		}),
		[entries],
	);
	const activityName = i18n.t(`group.activities.${ACTIVITY_KEYS[activity]}`);

	return (
		<div className="flex flex-col gap-6" data-testid="class-comparison">
			<p className="mb-0 text-gray-400">{i18n.t('group.boards.intro', counts)}</p>
			<section className="flex flex-wrap items-end gap-4">
				<ChoiceField
					className="min-w-40"
					label={i18n.t('group.settings.activity')}
					value={activity}
					options={ACTIVITIES.map(a => ({ value: a, label: i18n.t(`group.activities.${ACTIVITY_KEYS[a]}`) }))}
					onChange={setActivity}
				/>
				<NumberField label={i18n.t('group.settings.level')} value={level} positive onChange={v => setLevel(Math.min(60, Math.max(1, v)))} />
				{activity === Activity.ActivityDungeon && (
					<ChoiceField
						className="min-w-72"
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
				<NumberField label={i18n.t('group.settings.iterations')} value={iterations} positive onChange={v => setIterations(Math.max(10, v))} />
				<Button variant="primary" loading={!!progress} disabled={!!progress || !entries.length} onClick={run}>
					{progress ? i18n.t('group.boards.running', progress) : i18n.t('group.boards.run')}
				</Button>
			</section>
			{error && <Notice level="error">{error}</Notice>}
			{failures.length > 0 && (
				<Notice level="warning" testId="board-failures">
					{i18n.t('group.boards.failed', { count: failures.length })}
					<ul className="mb-0">
						{failures.map(f => (
							<li key={f}>{f}</li>
						))}
					</ul>
				</Notice>
			)}
			<TierBoard
				testId="board-dps"
				title={i18n.t('group.boards.dps_title', { activity: activityName })}
				note={i18n.t('group.boards.dps_note')}
				rows={rowsFor('dps')}
			/>
			<TierBoard
				testId="board-healer"
				title={i18n.t('group.boards.healer_title', { activity: activityName })}
				note={i18n.t('group.boards.healer_note')}
				rows={rowsFor('healer')}
			/>
			<TierBoard
				testId="board-tank"
				title={i18n.t('group.boards.tank_title', { activity: activityName })}
				note={i18n.t('group.boards.tank_note')}
				controls={
					<ToggleGroup
						size="sm"
						value={tankRank}
						onChange={setTankRank}
						options={[
							{ value: 'tps', label: 'TPS' },
							{ value: 'dps', label: 'DPS' },
						]}
					/>
				}
				rows={rowsFor('tank')}
			/>
		</div>
	);
};
