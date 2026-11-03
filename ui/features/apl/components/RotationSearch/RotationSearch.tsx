import type { RotationSearchResult } from '@generated/proto/api';
import i18n from '@i18n/config';
import { useSimHost } from '@sim/context/SimHostContext';
import { Button } from '@ui-kit/Button';
import { ContentBlock } from '@ui-kit/ContentBlock';
import { Notice } from '@ui-kit/Notice';
import { useState } from 'react';

import { goalFor, goalUnit, searchRotation } from '../../model/rotation_search';

export interface RotationSearchViewProps {
	unit: string;
	running: boolean;
	disabled?: boolean;
	result?: Pick<RotationSearchResult, 'score' | 'startScore' | 'candidates' | 'changes'>;
	error?: string;
	onRun: () => void;
}

export const RotationSearchView = ({ unit, running, disabled, result, error, onRun }: RotationSearchViewProps) => {
	const figures = result && { unit, from: result.startScore.toFixed(1), to: result.score.toFixed(1), count: result.candidates };
	return (
		<ContentBlock config={{ header: { title: i18n.t('rotation_tab.search.title') } }}>
			<div className="flex flex-col gap-4">
				<p className="mb-0">{i18n.t('rotation_tab.search.description', { unit })}</p>
				<Button variant="primary" loading={running} disabled={running || disabled} onClick={onRun}>
					{i18n.t('rotation_tab.search.run')}
				</Button>
				{error && <Notice level="error">{error}</Notice>}
				{figures && result.changes.length === 0 && <p className="mb-0">{i18n.t('rotation_tab.search.unchanged', figures)}</p>}
				{figures && result.changes.length > 0 && (
					<div>
						<p className="mb-0">{i18n.t('rotation_tab.search.improved', figures)}</p>
						<ol className="mb-0">
							{result.changes.map((c, i) => (
								<li key={i}>{c}</li>
							))}
						</ol>
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
	const [error, setError] = useState<string>();

	const run = async () => {
		setRunning(true);
		setError(undefined);
		setResult(undefined);
		try {
			const found = await searchRotation(host.sim.makeRaidSimRequest(), goal);
			if (found.error) throw new Error(found.error.message);
			if (found.rotation && found.changes.length) player.setAplRotation(found.rotation);
			setResult(found);
		} catch (e) {
			setError(e instanceof Error ? e.message : String(e));
		} finally {
			setRunning(false);
		}
	};

	return <RotationSearchView unit={goalUnit(goal)} running={running} disabled={host.simDisabled} result={result} error={error} onRun={run} />;
};
