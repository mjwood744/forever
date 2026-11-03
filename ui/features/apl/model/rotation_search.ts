import { type RaidSimRequest, RotationGoal, RotationSearchRequest, RotationSearchResult } from '@generated/proto/api';

// What a build is searched for: threat for a tank spec, healing for a healer, damage for the rest.
export const goalFor = (spec: { isTankSpec: boolean; isHealingSpec: boolean }): RotationGoal =>
	spec.isTankSpec ? RotationGoal.RotationGoalTps : spec.isHealingSpec ? RotationGoal.RotationGoalHps : RotationGoal.RotationGoalDps;

export const goalUnit = (goal: RotationGoal): string => (goal === RotationGoal.RotationGoalTps ? 'TPS' : goal === RotationGoal.RotationGoalHps ? 'HPS' : 'DPS');

// The Go server answers on the page's own origin; there is no wasm fallback, so the search needs the server.
export async function searchRotation(request: RaidSimRequest, goal: RotationGoal): Promise<RotationSearchResult> {
	const body = RotationSearchRequest.toBinary(RotationSearchRequest.create({ request, goal }));
	const response = await fetch('/rotationSearch', { method: 'POST', headers: { 'Content-Type': 'application/x-protobuf' }, body: body as BodyInit });
	if (!response.ok) throw new Error(`HTTP ${response.status} from /rotationSearch`);
	return RotationSearchResult.fromBinary(new Uint8Array(await response.arrayBuffer()));
}
