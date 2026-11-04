import i18n from '@i18n/config';

import { type DuoKeys, keyFor, type KeyUse, type OpenerCast } from './keys_api';

const KeyCap = ({ k, small }: { k: KeyUse; small?: boolean }) => (
	<span
		className={`inline-grid shrink-0 place-items-center rounded-md border border-b-4 font-bold ${small ? 'h-8 min-w-14 px-2 text-sm' : 'h-12 min-w-18 px-3 text-lg'} ${
			k.main ? 'border-sky-600 bg-sky-950 text-sky-300' : 'border-gray-600 bg-gray-800 text-gray-100'
		}`}>
		{k.key}
	</span>
);

export interface CharacterKeys {
	name: string;
	/** Tailwind text colour for the class. */
	colour?: string;
	detail: string;
	keys?: DuoKeys;
}

/** Each character's keys: the one to mash, the ones for when something happens, and the rest on the bar. */
export const YourKeys = ({ characters }: { characters: CharacterKeys[] }) => (
	<section className="flex flex-col gap-2">
		<h2 className="mb-0 text-xl">{i18n.t('me.keys.title')}</h2>
		<p className="mb-0 text-sm text-gray-400">{i18n.t('me.keys.intro')}</p>
		<div className="grid gap-4 md:grid-cols-2">
			{characters.map(c => {
				const pressed = c.keys?.keys.filter(k => !k.extra) ?? [];
				const extra = c.keys?.keys.filter(k => k.extra && k.spells?.length) ?? [];
				// The mashed key first, then the rest as the sims check them.
				const ordered = [...pressed.filter(k => k.main), ...pressed.filter(k => !k.main)];
				return (
					<div key={c.name} className="rounded-lg border border-gray-700 bg-gray-900 p-4">
						<p className={`mb-0 text-lg font-bold ${c.colour ?? ''}`}>{c.name}</p>
						<p className="mb-2 text-sm text-gray-400">{c.detail}</p>
						{!c.keys && <p className="mb-0 text-gray-400">{i18n.t('me.keys.none')}</p>}
						{ordered.map(k => (
							<div key={k.key} className="flex items-start gap-3 border-t border-gray-700 py-2">
								<KeyCap k={k} />
								<div>
									<p className="mb-0 font-bold">{k.main ? i18n.t('me.keys.main') : i18n.t('me.keys.when', { when: k.when })}</p>
									<p className="mb-0 text-sm text-gray-400">{(k.spells ?? []).join(', ') || i18n.t('me.keys.nothing_yet')}</p>
								</div>
							</div>
						))}
						{extra.length > 0 && (
							<p className="mb-0 border-t border-gray-700 pt-2 text-sm text-gray-400">
								{i18n.t('me.keys.extra')}{' '}
								{extra.map((k, i) => (
									<span key={k.key}>
										{i > 0 && ' · '}
										<b className="text-gray-200">{k.key}</b> {(k.spells ?? []).join(', ')}
									</span>
								))}
							</p>
						)}
					</div>
				);
			})}
		</div>
	</section>
);

/** The first seconds of one simmed pull, one press a line: when, who, which key, and the spell it gave. */
export const Opener = ({ casts, keysOf }: { casts: OpenerCast[]; keysOf: (player: string) => DuoKeys | undefined }) => {
	const lines = casts.map(c => ({ ...c, key: keyFor(keysOf(c.player), c.spell) })).filter(c => c.key);
	return (
		<section className="flex flex-col gap-2">
			<h2 className="mb-0 text-xl">{i18n.t('me.opener.title')}</h2>
			<p className="mb-0 text-sm text-gray-400">{i18n.t('me.opener.intro')}</p>
			{!lines.length && <p className="mb-0 text-gray-400">{i18n.t('me.opener.none')}</p>}
			<ol className="mb-0 flex flex-col rounded-lg border border-gray-700 bg-gray-900 px-4 py-2">
				{lines.map((c, i) => (
					<li key={i} className="flex items-center gap-3 border-b border-gray-800 py-1 last:border-b-0">
						<span className="w-16 text-right text-sm text-gray-400 tabular-nums">
							{c.at < 0 ? i18n.t('me.opener.before') : i18n.t('me.opener.at', { at: c.at.toFixed(1) })}
						</span>
						<span className="w-20 truncate font-bold">{c.player}</span>
						<KeyCap k={c.key!} small />
						<span>{c.spell}</span>
					</li>
				))}
			</ol>
		</section>
	);
};
