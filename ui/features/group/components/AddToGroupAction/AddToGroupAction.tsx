import i18n from '@i18n/config';
import { useSimHost } from '@sim/context/SimHostContext';
import { PlayerSpecs } from '@sim/player/specs';
import { isIndividualSimHost } from '@sim/sim_host';
import { SidebarIconAction } from '@ui-kit/SidebarIconAction';

import { addToRoster } from '../../model/roster';

export interface AddToGroupActionProps {
	/** Opens the group page once the character is saved. */
	onAdded: () => void;
}

/**
 * Saves the page's character, as it would sim now, into the roster the group page picks from. Gear
 * planner pages (the healers) show it too, since a group needs its healer even where the page itself
 * cannot sim.
 */
export const AddToGroupAction = ({ onAdded }: AddToGroupActionProps) => {
	const host = useSimHost();
	if (!isIndividualSimHost(host) || host.disabled) return null;

	return (
		<SidebarIconAction
			icon="user"
			label={i18n.t('sidebar.buttons.add_to_group')}
			testId="add-to-group-action"
			onClick={() => {
				const spec = host.player.getPlayerSpec();
				addToRoster(host.sim.env.storage, PlayerSpecs.getFullSpecName(spec), spec.specID, host.player.toProto(false, true));
				onAdded();
			}}
		/>
	);
};
