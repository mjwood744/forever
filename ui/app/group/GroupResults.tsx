import type { GroupRole } from '@features/group/model/roles';
import type { ActivitySimResult } from '@generated/proto/api';
import i18n from '@i18n/config';
import { Notice } from '@ui-kit/Notice';
import { Table, TableCell, TableHeaderCell, TableRow } from '@ui-kit/Table';

export interface GroupResultsProps {
	result: ActivitySimResult;
	/** Each player's role, in party order. */
	roles: ReadonlyArray<GroupRole>;
	/** Index of the player holding the mobs. */
	tank: number;
}

// A role's own figures stand out from the rest of its row.
const FIGURE = 'text-gray-400 data-[key-figure=true]:font-bold data-[key-figure=true]:text-white';

const fixed = (n: number, digits = 0) => n.toLocaleString(undefined, { minimumFractionDigits: digits, maximumFractionDigits: digits });
const percent = (n: number) => `${fixed(n * 100, 1)}%`;

// The figures each role is judged by: damage for damage dealers, healing and mana for healers, threat,
// damage taken and survival for tanks.
const KEY_COLUMNS: Record<GroupRole, ReadonlyArray<string>> = {
	dps: ['dps'],
	healer: ['hps', 'mana', 'oom'],
	tank: ['tps', 'dtps', 'death'],
};

export const GroupResults = ({ result, roles, tank }: GroupResultsProps) => {
	const tankTps = result.players[tank]?.tps ?? 0;
	const overTank = result.players.filter((p, i) => i !== tank && p.tps > tankTps).map(p => p.name);
	const isKey = (i: number, column: string) => KEY_COLUMNS[roles[i] ?? 'dps'].includes(column);

	return (
		<div className="flex flex-col gap-6" data-testid="group-results">
			<div className="flex flex-wrap gap-6 text-lg">
				<span>{i18n.t('group.results.group_dps', { value: fixed(result.groupDps) })}</span>
				<span>{i18n.t('group.results.total_time', { value: fixed(result.totalSeconds) })}</span>
			</div>
			{overTank.length > 0 && <Notice level="warning">{i18n.t('group.results.threat_warning', { names: overTank.join(', ') })}</Notice>}
			<Table striped dense testId="group-players">
				<thead>
					<TableRow>
						<TableHeaderCell>{i18n.t('group.results.columns.name')}</TableHeaderCell>
						<TableHeaderCell>{i18n.t('group.results.columns.role')}</TableHeaderCell>
						<TableHeaderCell>DPS</TableHeaderCell>
						<TableHeaderCell>HPS</TableHeaderCell>
						<TableHeaderCell>TPS</TableHeaderCell>
						<TableHeaderCell>DTPS</TableHeaderCell>
						<TableHeaderCell>{i18n.t('group.results.columns.death')}</TableHeaderCell>
						<TableHeaderCell>{i18n.t('group.results.columns.oom')}</TableHeaderCell>
						<TableHeaderCell>{i18n.t('group.results.columns.mana')}</TableHeaderCell>
					</TableRow>
				</thead>
				<tbody>
					{result.players.map((p, i) => (
						<TableRow key={p.name}>
							<TableCell>{p.name}</TableCell>
							<TableCell>{i18n.t(`group.roles.${roles[i] ?? 'dps'}`)}</TableCell>
							<TableCell className={FIGURE} data-key-figure={isKey(i, 'dps')}>
								{fixed(p.dps, 1)}
							</TableCell>
							<TableCell className={FIGURE} data-key-figure={isKey(i, 'hps')}>
								{fixed(p.hps, 1)}
							</TableCell>
							<TableCell className={FIGURE} data-key-figure={isKey(i, 'tps')}>
								{fixed(p.tps, 1)}
							</TableCell>
							<TableCell className={FIGURE} data-key-figure={isKey(i, 'dtps')}>
								{fixed(p.dtps, 1)}
							</TableCell>
							<TableCell className={FIGURE} data-key-figure={isKey(i, 'death')}>
								{percent(p.chanceOfDeath)}
							</TableCell>
							<TableCell className={FIGURE} data-key-figure={isKey(i, 'oom')}>
								{fixed(p.secondsOom, 1)}
							</TableCell>
							<TableCell className={FIGURE} data-key-figure={isKey(i, 'mana')}>
								{fixed(p.manaSpent)}
							</TableCell>
						</TableRow>
					))}
				</tbody>
			</Table>
			<Table striped dense testId="group-fights">
				<thead>
					<TableRow>
						<TableHeaderCell>{i18n.t('group.results.columns.fight')}</TableHeaderCell>
						<TableHeaderCell>{i18n.t('group.results.columns.targets')}</TableHeaderCell>
						<TableHeaderCell>{i18n.t('group.results.columns.seconds')}</TableHeaderCell>
						{result.players.map(p => (
							<TableHeaderCell key={p.name}>{p.name}</TableHeaderCell>
						))}
					</TableRow>
				</thead>
				<tbody>
					{result.fights.map((f, n) => (
						<TableRow key={`${n}-${f.label}`}>
							<TableCell className={f.boss ? 'font-bold' : undefined}>{f.label}</TableCell>
							<TableCell>{f.targetCount}</TableCell>
							<TableCell>{fixed(f.seconds, 1)}</TableCell>
							{f.players.map((p, i) => (
								<TableCell
									key={p.name}
									title={i18n.t('group.results.fight_cell', { dps: fixed(p.dps), hps: fixed(p.hps), death: percent(p.chanceOfDeath) })}>
									{roles[i] === 'healer' ? `${fixed(p.hps)} HPS` : roles[i] === 'tank' ? `${fixed(p.tps)} TPS` : `${fixed(p.dps)} DPS`}
									{p.chanceOfDeath > 0 && <span className="ms-1 text-danger">{percent(p.chanceOfDeath)}</span>}
								</TableCell>
							))}
						</TableRow>
					))}
				</tbody>
			</Table>
		</div>
	);
};
