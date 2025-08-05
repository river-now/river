import { apiHelper, submit } from "river.now/client";
import {
	ACTIONS_ROUTER_MOUNT_ROOT,
	type BaseMutationPropsWithInput,
	type BaseQueryPropsWithInput,
	type RiverMutationOutput,
	type RiverMutationPattern,
	type RiverQueryOutput,
	type RiverQueryPattern,
} from "./river.gen.ts";

export const api = { query, mutate };

async function query<P extends RiverQueryPattern>(
	props: BaseQueryPropsWithInput<P>,
) {
	const opts = apiHelper.toQueryOpts(ACTIONS_ROUTER_MOUNT_ROOT, props);
	return await submit<RiverQueryOutput<P>>(
		apiHelper.buildURL(opts),
		{
			method: apiHelper.resolveMethod(opts),
		},
		props.options,
	);
}

async function mutate<P extends RiverMutationPattern>(
	props: BaseMutationPropsWithInput<P>,
) {
	const opts = apiHelper.toMutationOpts(ACTIONS_ROUTER_MOUNT_ROOT, props);
	return await submit<RiverMutationOutput<P>>(
		apiHelper.buildURL(opts),
		{
			method: apiHelper.resolveMethod(opts),
			body: JSON.stringify(props.input),
		},
		props.options,
	);
}
