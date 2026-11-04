import '../shared/page_boot';

import { PortalContainerContext } from '@ui-kit/hooks/usePortalContainer';
import { StrictMode } from 'react';
import { createRoot } from 'react-dom/client';

import { MePage } from './me/MePage';

const rootElem = document.getElementById('root');
if (!rootElem) throw new Error('No #root element on the page; ui/me/index.html should provide it.');

createRoot(rootElem).render(
	<StrictMode>
		<PortalContainerContext value={rootElem}>
			<MePage />
		</PortalContainerContext>
	</StrictMode>,
);
