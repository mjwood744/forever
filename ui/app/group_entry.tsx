import '../shared/page_boot';

import { PortalContainerContext } from '@ui-kit/hooks/usePortalContainer';
import { StrictMode } from 'react';
import { createRoot } from 'react-dom/client';

import { GroupPage } from './group/GroupPage';

const rootElem = document.getElementById('root');
if (!rootElem) throw new Error('No #root element on the page; ui/group/index.html should provide it.');

createRoot(rootElem).render(
	<StrictMode>
		<PortalContainerContext value={rootElem}>
			<GroupPage />
		</PortalContainerContext>
	</StrictMode>,
);
