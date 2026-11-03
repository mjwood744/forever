import type { Tier } from '@features/group/model/boards';
import i18n from '@i18n/config';
import { Button } from '@ui-kit/Button';
import { Icon } from '@ui-kit/Icon';
import { type ReactNode, useState } from 'react';

export interface TierRow {
	key: string;
	tier: Tier;
	label: string;
	/** "Experimental", "No gear": shown after the label. */
	tags: ReadonlyArray<string>;
	/** The class's colour token name, e.g. "mage". */
	classColor: string;
	/** "Mage · 14/0/32". */
	subtitle: string;
	icon: string;
	value: number;
	unit: string;
	/** A second figure beside the first, e.g. DPS on the tank board. */
	secondary?: { value: number; unit: string };
	/** Share of the top build's value. */
	share: number;
	/** Change since the last run, as a fraction; undefined without a last run. */
	change?: number;
	details: ReactNode;
}

export interface TierBoardProps {
	title: string;
	note: ReactNode;
	/** Controls shown in the header, e.g. the rank-by toggle. */
	controls?: ReactNode;
	rows: ReadonlyArray<TierRow>;
	testId?: string;
}

const TIERS: ReadonlyArray<Tier> = ['S', 'A', 'B', 'C'];
const TIER_COLORS: Record<Tier, string> = { S: 'rgb(255, 128, 0)', A: 'rgb(163, 53, 238)', B: 'rgb(0, 112, 221)', C: 'rgb(90, 90, 90)' };

const fmt = (n: number) => n.toLocaleString(undefined, { maximumFractionDigits: 0 });

const BoardRow = ({ row, rank }: { row: TierRow; rank: number }) => {
	const [open, setOpen] = useState(false);
	return (
		<div
			className="flex flex-col rounded-sm border border-gray-800 bg-gray-950 p-2"
			style={{ borderLeftColor: `var(--color-class-${row.classColor})`, borderLeftWidth: 3 }}
			data-testid="tier-row">
			<div className="flex items-center gap-2">
				<span className="w-6 text-right text-gray-500">{rank}</span>
				<img src={row.icon} alt="" className="size-8" />
				<div className="flex min-w-0 grow flex-col">
					<span className="truncate font-bold" style={{ color: `var(--color-class-${row.classColor})` }}>
						{row.label}
						{row.tags.map(tag => (
							<span key={tag} className="ms-2 text-xs font-normal text-gray-400">
								{tag}
							</span>
						))}
					</span>
					<span className="text-xs text-gray-400">{row.subtitle}</span>
				</div>
				<div className="flex flex-col items-end">
					<span className="font-bold text-white">
						{fmt(row.value)} <span className="text-xs font-normal text-gray-400">{row.unit}</span>
					</span>
					<span className="text-xs text-gray-400">
						{row.change !== undefined && Math.abs(row.change) >= 0.001 && (
							<span className={row.change > 0 ? 'me-2 text-success' : 'me-2 text-danger'}>
								{row.change > 0 ? '↑' : '↓'} {(Math.abs(row.change) * 100).toFixed(1)}%
							</span>
						)}
						{row.secondary ? `${fmt(row.secondary.value)} ${row.secondary.unit}` : `${Math.round(row.share * 100)}%`}
					</span>
				</div>
				<Button variant="unstyled" iconOnly aria-label={i18n.t('group.boards.details')} onClick={() => setOpen(o => !o)}>
					<Icon name={open ? 'chevron-down' : 'chevron-right'} />
				</Button>
			</div>
			{open && <div className="mt-2 border-t border-gray-800 pt-2 text-sm text-gray-300">{row.details}</div>}
		</div>
	);
};

/** Builds in S, A, B and C rows by their share of the best build's figure, best first. */
export const TierBoard = ({ title, note, controls, rows, testId }: TierBoardProps) => {
	const ranked = [...rows].sort((a, b) => b.value - a.value);
	return (
		<section className="flex flex-col rounded-sm border border-gray-800" data-testid={testId}>
			<header className="flex flex-wrap items-center justify-between gap-2 border-b border-gray-800 p-4">
				<h2 className="mb-0 text-xl font-bold">{title}</h2>
				{controls}
			</header>
			<div className="border-b border-gray-800 p-4 text-sm text-gray-400">{note}</div>
			{TIERS.map(tier => {
				const inTier = ranked.filter(r => r.tier === tier);
				if (!inTier.length) return null;
				return (
					<div key={tier} className="flex border-b border-gray-800">
						<div
							className="flex w-16 shrink-0 items-center justify-center text-4xl font-bold text-white"
							style={{ backgroundColor: TIER_COLORS[tier] }}>
							{tier}
						</div>
						<div className="grid grow grid-cols-1 gap-2 p-2 md:grid-cols-2 xl:grid-cols-3">
							{inTier.map(row => (
								<BoardRow key={row.key} row={row} rank={ranked.indexOf(row) + 1} />
							))}
						</div>
					</div>
				);
			})}
		</section>
	);
};
