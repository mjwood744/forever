import { MAX_GROUP_SIZE, readRoster, removeFromRoster, renameInRoster, type RosterEntry, rosterPlayer } from '@features/group/model/roster';
import { Activity, type ActivityDungeonInfo, type ActivitySimResult } from '@generated/proto/api';
import i18n from '@i18n/config';
import { PlayerSpecs } from '@sim/player/specs';
import { Button } from '@ui-kit/Button';
import { ChoiceField } from '@ui-kit/ChoiceField';
import { Notice } from '@ui-kit/Notice';
import { NumberField, StringField } from '@ui-kit/TextField';
import { useEffect, useMemo, useState } from 'react';

import { fetchDungeons, runActivity } from './activity_api';
import { GroupResults, type GroupRole } from './GroupResults';

const roleOf = (entry: RosterEntry): GroupRole => {
	const spec = PlayerSpecs.fromProto(entry.spec);
	return spec.isTankSpec ? 'tank' : spec.isHealingSpec ? 'healer' : 'dps';
};

const ACTIVITIES = [Activity.ActivityOpenWorld, Activity.ActivityDungeon, Activity.ActivityRaid];
const ACTIVITY_KEYS: Record<Activity, string> = {
	[Activity.ActivityOpenWorld]: 'open_world',
	[Activity.ActivityDungeon]: 'dungeon',
	[Activity.ActivityRaid]: 'raid',
};

export const GroupPage = () => {
	const [roster, setRoster] = useState(() => readRoster(window.localStorage));
	const [picked, setPicked] = useState<string[]>([]);
	const [tankId, setTankId] = useState<string>('');
	const [activity, setActivity] = useState(Activity.ActivityOpenWorld);
	const [level, setLevel] = useState(60);
	const [dungeons, setDungeons] = useState<ActivityDungeonInfo[]>([]);
	const [dungeon, setDungeon] = useState('');
	const [iterations, setIterations] = useState(1000);
	const [running, setRunning] = useState(false);
	const [error, setError] = useState('');
	const [ran, setRan] = useState<{ result: ActivitySimResult; roles: GroupRole[]; tank: number } | null>(null);

	// A character saved on a spec page in another tab shows up here without a reload.
	useEffect(() => {
		const onStorage = () => setRoster(readRoster(window.localStorage));
		window.addEventListener('storage', onStorage);
		window.addEventListener('focus', onStorage);
		return () => {
			window.removeEventListener('storage', onStorage);
			window.removeEventListener('focus', onStorage);
		};
	}, []);

	useEffect(() => {
		fetchDungeons(level)
			.then(ds => {
				setDungeons(ds);
				setDungeon(current => (ds.some(d => d.name === current) ? current : (ds[0]?.name ?? '')));
			})
			.catch(e => setError(String(e)));
	}, [level]);

	const group = useMemo(() => picked.map(id => roster.find(e => e.id === id)).filter((e): e is RosterEntry => !!e), [picked, roster]);
	// The chosen tank, else the first tank spec, else the first in the group.
	const tank = useMemo(() => {
		const chosen = group.findIndex(e => e.id === tankId);
		if (chosen >= 0) return chosen;
		return Math.max(
			0,
			group.findIndex(e => roleOf(e) === 'tank'),
		);
	}, [group, tankId]);

	const toggle = (id: string) =>
		setPicked(current => (current.includes(id) ? current.filter(p => p !== id) : current.length < MAX_GROUP_SIZE ? [...current, id] : current));

	const run = async () => {
		setRunning(true);
		setError('');
		try {
			const result = await runActivity({ players: group.map(rosterPlayer), tank, activity, level, dungeon, iterations });
			if (result.error) setError(result.error.message);
			else setRan({ result, roles: group.map(roleOf), tank });
		} catch (e) {
			setError(String(e));
		} finally {
			setRunning(false);
		}
	};

	return (
		<div className="mx-auto flex max-w-6xl flex-col gap-8 p-6 text-gray-200">
			<header className="flex flex-col gap-2">
				<h1 className="mb-0 text-3xl font-bold">{i18n.t('group.title')}</h1>
				<p className="mb-0 text-gray-400">{i18n.t('group.intro')}</p>
			</header>

			<section className="flex flex-col gap-2" data-testid="group-roster">
				<h2 className="mb-0 text-xl">{i18n.t('group.roster.title', { picked: group.length, max: MAX_GROUP_SIZE })}</h2>
				{!roster.length && <Notice level="info">{i18n.t('group.roster.empty')}</Notice>}
				{roster.map(entry => {
					const spec = PlayerSpecs.fromProto(entry.spec);
					const inGroup = picked.includes(entry.id);
					return (
						<div key={entry.id} className="flex flex-wrap items-center gap-2" data-in-group={inGroup}>
							<img src={spec.getIcon('small')} alt="" className="size-8" />
							<StringField
								className="w-56"
								value={entry.name}
								placeholder={i18n.t('group.roster.name')}
								onChange={name => {
									renameInRoster(window.localStorage, entry.id, name);
									setRoster(readRoster(window.localStorage));
								}}
							/>
							<span className="w-48 text-gray-400">{PlayerSpecs.getFullSpecName(spec)}</span>
							<Button
								variant={inGroup ? 'secondary' : 'primary'}
								size="sm"
								onClick={() => toggle(entry.id)}
								disabled={!inGroup && picked.length >= MAX_GROUP_SIZE}>
								{i18n.t(inGroup ? 'group.roster.leave' : 'group.roster.join')}
							</Button>
							<Button
								variant="link-danger"
								size="sm"
								onClick={() => {
									removeFromRoster(window.localStorage, entry.id);
									setPicked(current => current.filter(p => p !== entry.id));
									setRoster(readRoster(window.localStorage));
								}}>
								{i18n.t('group.roster.remove')}
							</Button>
						</div>
					);
				})}
			</section>

			<section className="flex flex-wrap items-end gap-4" data-testid="group-settings">
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
				{group.length > 0 && (
					<ChoiceField
						label={i18n.t('group.settings.tank')}
						value={group[tank]?.id ?? ''}
						options={group.map(e => ({ value: e.id, label: e.name }))}
						onChange={setTankId}
					/>
				)}
				<NumberField label={i18n.t('group.settings.iterations')} value={iterations} positive onChange={v => setIterations(Math.max(10, v))} />
				<Button
					variant="primary"
					loading={running}
					disabled={running || !group.length || (activity === Activity.ActivityDungeon && !dungeon)}
					onClick={run}>
					{i18n.t('group.settings.run')}
				</Button>
			</section>

			{error && <Notice level="error">{error}</Notice>}
			{ran && <GroupResults result={ran.result} roles={ran.roles} tank={ran.tank} />}
		</div>
	);
};
