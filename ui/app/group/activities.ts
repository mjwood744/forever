import { Activity } from '@generated/proto/api';

export const ACTIVITIES = [Activity.ActivityOpenWorld, Activity.ActivityDungeon, Activity.ActivityRaid];

/** Each activity's locale key under `group.activities`. */
export const ACTIVITY_KEYS: Record<Activity, string> = {
	[Activity.ActivityOpenWorld]: 'open_world',
	[Activity.ActivityDungeon]: 'dungeon',
	[Activity.ActivityRaid]: 'raid',
};
