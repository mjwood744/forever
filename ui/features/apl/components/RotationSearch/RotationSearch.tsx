import type { RotationSearchResult } from '@generated/proto/api';
import { APLRotation } from '@generated/proto/apl';
import i18n from '@i18n/config';
import { useSimHost } from '@sim/context/SimHostContext';
import { Accordion, AccordionItem } from '@ui-kit/Accordion';
import { Button } from '@ui-kit/Button';
import { ContentBlock } from '@ui-kit/ContentBlock';
import { Notice } from '@ui-kit/Notice';
import { useState } from 'react';

import { goalFor, goalUnit, searchRotation } from '../../model/rotation_search';

export interface RotationSearchViewProps {
	unit: string;
	running: boolean;
	disabled?: boolean;
	result?: Pick<RotationSearchResult, 'score' | 'startScore' | 'candidates' | 'changes' | 'steps'>;
	error?: string;
	// Shown once a better rotation replaced the player's; puts the old one back.
	onUndo?: () => void;
	onRun: () => void;
}

export const RotationSearchView = ({ unit, running, disabled, result, error, onUndo, onRun }: RotationSearchViewProps) => {
	const [open, setOpen] = useState<string[]>([]);
	const improved = !!result && result.changes.length > 0;
	const gain = result && result.startScore > 0 ? ((result.score / result.startScore - 1) * 100).toFixed(1) : undefined;
	const figures = result && {
		unit,
		from: result.startScore.toFixed(1),
		to: result.score.toFixed(1),
		count: result.candidates,
		gain,
	};
	return (
		<ContentBlock config={{ header: { title: i18n.t('rotation_tab.search.title') } }}>
			<div className="flex flex-col gap-4" data-testid="rotation-search">
				<p className="mb-0">{i18n.t('rotation_tab.search.description', { unit })}</p>
				<Button variant="primary" loading={running} disabled={running || disabled} onClick={onRun}>
					{i18n.t('rotation_tab.search.run')}
				</Button>
				{error && <Notice level="error">{error}</Notice>}
				{figures && (
					<div className="flex flex-col gap-4" data-testid="rotation-search-result">
						<div>
							<span className={`block text-lg font-bold ${improved ? 'text-success' : ''}`}>
								{improved
									? gain
										? i18n.t('rotation_tab.search.headline_gain', figures)
										: i18n.t('rotation_tab.search.headline_better', figures)
									: i18n.t('rotation_tab.search.headline_same')}
							</span>
							<span className="text-muted">
								{improved ? i18n.t('rotation_tab.search.score_improved', figures) : i18n.t('rotation_tab.search.score_same', figures)}
							</span>
						</div>
						{result.steps.length > 0 && (
							<div>
								<p className="mb-1">{i18n.t('rotation_tab.search.steps_intro')}</p>
								<ol className="mb-0 pl-6" data-testid="rotation-search-steps">
									{result.steps.map((step, i) => (
										<li key={i}>
											<span className="font-bold">{step.ability}</span>
											{step.conditional && <span className="text-muted"> {i18n.t('rotation_tab.search.conditional')}</span>}
										</li>
									))}
								</ol>
							</div>
						)}
						{improved && <p className="mb-0 text-muted">{i18n.t('rotation_tab.search.applied')}</p>}
						{onUndo && (
							<Button variant="outline-secondary" onClick={onUndo}>
								{i18n.t('rotation_tab.search.undo')}
							</Button>
						)}
						<Accordion value={open} onValueChange={setOpen} testId="rotation-search-details">
							<AccordionItem value="details" title={i18n.t('rotation_tab.search.details')}>
								<div className="flex flex-col gap-2">
									<p className="mb-0">{i18n.t('rotation_tab.search.details_count', figures)}</p>
									{improved && (
										<div>
											<p className="mb-1">{i18n.t('rotation_tab.search.details_changes')}</p>
											<ol className="mb-0 pl-6">
												{result.changes.map((c, i) => (
													<li key={i}>{c}</li>
												))}
											</ol>
										</div>
									)}
								</div>
							</AccordionItem>
						</Accordion>
					</div>
				)}
			</div>
		</ContentBlock>
	);
};

// Searches from the player's current rotation on the Go server and puts the rotation found in its place.
export const RotationSearch = () => {
	const host = useSimHost();
	const player = host.player;
	const goal = goalFor(player.playerSpec);
	const [running, setRunning] = useState(false);
	const [result, setResult] = useState<RotationSearchResult>();
	const [previous, setPrevious] = useState<APLRotation>();
	const [error, setError] = useState<string>();

	const run = async () => {
		setRunning(true);
		setError(undefined);
		setResult(undefined);
		setPrevious(undefined);
		try {
			const before = APLRotation.clone(player.aplRotation);
			const found = await searchRotation(host.sim.makeRaidSimRequest(), goal);
			if (found.error) throw new Error(found.error.message);
			if (found.rotation && found.changes.length) {
				player.setAplRotation(found.rotation);
				setPrevious(before);
			}
			setResult(found);
		} catch (e) {
			setError(e instanceof Error ? e.message : String(e));
		} finally {
			setRunning(false);
		}
	};

	const undo = () => {
		if (!previous) return;
		player.setAplRotation(previous);
		setPrevious(undefined);
		setResult(undefined);
	};

	return (
		<RotationSearchView
			unit={goalUnit(goal)}
			running={running}
			disabled={host.simDisabled}
			result={result}
			error={error}
			onUndo={previous ? undo : undefined}
			onRun={run}
		/>
	);
};
